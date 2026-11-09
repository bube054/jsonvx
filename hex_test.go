package jsonvx

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestHexNumberValue(t *testing.T) {
	hex := NewParserConfig(WithAllowHexNumbers(true))
	hexPlus := NewParserConfig(WithAllowHexNumbers(true), WithAllowLeadingPlus(true))

	tests := []struct {
		msg      string
		input    string
		cfg      *ParserConfig
		expected float64
	}{
		{msg: "README example", input: "0x1F", cfg: hex, expected: 31},
		{msg: "lowercase digits", input: "0xff", cfg: hex, expected: 255},
		{msg: "uppercase digits", input: "0xFF", cfg: hex, expected: 255},
		{msg: "uppercase prefix", input: "0X10", cfg: hex, expected: 16},
		{msg: "mixed case digits", input: "0xDeCaF", cfg: hex, expected: 912559},
		{msg: "zero", input: "0x0", cfg: hex, expected: 0},
		{msg: "leading zeros", input: "0x000A", cfg: hex, expected: 10},
		{msg: "negative", input: "-0x10", cfg: hex, expected: -16},
		{msg: "negative uppercase", input: "-0X2A", cfg: hex, expected: -42},
		{msg: "leading plus", input: "+0x10", cfg: hexPlus, expected: 16},
		{msg: "largest int64", input: "0x7FFFFFFFFFFFFFFF", cfg: hex, expected: 9223372036854775807},
		{msg: "beyond int64", input: "0xFFFFFFFFFFFFFFFF", cfg: hex, expected: 18446744073709551615},
		{msg: "beyond uint64", input: "0x10000000000000000", cfg: hex, expected: 18446744073709551616},
		{msg: "JSON5 config", input: "0xdecaf", cfg: JSON5Config(), expected: 912559},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			parser := NewParser([]byte(test.input), test.cfg)
			node, err := parser.Parse()
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			num, ok := AsNumber(node)
			if !ok {
				t.Fatalf("expected *Number, got %T", node)
			}

			got, err := num.Value()
			if err != nil || got != test.expected {
				t.Errorf("Value() = (%v, %v), expected %v", got, err, test.expected)
			}
		})
	}
}

func TestHexNumberTooLargeForFloat64(t *testing.T) {
	// 0x1 followed by 256 zeros is 2^1024, beyond the largest float64.
	input := "0x1" + strings.Repeat("0", 256)

	parser := NewParser([]byte(input), NewParserConfig(WithAllowHexNumbers(true)))
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	num, _ := AsNumber(node)
	got, err := num.Value()

	if !errors.Is(err, ErrNotNumber) || math.IsInf(got, 0) {
		t.Errorf("Value() = (%v, %v), expected (0, %v)", got, err, ErrNotNumber)
	}
}

func TestHexNumberInObject(t *testing.T) {
	parser := NewParser([]byte(`{hexadecimal: 0xdecaf, list: [0x1, -0x2]}`), JSON5Config())
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	obj, _ := AsObject(node)

	for path, expected := range map[string]float64{"hexadecimal": 912559, "list.0": 1, "list.1": -2} {
		value, err := obj.QueryPath(strings.Split(path, ".")...)
		if err != nil {
			t.Fatalf("QueryPath(%q) error: %v", path, err)
		}

		num, _ := AsNumber(value)
		got, err := num.Value()
		if err != nil || got != expected {
			t.Errorf("%s: Value() = (%v, %v), expected %v", path, got, err, expected)
		}
	}
}

func TestHexTokenValue(t *testing.T) {
	tests := []struct {
		literal  string
		expected any
	}{
		{literal: "0x1F", expected: int64(31)},
		{literal: "-0x10", expected: int64(-16)},
		{literal: "0x7FFFFFFFFFFFFFFF", expected: int64(math.MaxInt64)},
		{literal: "-0x8000000000000000", expected: int64(math.MinInt64)},
		{literal: "0xFFFFFFFFFFFFFFFF", expected: float64(18446744073709551615)},
	}

	for _, test := range tests {
		token := newToken(NUMBER, HEX, []byte(test.literal), 1, 1, nil)
		if got := token.Value(); got != test.expected {
			t.Errorf("Token.Value() for %s = %v (%T), expected %v (%T)", test.literal, got, got, test.expected, test.expected)
		}
	}
}

func TestHexToBigInt(t *testing.T) {
	valid := map[string]string{
		"0x0":                 "0",
		"0x1f":                "31",
		"0X1F":                "31",
		"-0xff":               "-255",
		"+0xff":               "255",
		"0xFFFFFFFFFFFFFFFFF": "295147905179352825855",
	}

	for input, expected := range valid {
		got, ok := hexToBigInt([]byte(input))
		if !ok || got.String() != expected {
			t.Errorf("hexToBigInt(%q) = (%v, %v), expected %s", input, got, ok, expected)
		}
	}

	// Malformed input the lexer never produces is rejected rather than misread.
	for _, input := range []string{"", "0x", "1F", "x1F", "0x1G", "0x-5", "0x+5", "--0x5", "0x_1"} {
		if got, ok := hexToBigInt([]byte(input)); ok {
			t.Errorf("hexToBigInt(%q) = %v, expected it to be rejected", input, got)
		}
	}
}

func TestToIntUnchanged(t *testing.T) {
	// ToInt is exported and stays decimal-only, so "017" is 17, not octal 15.
	if got, err := ToInt([]byte("017")); err != nil || got != 17 {
		t.Errorf("ToInt(\"017\") = (%v, %v), expected 17", got, err)
	}

	if _, err := ToInt([]byte("0x1F")); err == nil {
		t.Errorf("ToInt(\"0x1F\") should still fail; hex goes through Number.Value")
	}
}
