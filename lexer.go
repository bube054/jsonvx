package jsonvx

import (
	"bytes"
	"fmt"
)

// Lexer performs lexical analysis on a JSON-like input stream.
// It breaks the raw input into a sequence of tokens according to the parsing rules
// defined in the provided ParserConfig.
//
// The Lexer operates on raw byte slices for performance, and tracks position and line
// information to support error reporting and debugging.
type Lexer struct {
	input  []byte        // input is the raw byte slice being tokenized.
	config *ParserConfig // config holds the parsing options that control how the lexer interprets input.

	line       int // line tracks the current line number in the input (starting from 1).
	column     int
	lastColumn int
	pos        int // pos is the current character offset in the input (starting from 0).
	readPos    int // char is the current character under examination.

	char byte // readPos is the next position to be read.
}

// NewLexer creates a new Lexer instance using the given input and configuration.
//
// It immediately reads the first character to initialize internal state, so the lexer is ready
// for tokenization right after creation.
func NewLexer(input []byte, cfg *ParserConfig) *Lexer {
	l := &Lexer{input: input, config: cfg, line: 1, column: 0}
	l.readChar()
	return l
}

// String returns a human-readable representation of the current state of the Lexer.
// It includes the input, current line number, character position, read position, and the current character.
func (l *Lexer) String() string {
	return fmt.Sprintf(
		"Lexer{\n  input: %q,\n  line: %d,\n  pos: %d,\n  readPos: %d,\n  char: %q\n}",
		string(l.input), l.line, l.pos, l.readPos, l.char,
	)
}

