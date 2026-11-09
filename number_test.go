package jsonvx

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func mustParseNumber(t *testing.T, input string, cfg *ParserConfig) *Number {
	t.Helper()

	parser := NewParser([]byte(input), cfg)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error for %q: %v", input, err)
	}

	num, ok := AsNumber(node)
	if !ok {
		t.Fatalf("expected *Number for %q, got %T", input, node)
	}

	return num
}

func TestLargeIDExactWithInt64(t *testing.T) {
	// The motivating case: an ID one past 2^53 is rounded by Value but exact via Int64.
	parser := NewParser([]byte(`{"user_id": 9007199254740993}`), nil)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	obj, _ := AsObject(node)
	value, _ := obj.QueryPath("user_id")
	num, _ := AsNumber(value)

	if approx, _ := num.Value(); approx != 9007199254740992 {
		t.Errorf("Value() = %.0f, expected the rounded 9007199254740992", approx)
	}

	if exact, err := num.Int64(); err != nil || exact != 9007199254740993 {
		t.Errorf("Int64() = (%d, %v), expected 9007199254740993", exact, err)
	}
}

func TestNumberInt64(t *testing.T) {
	hex := NewParserConfig(WithAllowHexNumbers(true))
	plus := NewParserConfig(WithAllowLeadingPlus(true))
	json5 := JSON5Config()

	tests := []struct {
		msg         string
		input       string
		cfg         *ParserConfig
		expected    int64
		expectedErr error
	}{
		{msg: "small", input: "42", expected: 42},
		{msg: "negative", input: "-42", expected: -42},
		{msg: "zero", input: "0", expected: 0},
		{msg: "negative zero", input: "-0", expected: 0},
		{msg: "leading plus", input: "+5", cfg: plus, expected: 5},
		{msg: "2^53 + 1", input: "9007199254740993", expected: 9007199254740993},
		{msg: "largest int64", input: "9223372036854775807", expected: math.MaxInt64},
		{msg: "smallest int64", input: "-9223372036854775808", expected: math.MinInt64},
		{msg: "hex", input: "0x1F", cfg: hex, expected: 31},
		{msg: "negative hex", input: "-0x10", cfg: hex, expected: -16},
		{msg: "smallest int64 in hex", input: "-0x8000000000000000", cfg: hex, expected: math.MinInt64},

		{msg: "one past largest int64", input: "9223372036854775808", expectedErr: ErrNumberOutOfRange},
		{msg: "one past smallest int64", input: "-9223372036854775809", expectedErr: ErrNumberOutOfRange},
		{msg: "twenty digits", input: "99999999999999999999", expectedErr: ErrNumberOutOfRange},
		{msg: "hex beyond int64", input: "0xFFFFFFFFFFFFFFFF", cfg: hex, expectedErr: ErrNumberOutOfRange},

		{msg: "decimal", input: "1.5", expectedErr: ErrNotInteger},
		{msg: "whole number written as decimal", input: "1.0", expectedErr: ErrNotInteger},
		{msg: "exponent", input: "1e3", expectedErr: ErrNotInteger},
		{msg: "exponent beyond float64", input: "1e400", expectedErr: ErrNotInteger},
		{msg: "Infinity", input: "Infinity", cfg: json5, expectedErr: ErrNotInteger},
		{msg: "NaN", input: "NaN", cfg: json5, expectedErr: ErrNotInteger},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			num := mustParseNumber(t, test.input, test.cfg)
			got, err := num.Int64()

			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("Int64() error = %v, expected %v", err, test.expectedErr)
			}

			if test.expectedErr == nil && got != test.expected {
				t.Errorf("Int64() = %d, expected %d", got, test.expected)
			}
		})
	}
}

func TestNumberBigInt(t *testing.T) {
	hex := NewParserConfig(WithAllowHexNumbers(true))
	huge := "1" + strings.Repeat("0", 400)

	tests := []struct {
		msg         string
		input       string
		cfg         *ParserConfig
		expected    string
		expectedErr error
	}{
		{msg: "small", input: "42", expected: "42"},
		{msg: "2^53 + 1", input: "9007199254740993", expected: "9007199254740993"},
		{msg: "twenty digits", input: "99999999999999999999", expected: "99999999999999999999"},
		{msg: "negative beyond int64", input: "-9223372036854775809", expected: "-9223372036854775809"},
		{msg: "401 digits", input: huge, expected: huge},
		{msg: "hex beyond int64", input: "0xFFFFFFFFFFFFFFFF", cfg: hex, expected: "18446744073709551615"},
		{msg: "decimal", input: "1.5", expectedErr: ErrNotInteger},
		{msg: "exponent", input: "1e3", expectedErr: ErrNotInteger},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			num := mustParseNumber(t, test.input, test.cfg)
			got, err := num.BigInt()

			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("BigInt() error = %v, expected %v", err, test.expectedErr)
			}

			if test.expectedErr == nil && got.String() != test.expected {
				t.Errorf("BigInt() = %s, expected %s", got, test.expected)
			}
		})
	}
}

