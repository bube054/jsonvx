package jsonvx

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
)

var (
	ErrJSONSyntax          = errors.New("JSON syntax error")
	ErrJSONUnexpectedChar  = errors.New("unexpected character in JSON input")
	ErrJSONDuplicateKey    = errors.New("duplicate key in JSON object")
	ErrJSONNoContent       = errors.New("no meaningful content to parse")
	ErrJSONMultipleContent = errors.New("multiple JSON values")
)

type Parser struct {
	input  []byte
	config *ParserConfig
	tokens Tokens

	curToken  Token
	curPos    int
	peekToken Token
	peekPos   int
}

func NewParser(input []byte, config *ParserConfig) Parser {
	if config == nil {
		config = NewParserConfig()
	}

	p := Parser{
		input:  input,
		config: config,
	}

	return p
}

func (p *Parser) Parse() (JSON, error) {
	l := NewLexer(p.input, p.config)

	tokens := l.Tokens()
	chunks, err := tokens.Split()
	lastIndex := len(chunks) - 1
	// fmt.Println(tokens)
	// fmt.Println(chunks, err)

	if err != nil {

		if lastIndex > 0 && (errors.Is(err, ErrIllegalToken) || errors.Is(err, ErrUnexpectedToken)) {
			chunk := chunks[lastIndex]
			illegalToken := tokens[chunk[0]]
			return nil, WrapJSONUnexpectedCharError(illegalToken)
		}

	}

	if len(chunks) == 0 {
		return nil, ErrJSONNoContent
	}

	if lastIndex > 0 && len(chunks) != 1 {
		chunk := chunks[lastIndex]
		extraToken := tokens[chunk[0]]
		return nil, WrapJSONMultipleContentError(extraToken)
	}

	chunk := chunks[0]
	p.tokens = tokens[chunk[0] : chunk[1]+1]
	p.curPos = -1
	p.curToken = Token{Kind: EOF}
	p.peekPos = -1
	p.peekToken = Token{Kind: EOF}

	p.nextToken()
	p.nextToken()

	return p.parse()
}

func (p *Parser) parse() (JSON, error) {
	switch p.curToken.Kind {
	case NULL:
		return p.parseNull()
	case BOOLEAN:
		return p.parseBoolean()
	case STRING:
		return p.parseString()
	case NUMBER:
		return p.parseNumber()
	case LEFT_SQUARE_BRACE:
		return p.parseArray()
	case LEFT_CURLY_BRACE:
		return p.parseObject()
	case ILLEGAL:
		return p.parseIllegal()
	default:
		return p.parseDefault()
	}
}

func (p *Parser) parseDefault() (JSON, error) {
	// fmt.Println("prev Token", p.prevToken)
	// fmt.Println("current Token", p.curToken)
	// fmt.Println("peek Token", p.peekToken)
	return nil, WrapJSONUnexpectedCharError(p.curToken)
}

func (p *Parser) parseIllegal() (JSON, error) {
	return nil, WrapJSONUnexpectedCharError(p.curToken)
}

func (p *Parser) parseNull() (JSON, error) {
	return newNull(&p.tokens[p.curPos], p.nextToken), nil
}

func (p *Parser) parseBoolean() (JSON, error) {
	return newBoolean(&p.tokens[p.curPos], p.nextToken), nil
}

func (p *Parser) parseString() (JSON, error) {
	return newString(&p.tokens[p.curPos], p.nextToken), nil
}

func (p *Parser) parseNumber() (JSON, error) {
	return newNumber(&p.tokens[p.curPos], p.nextToken), nil
}

func (p *Parser) parseArray() (JSON, error) {
	items := []JSON{}
	p.nextToken()

	p.ignoreWhitespacesOrComments()

	for !p.expectCurToken(RIGHT_SQUARE_BRACE) {
		item, err := p.parse()
		if err != nil {
			// putArray(items)
			return nil, err
		}

		items = append(items, item)

		p.ignoreWhitespacesOrComments()

		hasComma := p.expectCurToken(COMMA)
		isClosingBracket := p.expectCurToken(RIGHT_SQUARE_BRACE)
		isNextClosingBracket := p.expectPeekToken(RIGHT_SQUARE_BRACE, true)

		isTrailingComma := hasComma && isNextClosingBracket
		isValidArrayEnd := isClosingBracket || isTrailingComma

		if !p.config.AllowTrailingCommaArray && isTrailingComma {
			return nil, WrapJSONSyntaxError(p.curToken)
		}

		if !isValidArrayEnd && !hasComma {
			return nil, WrapJSONSyntaxError(p.curToken)
		}

		if isValidArrayEnd {
			if isTrailingComma {
				p.nextToken()
				p.ignoreWhitespacesOrComments()
			}
			break
		}

		p.nextToken()
		p.ignoreWhitespacesOrComments()
	}

	return newArray(items, p.nextToken), nil
}

