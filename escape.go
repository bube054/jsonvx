package jsonvx

import (
	"bytes"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// unescapeString decodes the escape sequences in the contents of a quoted string
// (the text between the quotes) and returns the resulting string.
//
// The lexer has already rejected escapes that the active ParserConfig does not allow,
// so this applies the full set of rules unconditionally:
//
//   - JSON: \" \\ \/ \b \f \n \r \t and \uXXXX, including UTF-16 surrogate pairs
//     such as \ud83d\ude00. A lone or mismatched surrogate becomes U+FFFD,
//     matching encoding/json.
//   - JSON5: \' \v \0 \xHH, a backslash before a line terminator (\n, \r\n, \r,
//     U+2028, U+2029) continues the string onto the next line, and a backslash
//     before any other character yields that character.
func unescapeString(content []byte) string {
	// Fast path: most strings contain no escapes.
	if bytes.IndexByte(content, '\\') < 0 {
		return string(content)
	}

	var b strings.Builder
	b.Grow(len(content))

	for i := 0; i < len(content); {
		c := content[i]

		if c != '\\' {
			b.WriteByte(c)
			i++
			continue
		}

		// A trailing lone backslash cannot come from the lexer; keep it as is.
		if i+1 >= len(content) {
			b.WriteByte('\\')
			break
		}

		next := content[i+1]
		i += 2 // skip the backslash and the escape character

		switch next {
		case '"', '\'', '\\', '/':
			b.WriteByte(next)
		case 'b':
			b.WriteByte('\b')
		case 'f':
			b.WriteByte('\f')
		case 'n':
			b.WriteByte('\n')
		case 'r':
			b.WriteByte('\r')
		case 't':
			b.WriteByte('\t')
		case 'v':
			b.WriteByte('\v')
		case '0':
			b.WriteByte(0)
		case '\n':
			// Line continuation: the backslash and newline are dropped.
		case '\r':
			// Line continuation over CR or CRLF.
			if i < len(content) && content[i] == '\n' {
				i++
			}
		case 'x':
			if r, ok := parseHex(content[i:], 2); ok {
				b.WriteRune(r)
				i += 2
			} else {
				b.WriteByte('x')
			}
		case 'u':
			r, n := decodeUnicodeEscape(content[i:])
			if n == 0 {
				b.WriteByte('u') // not followed by 4 hex digits; cannot come from the lexer
			} else {
				b.WriteRune(r)
				i += n
			}
		default:
			// Any other character (possibly multi-byte) stands for itself,
			// except U+2028 and U+2029, which are JSON5 line continuations.
			r, size := utf8.DecodeRune(content[i-1:])
			i += size - 1

			if r != '\u2028' && r != '\u2029' {
				b.WriteRune(r)
			}
		}
	}

	return b.String()
}

// decodeUnicodeEscape decodes the hex digits that follow "\u" at the start of input,
// combining a UTF-16 surrogate pair ("d83d\ude00") into one rune. It returns the rune
// and the number of bytes consumed, or 0 bytes if input does not start with 4 hex digits.
func decodeUnicodeEscape(input []byte) (rune, int) {
	r, ok := parseHex(input, 4)
	if !ok {
		return 0, 0
	}

	if !utf16.IsSurrogate(r) {
		return r, 4
	}

	// A high surrogate must be followed by \u and a low surrogate to form a pair.
	if r < 0xDC00 && len(input) >= 10 && input[4] == '\\' && input[5] == 'u' {
		if low, ok := parseHex(input[6:], 4); ok {
			if pair := utf16.DecodeRune(r, low); pair != utf8.RuneError {
				return pair, 10
			}
		}
	}

	return utf8.RuneError, 4
}

// parseHex parses exactly n hex digits at the start of input.
func parseHex(input []byte, n int) (rune, bool) {
	if len(input) < n {
		return 0, false
	}

	var r rune
	for _, c := range input[:n] {
		var v byte
		switch {
		case '0' <= c && c <= '9':
			v = c - '0'
		case 'a' <= c && c <= 'f':
			v = c - 'a' + 10
		case 'A' <= c && c <= 'F':
			v = c - 'A' + 10
		default:
			return 0, false
		}
		r = r<<4 | rune(v)
	}

	return r, true
}
