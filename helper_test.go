package jsonvx

import (
	"testing"
)

func TestIsNewLine(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       byte
		expected bool
	}{
		{msg: "Valid newline", p1: '\n', expected: true},
		{msg: "Invalid newline", p1: '\t', expected: false},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := isNewLine(test.p1)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}

func BenchmarkIsNewLine(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isNewLine('\n')
	}
}

func TestIsWhiteSpace(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       byte
		p2       bool
		expected bool
	}{
		// Standard whitespaces without AllowExtraWS
		{msg: "Valid space, without AllowExtraWS", p1: '\x20', p2: false, expected: true},
		{msg: "Valid line feed, without AllowExtraWS", p1: '\x0A', p2: false, expected: true},
		{msg: "Valid carriage return, without AllowExtraWS", p1: '\x0D', p2: false, expected: true},
		{msg: "Valid horizontal tab, without AllowExtraWS", p1: '\x09', p2: false, expected: true},

		//  Standard whitespaces with AllowExtraWS
		{msg: "Valid space, with AllowExtraWS", p1: '\x20', p2: true, expected: true},
		{msg: "Valid line feed, with AllowExtraWS", p1: '\x0A', p2: true, expected: true},
		{msg: "Valid carriage return, with AllowExtraWS", p1: '\x0D', p2: true, expected: true},
		{msg: "Valid horizontal tab, with AllowExtraWS", p1: '\x09', p2: true, expected: true},

		//  Additional whitespaces without AllowExtraWS
		{msg: "Valid line tabulation, with AllowExtraWS", p1: '\x0B', p2: true, expected: true},
		{msg: "Valid form feed, with AllowExtraWS", p1: '\x0C', p2: true, expected: true},
		{msg: "Valid next line, with AllowExtraWS", p1: '\u0085', p2: true, expected: true},
		{msg: "Valid no break space, with AllowExtraWS", p1: '\u00A0', p2: true, expected: true},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := isWhiteSpace(test.p1, test.p2)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}

func BenchmarkIsWhiteSpace(b *testing.B) {
	for i := 0; i < b.N; i++ {
		isWhiteSpace(' ', true)
	}
}

func TestIs4HexDigits(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       [4]byte
		expected bool
	}{
		// Valid 4 hex digits
		{msg: "Valid hex only digits", p1: [4]byte{48, 48, 51, 49}, expected: true},
		{msg: "Valid hex uppercase", p1: [4]byte{'A', 'B', 'C', 'F'}, expected: true},
		{msg: "Valid hex lowercase", p1: [4]byte{'a', 'b', 'c', 'f'}, expected: true},
		{msg: "Mixed valid hex", p1: [4]byte{'0', 'F', 'a', '9'}, expected: true},

		// Invalid 4 hex digits
		{msg: "Invalid non-hex character", p1: [4]byte{'0', 'G', 'a', '9'}, expected: false},
		{msg: "All invalid", p1: [4]byte{'g', 'h', 'z', '!'}, expected: false},
		{msg: "Edge case: just below range", p1: [4]byte{'/', 'A', 'B', 'C'}, expected: false},
		{msg: "Edge case: just above range", p1: [4]byte{'G', '0', '1', '2'}, expected: false},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := is4HexDigits(test.p1)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}

func BenchmarkIs4HexDigits(b *testing.B) {
	for i := 0; i < b.N; i++ {
		is4HexDigits([4]byte{'0', 'F', 'a', '9'})
	}
}

func TestIsPossibleJSIdentifier(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       byte
		expected bool
	}{
		// Valid possible js indent
		{msg: "dollar sign", p1: '$', expected: true},
		{msg: "underscore", p1: '_', expected: true},
		{msg: "digit", p1: '8', expected: true},
		{msg: "lowercase letter", p1: 'b', expected: true},
		{msg: "upper letter", p1: 'F', expected: true},
		{msg: "unicode letter", p1: 'ï', expected: true},

		// Invalid possible js indent
		{msg: "space", p1: ' ', expected: false},
		{msg: "nothing", p1: 0, expected: false},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := isPossibleJSIdentifier(test.p1)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}