func (p *Parser) parseObject() (JSON, error) {
	properties := []KeyValue{}
	p.nextToken()

	p.ignoreWhitespacesOrComments()

	var seen map[string]int // decoded key -> its index in properties, created on the first key

	for !p.expectCurToken(RIGHT_CURLY_BRACE) {
		keyToken := p.curToken
		jsonKey, err := p.parse()
		if err != nil {
			return nil, err
		}

		keyString, ok := jsonKey.(*String)

		if !ok {
			return nil, WrapJSONSyntaxError(keyToken)
		}

		if !ok {
			return nil, WrapJSONSyntaxError(keyToken)
		}

		key := keyString.Token.Literal

		if keyString.Token.SubKind != IDENT && len(key) < 2 {
			return nil, WrapJSONSyntaxError(keyToken)
		}

		p.ignoreWhitespacesOrComments()

		hasColon := p.expectCurToken(COLON)

		if !hasColon {
			return nil, WrapJSONSyntaxError(p.curToken)
		}

		p.nextToken()

		p.ignoreWhitespacesOrComments()

		value, err := p.parse()
		if err != nil {
			return nil, err
		}

		p.ignoreWhitespacesOrComments()

		valueString, ok := value.(*String)

		if ok && valueString.Token.Kind == STRING && valueString.Token.SubKind == IDENT {
			return nil, WrapJSONSyntaxError(*valueString.Token)
		}

		// Keys are decoded up front so that sorting, QueryPath and ForEach all see
		// the real key (e.g. "caf\u00e9" is stored as "café").
		keySlice := key
		if keyString.Token.SubKind != IDENT {
			keySlice = []byte(unescapeString(stringContent(key)))
		}

		// A repeated key either fails (DisallowDuplicateKeys) or replaces the earlier
		// value, so the last one wins and each key appears only once in the object.
		// Removing duplicates here also means no equal keys ever reach the sort below.
		if seen == nil {
			seen = make(map[string]int)
		}

		if i, dup := seen[string(keySlice)]; dup {
			if p.config.DisallowDuplicateKeys {
				return nil, WrapJSONDuplicateKeyError(keyToken, keySlice)
			}

			properties[i].value = value
		} else {
			seen[string(keySlice)] = len(properties)
			properties = append(properties, KeyValue{key: keySlice, value: value})
		}

		hasComma := p.expectCurToken(COMMA)
		isClosingBracket := p.expectCurToken(RIGHT_CURLY_BRACE)
		isNextClosingBracket := p.expectPeekToken(RIGHT_CURLY_BRACE, true)

		isTrailingComma := hasComma && isNextClosingBracket
		isValidArrayEnd := isClosingBracket || isTrailingComma

		if !p.config.AllowTrailingCommaObject && isTrailingComma {
			return nil, WrapJSONSyntaxError(p.curToken)
		}

		if !isValidArrayEnd && !hasComma {
			return nil, WrapJSONSyntaxError(p.curToken)
		}

		if isValidArrayEnd {
			if isTrailingComma {
				p.nextToken()
				p.ignoreWhitespacesOrComments()
			}
			break
		}

		p.nextToken()
		p.ignoreWhitespacesOrComments()
	}

	sort.Slice(properties, func(i, j int) bool {
		return bytes.Compare(properties[i].key, properties[j].key) < 0
	})

	return newObject(properties, p.nextToken), nil
}

func (p *Parser) nextToken() {
	p.curToken = p.peekToken
	p.curPos = p.peekPos

	if p.peekPos+1 >= len(p.tokens) {
		p.peekToken = Token{Kind: EOF, SubKind: NONE}
		p.peekPos = len(p.tokens)
	} else {
		p.peekPos++
		p.peekToken = p.tokens[p.peekPos]
	}
}

func (p *Parser) expectCurToken(kind TokenKind) bool {
	if p.curToken.Kind == kind {
		return true
	} else {
		return false
	}
}

func (p *Parser) expectPeekToken(kind TokenKind, ignoreWhitespaceOrComments bool) bool {
	if p.peekToken.Kind == kind {
		return true
	}

	if !ignoreWhitespaceOrComments {
		return false
	}

	if p.peekToken.Kind != WHITESPACE && p.peekToken.Kind != COMMENT {
		return false
	}

	peekPos := p.peekPos + 1

	for peekPos < len(p.tokens) {
		peekToken := p.tokens[peekPos]

		if peekToken.Kind == kind {
			return true
		}

		if peekToken.Kind != WHITESPACE && peekToken.Kind != COMMENT {
			return false
		}

		peekPos++
	}

	return false
}

func (p *Parser) ignoreWhitespacesOrComments() {
	for p.expectCurToken(WHITESPACE) || p.expectCurToken(COMMENT) {
		p.nextToken()
	}
}

func WrapUnexpectedCharError(baseErr error, token Token) error {
	return fmt.Errorf("%w: %q at line %d, column %d", baseErr, token.Literal, token.Line, token.Column)
}
func WrapJSONUnexpectedCharError(token Token) error {
	return WrapUnexpectedCharError(ErrJSONUnexpectedChar, token)
}

