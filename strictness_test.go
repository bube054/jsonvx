package jsonvx

import (
	"testing"
)

// Regression tests for the strictness bugs JSONTestSuite found: numbers validated
// with Go's number syntax instead of JSON's, NUL bytes treated as the end of input,
// raw control characters in strings, and unquoted words accepted as values.

type acceptTest struct {
	input  string
	cfg    *ParserConfig
	accept bool
}

func runAcceptTests(t *testing.T, tests []acceptTest) {
	t.Helper()

	for _, test := range tests {
		mode := "strict"
		if test.cfg != nil {
			mode = "configured"
		}

		parser := NewParser([]byte(test.input), test.cfg)
		node, err := parser.Parse()

		if test.accept && err != nil {
			t.Errorf("%s %q: expected it to be accepted, got error: %v", mode, test.input, err)
		}

		if !test.accept && err == nil {
			t.Errorf("%s %q: expected an error, got %v", mode, test.input, node)
		}
	}
}

func TestNumberGrammarStrict(t *testing.T) {
	valid := []string{"0", "-0", "123", "-123", "0.0", "1.5", "-1.5e10", "1E5", "0e1", "0e+1", "0E-1", "1e400"}
	invalid := []string{
		"-01", "012", "00", "-012", // leading zeros
		"2.e3", "0.e1", "2.e+3", "5.", // point with no digits after it
		".5", "-.5", // point with no digits before it
		"inf", "Inf", "infinity", "nan", // words Go's strconv reads as numbers
		"1e", "1e+", "-", "1.5.5", "1ee5", "--1", "1.e", "+1",
	}

	var tests []acceptTest
	for _, input := range valid {
		tests = append(tests, acceptTest{input: input, accept: true})
	}
	for _, input := range invalid {
		tests = append(tests, acceptTest{input: input, accept: false})
	}

	runAcceptTests(t, tests)
}

func TestNumberGrammarJSON5(t *testing.T) {
	json5 := JSON5Config()

	valid := []string{".5", "5.", "-.5", "+.5", "5.e3", "+1", "Infinity", "-Infinity", "NaN", "0x1F", "-0x1f", "+0x1F", "1e5"}
	invalid := []string{"01", "-01", ".", ".e3", "inf", "nan", "1e", "0x", "0xG"}

	var tests []acceptTest
	for _, input := range valid {
		tests = append(tests, acceptTest{input: input, cfg: json5, accept: true})
	}
	for _, input := range invalid {
		tests = append(tests, acceptTest{input: input, cfg: json5, accept: false})
	}

	runAcceptTests(t, tests)
}

func TestNumberClassificationUnchanged(t *testing.T) {
	json5 := JSON5Config()

	tests := []struct {
		input    string
		cfg      *ParserConfig
		expected TokenSubKind
	}{
		{input: "5", expected: INTEGER},
		{input: "-5", expected: INTEGER},
		{input: "1.5", expected: FLOAT},
		{input: "1e5", expected: FLOAT},
		{input: "0e1", expected: FLOAT},
		{input: "1e400", expected: SCI_NOT},
		{input: ".5", cfg: json5, expected: FLOAT},
		{input: "5.", cfg: json5, expected: FLOAT},
		{input: "0x1F", cfg: json5, expected: HEX},
	}

	for _, test := range tests {
		num := mustParseNumber(t, test.input, test.cfg)
		if num.Token.SubKind != test.expected {
			t.Errorf("%q: SubKind = %v, expected %v", test.input, num.Token.SubKind, test.expected)
		}
	}
}

