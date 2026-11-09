package jsonvx

import (
	"bytes"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrIllegalToken             = errors.New("illegal token encountered")
	ErrUnexpectedToken          = errors.New("unexpected token encountered")
	ErrUnbalancedArrayBrackets  = errors.New("unbalanced brackets in array")
	ErrUnbalancedObjectBrackets = errors.New("unbalanced brackets in object")
)

// TokenSubKind represents the subtype of a token, such as numeric format or identifier type.
type TokenSubKind int

const (
	NONE TokenSubKind = iota // NONE represents the absence of a sub kind.

	FALSE // FALSE represents a boolean false.
	TRUE  // TRUE represents a boolean true.

	SINGLE_QUOTED // SINGLE_QUOTED represents a single quoted string value.
	DOUBLE_QUOTED // DOUBLE_QUOTED represents a double quoted string value.
	IDENT         // IDENT represents an unquoted identifier.

	INTEGER // INTEGER represents an integer (positive or negative).
	FLOAT   // FLOAT represents a floating-point number (positive or negative).
	SCI_NOT // SCI_NOT represents scientific notation (e.g., 1e10).
	HEX     // HEX represents a hexadecimal number (e.g., 0xFF).
	INF     // INF represents an Infinity literal (positive or negative).
	NaN     // NaN represents a Not-a-Number literal.

	LINE_COMMENT  // LINE_COMMENT represents a single-line comment (// ...).
	BLOCK_COMMENT // BLOCK_COMMENT represents a block comment (/* ... */).

	INVALID_CHARACTER      // INVALID_CHARACTER represents an invalid or unexpected character.
	INVALID_WHITESPACE     // INVALID_WHITESPACE represents an invalid or misplaced whitespace.
	INVALID_NULL           // INVALID_NULL represents an invalid 'null' literal.
	INVALID_TRUE           // INVALID_TRUE represents an invalid 'true' literal.
	INVALID_FALSE          // INVALID_FALSE represents an invalid 'false' literal.
	INVALID_COMMENT        // INVALID_COMMENT represents an improperly formed comment.
	INVALID_LINE_COMMENT   // INVALID_LINE_COMMENT represents a malformed single-line comment.
	INVALID_BLOCK_COMMENT  // INVALID_BLOCK_COMMENT represents a malformed block comment.
	INVALID_STRING         // INVALID_STRING represents a malformed or unterminated string.
	INVALID_HEX_STRING     // INVALID_HEX_STRING represents an invalid hexadecimal string.
	INVALID_NEWLINE_STRING // INVALID_NEWLINE_STRING represents a string that contains an invalid newline.
	INVALID_ESCAPED_STRING // INVALID_ESCAPED_STRING represents a string with an invalid escape sequence.
	INVALID_LEADING_ZERO   // INVALID_LEADING_ZERO represents a number with an invalid leading zero.
	INVALID_LEADING_PLUS   // INVALID_LEADING_PLUS represents a number with an invalid leading plus sign.
	INVALID_NaN            // INVALID_NaN represents a malformed NaN literal.
	INVALID_INF            // INVALID_INF represents a malformed Infinity literal.
	INVALID_POINT_EDGE_DOT // INVALID_POINT_EDGE_DOT represents a number with a misplaced or standalone dot.
	INVALID_HEX_NUMBER     // INVALID_HEX_NUMBER represents a malformed hexadecimal number.
)