func TestIsNaN(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       []byte
		expected bool
	}{
		// Valid possible Nan
		{msg: "Valid NaN", p1: []byte("NaN"), expected: true},

		// Invalid possible Nan
		{msg: "Invalid NaN", p1: []byte("Nan"), expected: false},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := isNaN(test.p1)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}

func TestIsInf(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       []byte
		expected bool
	}{
		// Valid possible Nan
		{msg: "Valid Infinity", p1: []byte("Infinity"), expected: true},
		{msg: "Valid pos Infinity", p1: []byte("+Infinity"), expected: true},
		{msg: "Valid neg Infinity", p1: []byte("-Infinity"), expected: true},

		// Invalid possible Nan
		{msg: "Invalid Infinity case-sensitive", p1: []byte("infinity"), expected: false},
		{msg: "Invalid pos Infinity case-sensitive", p1: []byte("+infinity"), expected: false},
		{msg: "Invalid neg Infinity case-sensitive", p1: []byte("-infinity"), expected: false},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := isInf(test.p1)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}

func TestIsSciNot(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       []byte
		expected bool
	}{
		// Valid scientific notation with e
		{msg: "Valid e sci-not neutral mantissa and exponent", p1: []byte("2e10"), expected: true},
		{msg: "Valid e sci-not pos mantissa and pos exponent", p1: []byte("+2e+10"), expected: true},
		{msg: "Valid e sci-not neg mantissa and pos exponent", p1: []byte("-2e+10"), expected: true},
		{msg: "Valid e sci-not pos mantissa and neg exponent", p1: []byte("+2e-10"), expected: true},
		{msg: "Valid e sci-not neg mantissa and neg exponent", p1: []byte("-2e-10"), expected: true},
		{msg: "Valid e sci-not neg mantissa and float mantissa and neg exponent", p1: []byte("-2.0e-10"), expected: true},

		// Valid scientific notation with E
		{msg: "Valid E sci-not neutral mantissa and exponent", p1: []byte("2E10"), expected: true},
		{msg: "Valid E sci-not pos mantissa and pos exponent", p1: []byte("+2E+10"), expected: true},
		{msg: "Valid E sci-not neg mantissa and pos exponent", p1: []byte("-2E+10"), expected: true},
		{msg: "Valid E sci-not pos mantissa and neg exponent", p1: []byte("+2E-10"), expected: true},
		{msg: "Valid E sci-not neg mantissa and neg exponent", p1: []byte("-2E-10"), expected: true},
		{msg: "Valid E sci-not neg mantissa and float mantissa and neg exponent", p1: []byte("-2.0E-10"), expected: true},

		// Invalid scientific notation
		{msg: "Invalid sci-not no mantissa", p1: []byte("e10"), expected: false},
		{msg: "Invalid sci-not no exponent", p1: []byte("2e"), expected: false},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := isScientificNotation(test.p1)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}

func TestIsHex(t *testing.T) {
	var tests = []struct {
		msg      string
		p1       []byte
		expected bool
	}{
		// Valid hex
		{msg: "Valid hex", p1: []byte("0xdecaf"), expected: true},
		{msg: "Valid hex 0", p1: []byte("0x0"), expected: true},
		{msg: "Valid hex 26", p1: []byte("0x1A"), expected: true},
		{msg: "Valid hex 255", p1: []byte("0xFF"), expected: true},
		{msg: "Valid hex 16", p1: []byte("0X10"), expected: true},
		{msg: "Valid hex 127", p1: []byte("0X7f"), expected: true},
		{msg: "Valid hex -42", p1: []byte("-0X2A"), expected: true},

		// Invalid hex
		{msg: "Invalid hex no digits", p1: []byte("0x"), expected: false},
		{msg: "Invalid hex no F>", p1: []byte("0xG1"), expected: false},
		{msg: "Invalid hex no prefix", p1: []byte("123"), expected: false},
		{msg: "Invalid hex no CSS style hex color", p1: []byte("FF00FF"), expected: false},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := isHex(test.p1)

			if got != test.expected {
				t.Errorf("got %t, expected %t", got, test.expected)
			}
		})
	}
}