// Token returns the current token being parsed by the lexer.
func (l *Lexer) Token() Token {
	if l.config == nil {
		l.config = NewParserConfig()
	}

	pos := l.pos
	col := l.column
	char := l.char
	nextChar := l.peek()

	switch char {

	// Lexing white space starts here
	case
		' ',  // space (U+0020)
		'\n', // line feed (U+000A)
		'\r', // carriage return (U+000D)
		'\t': // horizontal tab (U+0009)
		return newToken(WHITESPACE, NONE, l.input[l.pos:l.readPos], l.line, col, l.readChar)

	case
		'\v',     // vertical tab (U+000B)
		'\f',     // form feed (U+000C)
		'\u0085', // next line (NEL, U+0085)
		'\u00A0': // no-break space (U+00A0)
		if l.config.AllowExtraWS {
			return newToken(WHITESPACE, NONE, l.input[l.pos:l.readPos], l.line, col, l.readChar)
		} else {
			return newToken(ILLEGAL, INVALID_WHITESPACE, l.input[l.pos:], l.line, col, nil)
		}
		// Lexing white space ends here

	// Lexing left curly brace starts here
	case '{':
		return newToken(LEFT_CURLY_BRACE, NONE, l.input[pos:l.readPos], l.line, col, l.readChar)
	// Lexing left curly brace ends here

	// Lexing right brace starts here
	case '}':
		return newToken(RIGHT_CURLY_BRACE, NONE, l.input[pos:l.readPos], l.line, col, l.readChar)
	// Lexing right brace ends here

	// Lexing left square brace starts here
	case '[':
		return newToken(LEFT_SQUARE_BRACE, NONE, l.input[pos:l.readPos], l.line, col, l.readChar)
	// Lexing left square brace ends here

	// Lexing right square brace starts here
	case ']':
		return newToken(RIGHT_SQUARE_BRACE, NONE, l.input[pos:l.readPos], l.line, col, l.readChar)
		// Lexing right square brace ends here

	// Lexing right square brace starts here
	case ',':
		return newToken(COMMA, NONE, l.input[pos:l.readPos], l.line, col, l.readChar)
		// Lexing right square brace ends here

	// Lexing colon starts here
	case ':':
		return newToken(COLON, NONE, l.input[pos:l.readPos], l.line, col, l.readChar)
		// Lexing colon ends here

	// Lexing comment starts here
	case '/': // forward slash

		switch nextChar {
		case '/':
			if !l.config.AllowLineComments {
				return newToken(ILLEGAL, INVALID_LINE_COMMENT, l.input[l.pos:], l.line, col, nil)
			}

			for !isNewLine(l.char) && l.char != 0 {
				l.readChar()
			}

			if l.char == 0 {
				return newToken(COMMENT, LINE_COMMENT, l.input[pos:], l.line, col, nil)
			}

			return newToken(COMMENT, LINE_COMMENT, l.input[pos:l.readPos], l.line, col, l.readChar)

		case '*': // asterisk
			if !l.config.AllowBlockComments {
				return newToken(ILLEGAL, INVALID_BLOCK_COMMENT, l.input[l.pos:], l.line, col, nil)
			}

			for !(l.char == 47 && l.prev() == 42) && l.char != 0 {
				l.readChar()
			}

			if l.char == 0 {
				return newToken(ILLEGAL, INVALID_BLOCK_COMMENT, l.input[pos:], l.line, col, nil)
			}

			return newToken(COMMENT, BLOCK_COMMENT, l.input[pos:l.readPos], l.line, col, l.readChar)
		default:
			return newToken(ILLEGAL, INVALID_COMMENT, l.input[l.pos:l.readPos], l.line, col, nil)
		}
		// Lexing comment ends here

	// Lexing string starts here
	case '"', '\'': // double or single quote
		if l.char == '\'' && !l.config.AllowSingleQuotes {
			return newToken(ILLEGAL, INVALID_STRING, l.input[l.pos:], l.line, col, nil)
		}

		l.readChar()

		for {
			if l.char == 0 {
				return newToken(ILLEGAL, INVALID_STRING, l.input[pos:], l.line, col, nil)
			}

			// An unescaped closing quote ends the string. Escaped quotes never reach
			// this check, because escape sequences are consumed whole below.
			if l.char == char {
				if char == '\'' {
					return newToken(STRING, SINGLE_QUOTED, l.input[pos:l.readPos], l.line, col, l.readChar)
				}

				return newToken(STRING, DOUBLE_QUOTED, l.input[pos:l.readPos], l.line, col, l.readChar)
			}

			// Handle escape sequences: validate the character after the backslash,
			// then consume the backslash so that character is skipped as a unit.
			// This keeps runs like \\\" correct (escaped backslash, escaped quote).
			if l.char == '\\' {
				next := l.peek()

				switch next {
				case char: // escaped quote
				case '\\': // escaped backslash
				case '/': // escaped forward slash
				case 'b': // backspace
				case 'f': // form feed
				case 'n': // newline
				case 'r': // carriage return
				case 't': // tab
				case 'u': // unicode escape
					second := l.peekBy(2)
					third := l.peekBy(3)
					fourth := l.peekBy(4)
					fifth := l.peekBy(5)

					if !is4HexDigits([4]byte{second, third, fourth, fifth}) {
						return newToken(ILLEGAL, INVALID_HEX_STRING, l.input[pos:], l.line, col, nil)
					}

				case '\n': // escaped newline
					if !l.config.AllowNewlineInStrings {
						return newToken(ILLEGAL, INVALID_NEWLINE_STRING, l.input[pos:], l.line, col, nil)
					}

				default:
					if !l.config.AllowOtherEscapeChars {
						return newToken(ILLEGAL, INVALID_ESCAPED_STRING, l.input[pos:], l.line, col, nil)
					}
				}

				l.readChar() // consume the backslash; the escaped character is consumed below
			}

			l.readChar()
		}
	// Lexing string ends here

	case 0:
		return newToken(EOF, NONE, nil, l.line, col, nil)

	default:

		switch char {

		// Lexing null starts here
		case 'n':
			second := l.peekBy(1) // expect 'u'
			third := l.peekBy(2)  // expect 'l'
			fourth := l.peekBy(3) // expect 'l'

			if second == 'u' && third == 'l' && fourth == 'l' {
				l.readChar()
				l.readChar()
				l.readChar()

				return newToken(NULL, NONE, l.input[pos:l.readPos], l.line, col, l.readChar)
			}

			// Lexing null ends here

		// Lexing true starts here
		case 't':
			second := l.peekBy(1) // expect 'r'
			third := l.peekBy(2)  // expect 'u'
			fourth := l.peekBy(3) // expect 'e'

			if second == 'r' && third == 'u' && fourth == 'e' {
				l.readChar()
				l.readChar()
				l.readChar()

				return newToken(BOOLEAN, TRUE, l.input[pos:l.readPos], l.line, col, l.readChar)
			}

			// Lexing true ends here

		// Lexing false starts here
		case 'f':

			second := l.peekBy(1) // expect 'a'
			third := l.peekBy(2)  // expect 'l'
			fourth := l.peekBy(3) // expect 's'
			fifth := l.peekBy(4)  // expect 'e'

			if second == 'a' && third == 'l' && fourth == 's' && fifth == 'e' {
				l.readChar()
				l.readChar()
				l.readChar()
				l.readChar()

				return newToken(BOOLEAN, FALSE, l.input[pos:l.readPos], l.line, col, l.readChar)
			}

			// Lexing false ends here
		}

		// Lexing number starts here
		count := 0
		if isPossibleNumber(l.char) {
			l.readChar()
			count++

			for isPossibleNumber(l.char) {
				l.readChar()
				count++
			}

			num := l.input[pos:l.pos]

			numberParts := bytes.Split(num, []byte("."))
			integerPart := numberParts[0]

			hasLeadingZero := len(integerPart) > 1 && integerPart[0] == '0'
			hasHexPrefix := len(integerPart) > 1 && (integerPart[1] == 'x' || integerPart[1] == 'X')

			// if not does not start with 0X or 0x
			if hasLeadingZero && !hasHexPrefix {
				return newToken(ILLEGAL, INVALID_LEADING_ZERO, l.input[pos:], l.line, col, nil)
			}

			if startsWithPlus(num) && !l.config.AllowLeadingPlus {
				return newToken(ILLEGAL, INVALID_LEADING_PLUS, l.input[pos:], l.line, col, nil)
			}

			isNaNum := isNaN(num)

			if isNaNum && !l.config.AllowNaN {
				return newToken(ILLEGAL, INVALID_NaN, l.input[pos:], l.line, col, nil)
			}

			if isNaNum {
				return newToken(NUMBER, NaN, num, l.line, col, nil)
			}

			isInfinity := isInf(num)

			if isInfinity && !l.config.AllowInfinity {
				return newToken(ILLEGAL, INVALID_INF, l.input[pos:], l.line, col, nil)
			}

			if isInfinity {
				return newToken(NUMBER, INF, num, l.line, col, nil)
			}

			if isInteger(num) {
				return newToken(NUMBER, INTEGER, num, l.line, col, nil)
			}

			if startsOrEndsWithDot(num) && !l.config.AllowPointEdgeNumbers {
				return newToken(ILLEGAL, INVALID_POINT_EDGE_DOT, l.input[pos:], l.line, col, nil)
			}

			if isFloat(num) {
				return newToken(NUMBER, FLOAT, num, l.line, col, nil)
			}

			if isScientificNotation(num) {
				return newToken(NUMBER, SCI_NOT, num, l.line, col, nil)
			}

			isHexDec := isHex(num)
			if isHexDec && !l.config.AllowHexNumbers {
				return newToken(ILLEGAL, INVALID_HEX_NUMBER, l.input[pos:], l.line, col, nil)
			}

			if isHexDec {
				return newToken(NUMBER, HEX, num, l.line, col, nil)
			}

		}

		for range count {
			l.unReadChar()
		}

		// Lexing number ends here

		// Lexing ident starts here
		if isPossibleJSIdentifier(l.char) {
			l.readChar()

			for isPossibleJSIdentifier(l.char) {
				l.readChar()
			}

			if l.config.AllowUnquoted && !startsWithDigit(l.input[pos:l.pos]) {
				return newToken(STRING, IDENT, l.input[pos:l.pos], l.line, col, nil)
			}
		}
		// Lexing ident ends here

		return newToken(ILLEGAL, INVALID_CHARACTER, l.input[pos:], l.line, col, nil)
	}
}