func WrapJSONSyntaxError(token Token) error {
	return WrapUnexpectedCharError(ErrJSONSyntax, token)
}

func WrapJSONMultipleContentError(token Token) error {
	return fmt.Errorf("%w: extra value %q at line %d, column %d",
		ErrJSONMultipleContent,
		token.Literal,
		token.Line,
		token.Column,
	)
}

func WrapJSONDuplicateKeyError(token Token, key []byte) error {
	return fmt.Errorf("%w: %q at line %d, column %d",
		ErrJSONDuplicateKey,
		key,
		token.Line,
		token.Column,
	)
}

// var (
// 	DefaultArrayCap  = 8
// 	DefaultObjectCap = 8
// )

// var arrayPool = sync.Pool{
// 	New: func() any {
// 		array := make([]JSON, 0, DefaultArrayCap)
// 		return &array
// 	},
// }

// func getArray() *[]JSON {
// 	return arrayPool.Get().(*[]JSON)
// }

// func putArray(a *[]JSON) {
// 	*a = (*a)[:0]
// 	arrayPool.Put(a)
// }

// var objectPool = sync.Pool{
// 	New: func() any {
// 		object := make([]KeyValue, 0, DefaultObjectCap)
// 		return object
// 	},
// }

// func getObject() []KeyValue {
// 	return objectPool.Get().([]KeyValue)
// }

// func putObject(m []KeyValue) {
// 	m = (m)[:0]
// 	objectPool.Put(&m)
// }

// SyntaxError: JSON.parse: unterminated string literal
// SyntaxError: JSON.parse: bad control character in string literal
// SyntaxError: JSON.parse: bad character in string literal
// SyntaxError: JSON.parse: bad Unicode escape
// SyntaxError: JSON.parse: bad escape character
// SyntaxError: JSON.parse: unterminated string
// SyntaxError: JSON.parse: no number after minus sign
// SyntaxError: JSON.parse: unexpected non-digit
// SyntaxError: JSON.parse: missing digits after decimal point
// SyntaxError: JSON.parse: unterminated fractional number/
// SyntaxError: JSON.parse: missing digits after exponent indicator
// SyntaxError: JSON.parse: missing digits after exponent sign
// SyntaxError: JSON.parse: exponent part is missing a number
// SyntaxError: JSON.parse: unexpected end of data
// SyntaxError: JSON.parse: unexpected keyword
// SyntaxError: JSON.parse: unexpected character
// SyntaxError: JSON.parse: end of data while reading object contents
// SyntaxError: JSON.parse: expected property name or '}'
// SyntaxError: JSON.parse: end of data when ',' or ']' was expected
// SyntaxError: JSON.parse: expected ',' or ']' after array element
// SyntaxError: JSON.parse: end of data when property name was expected
// SyntaxError: JSON.parse: expected double-quoted property name
// SyntaxError: JSON.parse: end of data after property name when ':' was expected
// SyntaxError: JSON.parse: expected ':' after property name in object
// SyntaxError: JSON.parse: end of data after property value in object
// SyntaxError: JSON.parse: expected ',' or '}' after property value in object
// SyntaxError: JSON.parse: expected ',' or '}' after property-value pair in object literal
// SyntaxError: JSON.parse: property names must be double-quoted strings
// SyntaxError: JSON.parse: expected property name or '}'
// SyntaxError: JSON.parse: unexpected character
// SyntaxError: JSON.parse: unexpected non-whitespace character after JSON data

// ✅ Option 1: 123+4i or 123-4i
// json
// Copy
// Edit
// "3+4i"
// "3-4i"
// Similar to: Mathematical notation.

// Pros: Familiar to humans; used in many languages (e.g., MATLAB, Python's str() for complex).

// Cons: Needs parsing logic to extract real/imag parts.

// ✅ Option 2: complex(3,4)
// json
// Copy
// Edit
// "complex(3,4)"
// Similar to: Python's constructor syntax.

// Pros: Explicit about structure; easy to parse.

// Cons: Verbose; less “number-like”.

// 🌀 Less conventional options:
// Option 3: 3@4
// json
// Copy
// Edit
// "3@4"
// Interpreted as: real@imag.

// Pros: Short and parseable.

// Cons: Not standard anywhere; might be confusing.

// Option 4: 3+4j
// json
// Copy
// Edit
// "3+4j"
// Used by: Python.

// Pros: Already in use.

// Cons: j instead of i might be less intuitive for non-engineers.

// 🔥 My recommendation
// If JSON followed existing number patterns and you wanted the cleanest textual form, it would likely look like:

// json
// Copy
// Edit
// "3+4i"   // for 3 + 4i
// "5-2i"   // for 5 - 2i
// "0+1i"   // for pure imaginary
// "4+0i"   // for pure real (still formatted as complex)