func TestNumberErrorKinds(t *testing.T) {
	tests := []struct {
		input    string
		expected TokenSubKind
	}{
		{input: "-01", expected: INVALID_LEADING_ZERO},
		{input: "012", expected: INVALID_LEADING_ZERO},
		{input: "2.e3", expected: INVALID_POINT_EDGE_DOT},
		{input: ".5", expected: INVALID_POINT_EDGE_DOT},
		{input: "-.5", expected: INVALID_POINT_EDGE_DOT},
	}

	for _, test := range tests {
		token := NewLexer([]byte(test.input), NewParserConfig()).Token()
		if token.Kind != ILLEGAL || token.SubKind != test.expected {
			t.Errorf("%q: got %v %v, expected ILLEGAL %v", test.input, token.Kind, token.SubKind, test.expected)
		}
	}
}

func TestNULByteIsNotEndOfInput(t *testing.T) {
	json5 := JSON5Config()

	runAcceptTests(t, []acceptTest{
		// Before the fix, everything after a NUL byte was silently ignored.
		{input: "[1]\x00garbage", accept: false},
		{input: "{\"a\": 1}\x00{\"b\": 2}", accept: false},
		{input: "123\x00", accept: false},
		{input: "\x00", accept: false},
		{input: "[1]\x00", cfg: json5, accept: false},

		// A NUL byte inside a string is a raw control character.
		{input: "\"a\x00b\"", accept: false},
		{input: "\"a\x00b\"", cfg: json5, accept: true},

		// Comments that run to the end of the input still work.
		{input: "1 // trailing comment", cfg: json5, accept: true},
		{input: "1 /* unterminated", cfg: json5, accept: false},
	})
}

func TestControlCharactersInStrings(t *testing.T) {
	json5 := JSON5Config()
	controlChars := NewParserConfig(WithAllowControlCharsInStrings(true))

	runAcceptTests(t, []acceptTest{
		// Strict JSON: control characters must be escaped.
		{input: "\"a\tb\"", accept: false},
		{input: "\"a\nb\"", accept: false},
		{input: "\"a\rb\"", accept: false},
		{input: "\"\x1f\"", accept: false},
		{input: "\"\\t\\n\\r\"", accept: true}, // escaped forms are fine
		{input: "\"\x7f\"", accept: true},      // DEL is not a control character in JSON

		// AllowControlCharsInStrings (and JSON5): raw tabs and similar are allowed...
		{input: "\"a\tb\"", cfg: controlChars, accept: true},
		{input: "\"a\tb\"", cfg: json5, accept: true},
		{input: "'a\tb'", cfg: json5, accept: true},
		{input: "\"\x1f\"", cfg: json5, accept: true},

		// ...but raw line breaks never are.
		{input: "\"a\nb\"", cfg: json5, accept: false},
		{input: "\"a\rb\"", cfg: json5, accept: false},
		{input: "'a\nb'", cfg: json5, accept: false},

		// An escaped line break is AllowNewlineInStrings, which is unaffected.
		{input: "\"a\\\nb\"", cfg: json5, accept: true},
	})
}

func TestRawTabValueInJSON5(t *testing.T) {
	str, ok := AsString(mustParse(t, "\"a\tb\"", JSON5Config()))
	if !ok {
		t.Fatal("expected a string")
	}

	if got, _ := str.Value(); got != "a\tb" {
		t.Errorf("Value() = %q, expected %q", got, "a\tb")
	}
}

func TestUnquotedWordsOnlyAsKeys(t *testing.T) {
	json5 := JSON5Config()

	runAcceptTests(t, []acceptTest{
		{input: "{name: 1}", cfg: json5, accept: true},
		{input: "{a: {b: [1]}}", cfg: json5, accept: true},
		{input: "[Infinity, NaN]", cfg: json5, accept: true},

		{input: "[foo]", cfg: json5, accept: false},
		{input: "foo", cfg: json5, accept: false},
		{input: "[inf]", cfg: json5, accept: false},
		{input: "{a: foo}", cfg: json5, accept: false},
		{input: "{a: [foo]}", cfg: json5, accept: false},
	})
}

func mustParse(t *testing.T, input string, cfg *ParserConfig) JSON {
	t.Helper()

	parser := NewParser([]byte(input), cfg)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error for %q: %v", input, err)
	}

	return node
}