// Tokens returns a slice of all tokens produced so far by the lexer.
func (l *Lexer) Tokens() Tokens {
	tokens := []Token{}

	for {
		token := l.Token()

		tokens = append(tokens, token)

		if token.Kind == EOF || token.Kind == ILLEGAL {
			break
		}
	}

	return tokens
}

// TokensWithoutWhitespace returns a slice of all tokens without whitespace produced so far by the lexer.
func (l *Lexer) TokensWithout(kind TokenKind) Tokens {
	tokens := []Token{}

	for {
		token := l.Token()

		if token.Kind != kind {
			tokens = append(tokens, token)
		}

		if token.Kind == EOF || token.Kind == ILLEGAL {
			break
		}
	}

	return tokens
}

// readChar advances the lexer to the next character in the input,
// updating the current character, position, and line counters as needed.
func (l *Lexer) readChar() {
	if l.readPos > len(l.input)-1 {
		l.char = 0
	} else {
		l.char = l.input[l.readPos]
	}

	l.pos = l.readPos
	l.readPos++

	if l.char == '\n' {
		l.line++
		l.lastColumn = l.column
		l.column = 0
	} else {
		l.column++
	}
}

// unreadChar moves the lexer back by one character in the input,
// updating the current character, position, and line counters as needed.
func (l *Lexer) unReadChar() {
	if l.pos == 0 {
		return
	}

	l.readPos = l.pos
	l.pos--

	l.char = l.input[l.pos]

	if l.char == '\n' {
		l.line--
		l.column = l.lastColumn
	} else {
		l.column--
	}
}

// peek returns the next character in the input without advancing the lexer.
// If the end of input is reached, it returns 0.
func (l *Lexer) peek() byte {
	if l.readPos > (len(l.input) - 1) {
		return 0
	} else {
		return l.input[l.readPos]
	}
}

// peekBy returns the character at a position `target` bytes ahead in the input
// without moving the current position. Returns 0 if out of bounds.
func (l *Lexer) peekBy(target int) byte {
	pos := l.pos + target

	if pos > (len(l.input) - 1) {
		return 0
	} else {
		return l.input[pos]
	}
}

// prev returns the character immediately before the current position.
// Returns 0 if the current position is at the start.
func (l *Lexer) prev() byte {
	if (l.pos - 1) < 0 {
		return 0
	} else {
		return l.input[l.pos-1]
	}
}

// prevBy returns the character `target` bytes before the current position.
// Returns 0 if the target is out of bounds.
func (l *Lexer) prevBy(target int) byte {
	pos := l.pos - target

	if pos < 0 {
		return 0
	} else {
		return l.input[pos]
	}
}