// String returns a string representation of the TokenSubKind.
func (t TokenSubKind) String() string {
	m := map[TokenSubKind]string{
		NONE:                   "NONE",
		FALSE:                  "FALSE",
		TRUE:                   "TRUE",
		SINGLE_QUOTED:          "SINGLE_QUOTED",
		DOUBLE_QUOTED:          "DOUBLE_QUOTED",
		IDENT:                  "IDENT",
		INTEGER:                "INTEGER",
		FLOAT:                  "FLOAT",
		SCI_NOT:                "SCI_NOT",
		HEX:                    "HEX",
		INF:                    "INF",
		NaN:                    "NaN",
		LINE_COMMENT:           "LINE_COMMENT",
		BLOCK_COMMENT:          "BLOCK_COMMENT",
		INVALID_CHARACTER:      "INVALID_CHARACTER",
		INVALID_WHITESPACE:     "INVALID_WHITESPACE",
		INVALID_NULL:           "INVALID_NULL",
		INVALID_TRUE:           "INVALID_TRUE",
		INVALID_FALSE:          "INVALID_FALSE",
		INVALID_COMMENT:        "INVALID_COMMENT",
		INVALID_LINE_COMMENT:   "INVALID_LINE_COMMENT",
		INVALID_BLOCK_COMMENT:  "INVALID_BLOCK_COMMENT",
		INVALID_STRING:         "INVALID_STRING",
		INVALID_HEX_STRING:     "INVALID_HEX_STRING",
		INVALID_NEWLINE_STRING: "INVALID_NEWLINE_STRING",
		INVALID_ESCAPED_STRING: "INVALID_ESCAPED_STRING",
		INVALID_LEADING_ZERO:   "INVALID_LEADING_ZERO",
		INVALID_LEADING_PLUS:   "INVALID_LEADING_PLUS",
		INVALID_NaN:            "INVALID_NaN",
		INVALID_INF:            "INVALID_INF",
		INVALID_POINT_EDGE_DOT: "INVALID_POINT_EDGE_DOT",
		INVALID_HEX_NUMBER:     "INVALID_HEX_NUMBER",
	}

	if str, ok := m[t]; ok {
		return str
	}
	return "UNKNOWN"
}

// TokenKind represents the kind of a token, used during both lexical analysis
// and by tools that query parsed node structures.
type TokenKind int

const (
	EOF                TokenKind = iota // EOF indicates the end of input.
	ILLEGAL                             // ILLEGAL indicates an unrecognized or invalid token.
	WHITESPACE                          // WHITESPACE represents any space character.
	COMMENT                             // COMMENT represents a comment
	STRING                              // STRING represents a string literal.
	NUMBER                              // NUMBER represents any numeric literal.
	NULL                                // NULL represents a null value.
	BOOLEAN                             // BOOLEAN represents a boolean value.
	COMMA                               // COMMA represents a ',' separator.
	COLON                               // COLON represents a ':' separator.
	LEFT_SQUARE_BRACE                   // LEFT_SQUARE_BRACE represents '['.
	RIGHT_SQUARE_BRACE                  // RIGHT_SQUARE_BRACE represents ']'.
	LEFT_CURLY_BRACE                    // LEFT_CURLY_BRACE represents '{'.
	RIGHT_CURLY_BRACE                   // RIGHT_CURLY_BRACE represents '}'.
)

// String returns a string representation of the TokenKind.
func (t TokenKind) String() string {
	m := map[TokenKind]string{
		0:  "EOF",
		1:  "ILLEGAL",
		2:  "WHITESPACE",
		3:  "COMMENT",
		4:  "STRING",
		5:  "NUMBER",
		6:  "NULL",
		7:  "BOOLEAN",
		8:  "COMMA",
		9:  "COLON",
		10: "LEFT_SQUARE_BRACE",
		11: "RIGHT_SQUARE_BRACE",
		12: "LEFT_CURLY_BRACE",
		13: "RIGHT_CURLY_BRACE",
	}

	str := m[t]
	return str
}

// Token represents a single lexical token, including its kind, value, and position.
type Token struct {
	Kind    TokenKind    // The general kind of the token (e.g., STRING, NUMBER).
	SubKind TokenSubKind // The specific sub kind within a kind (e.g., INTEGER vs FLOAT).
	Literal []byte       // The literal value of the token.
	Line    int          // The line number where the token appears (1-based index).
	Column  int          // The column position in the line (0-based index).
}

// newToken creates and returns a new Token.
// If cb is non-nil, it is called during token creation.
func newToken(kind TokenKind, subKind TokenSubKind, literal []byte, line, column int, cb func()) Token {
	if cb != nil {
		cb()
	}

	return Token{
		Kind:    kind,
		SubKind: subKind,
		Literal: literal,
		Line:    line,
		Column:  column,
	}
}

// newTokenPtr creates and returns a pointer to a new Token.
// If cb is non-nil, it is called during token creation.
func newTokenPtr(kind TokenKind, subKind TokenSubKind, literal []byte, line, column int, cb func()) *Token {
	token := newToken(kind, subKind, literal, line, column, cb)

	return &token
}

// String returns a human-readable representation of the token.
func (t *Token) String() string {
	return fmt.Sprintf(
		"Token{Kind: %s, SubKind: %s, Literal: %s, Line: %d, Column: %d}",
		t.Kind,
		t.SubKind,
		string(t.Literal),
		t.Line,
		t.Column,
	)
}