func TestNumberRaw(t *testing.T) {
	json5 := JSON5Config()

	for _, input := range []string{"1.50", "9007199254740993", "-0", "1e3", "1E+3", "0x1F", "+5", ".5", "Infinity", "NaN"} {
		num := mustParseNumber(t, input, json5)

		if got, err := num.Raw(); err != nil || got != input {
			t.Errorf("Raw() = (%q, %v), expected %q", got, err, input)
		}
	}
}

func TestNumberAccessorsInvalid(t *testing.T) {
	var nilNum *Number
	empty := &Number{}

	if _, err := nilNum.BigInt(); err != ErrNotNumber {
		t.Errorf("nil BigInt() error = %v, expected %v", err, ErrNotNumber)
	}

	if _, err := empty.Int64(); err != ErrNotNumber {
		t.Errorf("empty Int64() error = %v, expected %v", err, ErrNotNumber)
	}

	if _, err := empty.Raw(); err != ErrNotNumber {
		t.Errorf("empty Raw() error = %v, expected %v", err, ErrNotNumber)
	}

	if _, err := nilNum.Raw(); err != ErrNotNumber {
		t.Errorf("nil Raw() error = %v, expected %v", err, ErrNotNumber)
	}
}

func TestLargeIntegersClassifiedAsInteger(t *testing.T) {
	// Whole numbers too large for int64 used to be labelled FLOAT, and very long
	// ones were rejected as illegal input.
	tests := []struct {
		input    string
		expected TokenSubKind
	}{
		{input: "9223372036854775807", expected: INTEGER},
		{input: "9223372036854775808", expected: INTEGER},
		{input: "99999999999999999999", expected: INTEGER},
		{input: "-99999999999999999999", expected: INTEGER},
		{input: "1" + strings.Repeat("0", 400), expected: INTEGER},
		{input: "1.5", expected: FLOAT},
	}

	for _, test := range tests {
		num := mustParseNumber(t, test.input, nil)
		if num.Token.SubKind != test.expected {
			t.Errorf("%.30s: SubKind = %v, expected %v", test.input, num.Token.SubKind, test.expected)
		}
	}
}

func TestLargeIntegerValue(t *testing.T) {
	// Beyond int64, Value still returns the nearest float64, as before.
	if got, err := mustParseNumber(t, "99999999999999999999", nil).Value(); err != nil || got != 1e20 {
		t.Errorf("Value() = (%v, %v), expected 1e20", got, err)
	}

	// Beyond float64's range, Value reports an error; BigInt still has the exact value.
	huge := mustParseNumber(t, "1"+strings.Repeat("0", 400), nil)
	if _, err := huge.Value(); !errors.Is(err, ErrNotNumber) {
		t.Errorf("Value() error = %v, expected %v", err, ErrNotNumber)
	}
}

func TestIntegerTokenValue(t *testing.T) {
	tests := []struct {
		literal  string
		expected any
	}{
		{literal: "42", expected: int64(42)},
		{literal: "9223372036854775807", expected: int64(math.MaxInt64)},
		{literal: "99999999999999999999", expected: float64(1e20)},
	}

	for _, test := range tests {
		token := newToken(NUMBER, INTEGER, []byte(test.literal), 1, 1, nil)
		if got := token.Value(); got != test.expected {
			t.Errorf("Token.Value() for %s = %v (%T), expected %v (%T)", test.literal, got, got, test.expected, test.expected)
		}
	}
}

func TestIsInteger(t *testing.T) {
	valid := []string{"0", "12", "-12", "+12", "99999999999999999999"}
	invalid := []string{"", "+", "-", "1.0", "1e3", "12a", "0x1F", "--1", "+-1"}

	for _, input := range valid {
		if !isInteger([]byte(input)) {
			t.Errorf("isInteger(%q) = false, expected true", input)
		}
	}

	for _, input := range invalid {
		if isInteger([]byte(input)) {
			t.Errorf("isInteger(%q) = true, expected false", input)
		}
	}
}
