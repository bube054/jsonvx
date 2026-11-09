package jsonvx

import (
	"errors"
	"fmt"
)

var (
	ErrJSONSyntax          = errors.New("JSON syntax error")
	ErrJSONUnexpectedChar  = errors.New("unexpected character in JSON input")
	ErrJSONDuplicateKey    = errors.New("duplicate key in JSON object")
	ErrJSONNoContent       = errors.New("no meaningful content to parse")
	ErrJSONMultipleContent = errors.New("multiple JSON values")
	ErrJSONTooDeep         = errors.New("JSON nesting too deep")
)

type Parser struct {
	input  []byte
	config *ParserConfig
	tokens Tokens

	curToken  Token
	curPos    int
	peekToken Token
	peekPos   int

	depth int // current nesting depth of arrays and objects
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

	// Whitespace tokens are left out: the parser skips them anyway, and in indented
	// JSON they are most of the tokens. Storing them made parsing several times slower
	// and kept them in memory for as long as the parsed tree is in use.
	tokens := l.TokensWithout(WHITESPACE)
	chunks, err := tokens.Split()
	lastIndex := len(chunks) - 1

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
	p.depth = 0

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

// enterNested records that the parser is going one level deeper into an array or object.
// It returns ErrJSONTooDeep once the configured MaxDepth is exceeded, so deeply nested
// input fails with an error instead of overflowing the stack.
func (p *Parser) enterNested() error {
	p.depth++

	if p.depth > p.config.maxDepth() {
		return WrapJSONTooDeepError(p.curToken, p.config.maxDepth())
	}

	return nil
}

func (p *Parser) parseArray() (JSON, error) {
	if err := p.enterNested(); err != nil {
		return nil, err
	}
	defer func() { p.depth-- }()

	items := []JSON{}
	p.nextToken()

	p.ignoreWhitespacesOrComments()

	for !p.expectCurToken(RIGHT_SQUARE_BRACE) {
		item, err := p.parse()
		if err != nil {
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
	if err := p.enterNested(); err != nil {
		return nil, err
	}
	defer func() { p.depth-- }()

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
		// The key keeps the position where it first appeared, as in JavaScript.
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

	// Properties stay in the order they were written. The table built above for
	// duplicate detection becomes the object's lookup index, so QueryPath can jump
	// straight to a key without the keys having to be sorted.
	return newObject(properties, seen, p.nextToken), nil
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

func WrapJSONTooDeepError(token Token, maxDepth int) error {
	return fmt.Errorf("%w: %q exceeds maximum depth of %d at line %d, column %d",
		ErrJSONTooDeep,
		token.Literal,
		maxDepth,
		token.Line,
		token.Column,
	)
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