func (t *Token) Value() any {
	switch t.Kind {
	case NULL:
		return nil
	case BOOLEAN:
		switch t.SubKind {
		case FALSE:
			return false
		case TRUE:
			return true
		default:
			return nil
		}
	case STRING:
		switch t.SubKind {
		case SINGLE_QUOTED, DOUBLE_QUOTED:
			return unescapeString(stringContent(t.Literal))
		case IDENT:
			return string(t.Literal)
		default:
			return nil
		}
	case NUMBER:
		switch t.SubKind {
		case INTEGER:
			// int64 when it fits; otherwise the nearest float64.
			if val, err := ToInt(t.Literal); err == nil {
				return val
			}
			val, _ := ToFloat(t.Literal)
			return val
		case HEX:
			// int64 when it fits, like INTEGER; otherwise the nearest float64.
			val, ok := hexToBigInt(t.Literal)
			if !ok {
				return nil
			}
			if val.IsInt64() {
				return val.Int64()
			}
			floatVal, _ := hexToFloat64(t.Literal)
			return floatVal
		case FLOAT, SCI_NOT:
			val, _ := ToFloat(t.Literal)
			return val
		default:
			return nil
		}
	default:
		return nil
	}
}

func (t *Token) Equal(t2 *Token) bool {
	if t == nil || t2 == nil {
		return t == t2
	}

	return t.Kind == t2.Kind &&
		t.SubKind == t2.SubKind &&
		bytes.Equal(t.Literal, t2.Literal) &&
		t.Line == t2.Line &&
		t.Column == t2.Column
}

// Tokens is a slice of Token.
type Tokens []Token

// String returns a formatted string listing all tokens in the slice.
func (tks Tokens) String() string {
	parts := make([]string, len(tks))

	for i, token := range tks {
		parts[i] = token.String()
	}

	return fmt.Sprintf("\n[\n%s\n]\n", strings.Join(parts, ",\n"))
}

func (tks Tokens) Split() ([][2]int, error) {
	count := 0
	streams := [][2]int{}
	// fmt.Println(tks)

	for count < len(tks) {
		token := tks[count]
		switch token.Kind {
		case NULL, BOOLEAN, STRING, NUMBER:
			streams = append(streams, [2]int{count, count})
			count++
		case COMMENT, WHITESPACE, EOF:
			count++
		case LEFT_SQUARE_BRACE:
			start := count
			count++
			lc := 1
			rc := 0

			if count < len(tks) && tks[count].Kind == RIGHT_SQUARE_BRACE {
				rc++
				count++
			}

			for lc != rc {
				if count >= len(tks) {
					index := count - 1
					streams = append(streams, [2]int{start, index})
					if index < len(tks) && tks[index].Kind == ILLEGAL {
						return streams, ErrUnexpectedToken
					}

					return streams, ErrUnbalancedArrayBrackets
				}
				tok := tks[count]

				if tok.Kind == LEFT_SQUARE_BRACE {
					lc++
				}

				if tok.Kind == RIGHT_SQUARE_BRACE {
					rc++
				}
				count++
			}

			streams = append(streams, [2]int{start, count - 1})
		case LEFT_CURLY_BRACE:
			start := count
			count++
			lc := 1
			rc := 0

			if count < len(tks) && tks[count].Kind == RIGHT_CURLY_BRACE {
				rc++
				count++
			}

			for lc != rc {
				if count >= len(tks) {
					index := count - 1
					streams = append(streams, [2]int{start, index})
					if index < len(tks) && tks[index].Kind == ILLEGAL {
						return streams, ErrUnexpectedToken
					}

					return streams, ErrUnbalancedObjectBrackets
				}
				tok := tks[count]

				if tok.Kind == LEFT_CURLY_BRACE {
					lc++
				}

				if tok.Kind == RIGHT_CURLY_BRACE {
					rc++
				}
				count++
			}

			streams = append(streams, [2]int{start, count - 1})
		case ILLEGAL:
			streams = append(streams, [2]int{count, count})
			return streams, ErrIllegalToken
		default:
			streams = append(streams, [2]int{count, count})
			return streams, ErrUnexpectedToken
		}
	}

	return streams, nil
}

func (tks Tokens) Equal(tks2 Tokens) bool {
	if len(tks) != len(tks2) {
		return false
	}

	for i, tk := range tks {
		tk2 := tks[i]

		if !tk.Equal(&tk2) {
			return false
		}
	}

	return true
}
