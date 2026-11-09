package jsonvx

import (
	"bytes"
	"math/big"
	"strconv"
	"unicode"
)

// isNewLine reports whether a byte is a newline.
func isNewLine(char byte) bool {
	return char == '\n'
}

// isWhiteSpace reports whether a byte is a whitespace.
func isWhiteSpace(char byte, AllowExtraWS bool) bool {
	switch char {
	case
		'\x20', // space
		'\x0A', // line feed
		'\x0D', // carriage return
		'\x09': // horizontal tab
		return true
	case
		'\x0B',   // line tabulation
		'\x0C',   // form feed
		'\u0085', // next line
		'\u00A0': // no break space
		return AllowExtraWS
	default:
		return false
	}
}

// is4HexDigits reports if all 4 bytes in the input array are valid hexadecimal characters (0-9, A-F, a-f).
func is4HexDigits(chars [4]byte) bool {
	for _, char := range chars {
		if (char >= '0' && char <= '9') || (char >= 'A' && char <= 'F') || (char >= 'a' && char <= 'f') {
			continue
		}
		return false
	}
	return true
}

func isJSCombiningMark(r rune) bool {
	return unicode.In(r, unicode.Mn, unicode.Mc)
}

func isDigit(b byte) bool {
	if b >= 48 && b <= 57 {
		return true
	}

	return false
}

func isPlus(b byte) bool {
	return b == 43
}

func isMinus(b byte) bool {
	return b == 45
}

func isDot(b byte) bool {
	return b == 46
}

func isExponent(b byte) bool {
	return b == 'e' || b == 'E'
}

func isPossibleJSIdentifier(b byte) bool {
	if b == '+' || b == '-' || b == '*' || b == '/' {
		return false
	}

	if isDigit(b) {
		return true
	}

	if b == '$' || b == '_' {
		return true
	}

	if unicode.IsLetter(rune(b)) {
		return true
	}

	return false
}

func isHexLetter(b byte) bool {
	return (b >= 65 && b <= 70) || (b >= 97 && b <= 102)
}

func isPossibleNumber(b byte) bool {
	return b == '+' ||
		b == '-' ||
		b == '.' ||
		b == 'e' ||
		b == 'E' ||
		b == 'N' ||
		b == 'a' ||
		b == 'I' ||
		b == 'i' ||
		b == 'n' ||
		b == 'f' ||
		b == 't' ||
		b == 'y' ||
		b == '0' ||
		b == 'x' ||
		b == 'X' ||
		(b >= 'A' && b <= 'F') ||
		(b >= 'a' && b <= 'f') ||
		isDigit(b)
}

func startsWithPlus(input []byte) bool {
	return bytes.HasPrefix(input, []byte{'+'})
}

func startsWithDigit(input []byte) bool {
	return len(input) > 0 && input[0] >= '0' && input[0] <= '9'
}

func startsOrEndsWithDot(input []byte) bool {
	return bytes.HasPrefix(input, []byte{'.'}) || bytes.HasSuffix(input, []byte{'.'})
}

func isNaN(input []byte) bool {
	if after, ok := bytes.CutPrefix(input, []byte{'+'}); ok {
		return isNaN(after)
	}

	if after, ok := bytes.CutPrefix(input, []byte{'-'}); ok {
		return isNaN(after)
	}

	if len(input) != 3 {
		return false
	}

	f := input[0]
	s := input[1]
	t := input[2]

	return f == 'N' && s == 'a' && t == 'N'
}

func isInf(input []byte) bool {
	if after, ok := bytes.CutPrefix(input, []byte{'+'}); ok {
		return isInf(after)
	}

	if after, ok := bytes.CutPrefix(input, []byte{'-'}); ok {
		return isInf(after)
	}

	if len(input) != 8 {
		return false
	}

	return input[0] == 'I' &&
		input[1] == 'n' &&
		input[2] == 'f' &&
		input[3] == 'i' &&
		input[4] == 'n' &&
		input[5] == 'i' &&
		input[6] == 't' &&
		input[7] == 'y'
}

func isInteger(input []byte) bool {
	// Decide by the characters (an optional sign, then digits), not by whether the
	// value fits in an int64, so whole numbers of any size are integers.
	if len(input) > 0 && (input[0] == '+' || input[0] == '-') {
		input = input[1:]
	}

	if len(input) == 0 {
		return false
	}

	for _, b := range input {
		if !isDigit(b) {
			return false
		}
	}

	return true
}

func isFloat(input []byte) bool {
	_, err := strconv.ParseFloat(string(input), 64)

	return err == nil
}

func isScientificNotation(input []byte) bool {
	parts := bytes.Split(input, []byte("e"))

	if len(parts) < 2 {
		parts = bytes.Split(input, []byte("E"))
	}

	if len(parts) < 2 {
		return false
	}

	part1 := parts[0]
	part2 := parts[1]

	part1IsFltOrInt := isInteger(part1) || isFloat(part1)
	part2IsInt := isInteger(part2)

	return part1IsFltOrInt && part2IsInt
}

func isHex(input []byte) bool {
	if bytes.HasPrefix(input, []byte("+")) {
		input = bytes.TrimPrefix(input, []byte("+"))
	}

	if bytes.HasPrefix(input, []byte("-")) {
		input = bytes.TrimPrefix(input, []byte("-"))
	}

	if len(input) < 3 {
		return false
	}

	if !(bytes.HasPrefix(input, []byte("0x")) || bytes.HasPrefix(input, []byte("0X"))) {
		return false
	}

	for _, b := range input[2:] {
		if (b >= 48 && b <= 57) || (b >= 65 && b <= 70) || (b >= 97 && b <= 102) {
			continue
		}

		return false
	}

	return true
}

// stringContent returns the text between the quotes of a quoted string literal,
// exactly as written (escape sequences are not decoded).
func stringContent(input []byte) []byte {
	double := []byte(`"`)
	single := []byte(`'`)

	end := double

	if bytes.HasPrefix(input, single) {
		end = single
	}

	after, _ := bytes.CutPrefix(input, end)
	before, _ := bytes.CutSuffix(after, end)

	return before
}

// func intValue(input []byte) int64 {
// 	num, _ := strconv.ParseInt(string(input), 10, 64)

// 	return num
// }

// func floatValue(input []byte) float64 {
// 	num, _ := strconv.ParseFloat(string(input), 64)

// 	return num
// }

func ToInt(input []byte) (int64, error) {
	return strconv.ParseInt(string(input), 10, 64)
}

func ToFloat(input []byte) (float64, error) {
	return strconv.ParseFloat(string(input), 64)
}

// hexToBigInt converts a hexadecimal literal such as 0x1F, 0XFF, -0x10 or +0x10
// to its exact integer value, however many digits it has. It reports false if
// input is not a well-formed hex literal.
func hexToBigInt(input []byte) (*big.Int, bool) {
	negative := false

	switch {
	case bytes.HasPrefix(input, []byte("-")):
		negative = true
		input = input[1:]
	case bytes.HasPrefix(input, []byte("+")):
		input = input[1:]
	}

	if len(input) < 3 || input[0] != '0' || (input[1] != 'x' && input[1] != 'X') {
		return nil, false
	}

	digits := input[2:]

	// big.Int.SetString would also accept a sign here ("0x-5"), so check the digits first.
	for _, b := range digits {
		if !isDigit(b) && !isHexLetter(b) {
			return nil, false
		}
	}

	n, ok := new(big.Int).SetString(string(digits), 16)
	if !ok {
		return nil, false
	}

	if negative {
		n.Neg(n)
	}

	return n, true
}

// hexToFloat64 converts a hexadecimal literal to the nearest float64. Values beyond
// float64's range become ±Inf; it reports false if input is not a hex literal.
func hexToFloat64(input []byte) (float64, bool) {
	n, ok := hexToBigInt(input)
	if !ok {
		return 0, false
	}

	f, _ := new(big.Float).SetInt(n).Float64()
	return f, true
}
