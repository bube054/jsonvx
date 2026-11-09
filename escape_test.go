package jsonvx

import (
	"encoding/json"
	"testing"
	"unicode/utf8"
)

func TestUnescapeString(t *testing.T) {
	tests := []struct {
		msg      string
		input    string // contents between the quotes, as written
		expected string
	}{
		// No escapes
		{msg: "empty", input: ``, expected: ""},
		{msg: "plain text", input: `hello world`, expected: "hello world"},
		{msg: "raw multi-byte text", input: `café 😀`, expected: "café 😀"},

		// Standard JSON escapes
		{msg: "escaped double quote", input: `say \"hi\"`, expected: `say "hi"`},
		{msg: "escaped backslash", input: `C:\\temp`, expected: `C:\temp`},
		{msg: "escaped forward slash", input: `a\/b`, expected: "a/b"},
		{msg: "backspace", input: `a\bb`, expected: "a\bb"},
		{msg: "form feed", input: `a\fb`, expected: "a\fb"},
		{msg: "newline", input: `a\nb`, expected: "a\nb"},
		{msg: "carriage return", input: `a\rb`, expected: "a\rb"},
		{msg: "carriage return then raw newline", input: "a\\r\nb", expected: "a\r\nb"},
		{msg: "tab", input: `a\tb`, expected: "a\tb"},
		{msg: "all standard escapes together", input: `\"\\\/\b\f\n\r\t`, expected: "\"\\/\b\f\n\r\t"},
		{msg: "double backslash before letter", input: `\\n`, expected: `\n`},
		{msg: "escape at start and end", input: `\nmiddle\n`, expected: "\nmiddle\n"},

		// Unicode escapes
		{msg: "unicode lowercase hex", input: `caf\u00e9`, expected: "café"},
		{msg: "unicode uppercase hex", input: `caf\u00E9`, expected: "café"},
		{msg: "unicode ascii", input: `\u0041\u0042`, expected: "AB"},
		{msg: "unicode null", input: `a\u0000b`, expected: "a\x00b"},
		{msg: "unicode three-byte rune", input: `\u20ac`, expected: "€"},
		{msg: "surrogate pair", input: `\ud83d\ude00`, expected: "😀"},
		{msg: "surrogate pair uppercase", input: `\uD83D\uDE00`, expected: "😀"},
		{msg: "lone high surrogate", input: `a\ud83db`, expected: "a\uFFFDb"},
		{msg: "lone low surrogate", input: `a\ude00b`, expected: "a\uFFFDb"},
		{msg: "high surrogate then non-surrogate escape", input: `\ud83d\u0041`, expected: "\uFFFDA"},
		{msg: "two high surrogates", input: `\ud83d\ud83d`, expected: "\uFFFD\uFFFD"},
		{msg: "high surrogate at end", input: `\ud83d`, expected: "\uFFFD"},

		// JSON5 escapes
		{msg: "escaped single quote", input: `it\'s`, expected: "it's"},
		{msg: "vertical tab", input: `a\vb`, expected: "a\vb"},
		{msg: "null character", input: `a\0b`, expected: "a\x00b"},
		{msg: "hex escape", input: `\x41\x42`, expected: "AB"},
		{msg: "hex escape above ascii", input: `\xe9`, expected: "é"},
		{msg: "line continuation LF", input: "Look, Mom! \\\nNo newline", expected: "Look, Mom! No newline"},
		{msg: "line continuation CRLF", input: "a\\\r\nb", expected: "ab"},
		{msg: "line continuation CR", input: "a\\\rb", expected: "ab"},
		{msg: "line continuation U+2028", input: "a\\\u2028b", expected: "ab"},
		{msg: "line continuation U+2029", input: "a\\\u2029b", expected: "ab"},
		{msg: "other character is itself", input: `\a\c\d`, expected: "acd"},
		{msg: "other multi-byte character is itself", input: `\é\😀`, expected: "é😀"},

		// Malformed input the lexer never produces is kept rather than lost
		{msg: "trailing lone backslash", input: `abc\`, expected: `abc\`},
		{msg: "short unicode escape", input: `\u12`, expected: "u12"},
		{msg: "short hex escape", input: `\x4`, expected: "x4"},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			got := unescapeString([]byte(test.input))
			if got != test.expected {
				t.Errorf("unescapeString(%q) = %q, expected %q", test.input, got, test.expected)
			}
		})
	}
}

func TestStringValueDecodesEscapes(t *testing.T) {
	json5 := JSON5Config()

	tests := []struct {
		msg      string
		input    string
		cfg      *ParserConfig
		expected string
		raw      string
	}{
		{msg: "plain string", input: `"hello"`, expected: "hello", raw: "hello"},
		{msg: "newline", input: `"a\nb"`, expected: "a\nb", raw: `a\nb`},
		{msg: "unicode", input: `"caf\u00e9"`, expected: "café", raw: `caf\u00e9`},
		{msg: "escaped quotes", input: `"say \"hi\""`, expected: `say "hi"`, raw: `say \"hi\"`},
		{msg: "escaped backslash only", input: `"\\"`, expected: `\`, raw: `\\`},
		{msg: "path", input: `"C:\\temp\\new"`, expected: `C:\temp\new`, raw: `C:\\temp\\new`},
		{msg: "emoji pair", input: `"\ud83d\ude00"`, expected: "😀", raw: `\ud83d\ude00`},
		// Regression: the lexer used to look back only two characters to decide whether a
		// quote was escaped, so an escaped backslash followed by an escaped quote was rejected.
		{msg: "escaped backslash then escaped quote", input: `"a\\\"b"`, expected: `a\"b`, raw: `a\\\"b`},
		{msg: "two escaped backslashes then escaped quote", input: `"\\\\\""`, expected: `\\"`, raw: `\\\\\"`},
		{msg: "single-quoted escaped backslash then escaped quote", input: `'a\\\'b'`, cfg: json5, expected: `a\'b`, raw: `a\\\'b`},
		{msg: "single-quoted with escaped quote", input: `'it\'s'`, cfg: json5, expected: "it's", raw: `it\'s`},
		{msg: "single-quoted with double quote inside", input: `'say "hi"'`, cfg: json5, expected: `say "hi"`, raw: `say "hi"`},
		{msg: "json5 hex escape", input: `"\x41"`, cfg: json5, expected: "A", raw: `\x41`},
		{msg: "json5 line continuation", input: "\"Look, Mom! \\\nNo \\\\n's!\"", cfg: json5, expected: `Look, Mom! No \n's!`, raw: "Look, Mom! \\\nNo \\\\n's!"},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			parser := NewParser([]byte(test.input), test.cfg)
			node, err := parser.Parse()
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}

			str, ok := AsString(node)
			if !ok {
				t.Fatalf("expected *String, got %T", node)
			}

			value, err := str.Value()
			if err != nil || value != test.expected {
				t.Errorf("Value() = (%q, %v), expected %q", value, err, test.expected)
			}

			raw, err := str.Raw()
			if err != nil || raw != test.raw {
				t.Errorf("Raw() = (%q, %v), expected %q", raw, err, test.raw)
			}
		})
	}
}

func TestUnterminatedStringWithEscapes(t *testing.T) {
	for _, input := range []string{`"abc\`, `"\`, `"\"`, `"a\\\"`, `'\'`} {
		parser := NewParser([]byte(input), JSON5Config())
		if node, err := parser.Parse(); err == nil {
			t.Errorf("input %q: expected an error, got %v", input, node)
		}
	}
}

func TestStringRawUnquotedKey(t *testing.T) {
	parser := NewParser([]byte(`{name: 1}`), NewParserConfig(WithAllowUnquoted(true)))
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	obj, _ := AsObject(node)
	obj.ForEach(func(key []byte, _ JSON, _ *Object) {
		if string(key) != "name" {
			t.Errorf("got key %q, expected %q", key, "name")
		}
	})
}

func TestStringRawInvalid(t *testing.T) {
	if _, err := (&String{}).Raw(); err != ErrNotString {
		t.Errorf("got %v, expected %v", err, ErrNotString)
	}
}

func TestQueryPathDecodedKeys(t *testing.T) {
	input := []byte(`{
		"caf\u00e9": 1,
		"a\"b": 2,
		"tab\there": 3,
		"\ud83d\ude00": 4,
		"plain": {"ne\u0073ted": 5}
	}`)

	parser := NewParser(input, nil)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	obj, _ := AsObject(node)

	tests := []struct {
		path     []string
		expected string
	}{
		{path: []string{"café"}, expected: "1"},
		{path: []string{`a"b`}, expected: "2"},
		{path: []string{"tab\there"}, expected: "3"},
		{path: []string{"😀"}, expected: "4"},
		{path: []string{"plain", "nested"}, expected: "5"},
	}

	for _, test := range tests {
		got, err := obj.QueryPath(test.path...)
		if err != nil {
			t.Errorf("QueryPath(%q) error: %v", test.path, err)
			continue
		}
		if got.String() != test.expected {
			t.Errorf("QueryPath(%q) = %v, expected %s", test.path, got, test.expected)
		}
	}

	// The raw, still-escaped spelling is no longer a key.
	if _, err := obj.QueryPath(`caf\u00e9`); err == nil {
		t.Errorf("QueryPath with escaped spelling should not match the decoded key")
	}
}

func TestObjectKeysSortedByDecodedValue(t *testing.T) {
	// "\u0061" is "a", so it must sort before "b" even though '\' sorts after 'b'.
	parser := NewParser([]byte(`{"b": 2, "\u0061": 1}`), nil)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	obj, _ := AsObject(node)
	var keys []string
	obj.ForEach(func(key []byte, _ JSON, _ *Object) { keys = append(keys, string(key)) })

	if len(keys) != 2 || keys[0] != "a" || keys[1] != "b" {
		t.Errorf("got keys %q, expected [a b]", keys)
	}
}

// stringValueMatchesEncodingJSON reports a mismatch when both jsonvx (in strict mode) and
// encoding/json accept input as a JSON string but decode it differently.
func stringValueMatchesEncodingJSON(t *testing.T, input []byte) {
	t.Helper()

	var want string
	if err := json.Unmarshal(input, &want); err != nil {
		return
	}

	parser := NewParser(input, nil)
	node, err := parser.Parse()
	if err != nil {
		return
	}

	str, ok := AsString(node)
	if !ok {
		return
	}

	got, err := str.Value()
	if err != nil {
		t.Fatalf("Value() error for %q: %v", input, err)
	}

	if got != want {
		t.Errorf("input %q: jsonvx = %q, encoding/json = %q", input, got, want)
	}
}

var encodingJSONStringSeeds = []string{
	`""`,
	`"plain"`,
	`"a\nb\tc\rd\be\ff"`,
	`"\"\\\/"`,
	`"\\"`,
	`"\\\\"`,
	`"a\\\"b"`,
	`"\\n"`,
	`"caf\u00e9"`,
	`"\u20AC"`,
	`"\ud83d\ude00"`,
	`"\ud83d"`,
	`"\ude00"`,
	`"\ud83d\u0041"`,
	`"\u0000"`,
	`"日本語"`,
}

func TestStringValueMatchesEncodingJSON(t *testing.T) {
	for _, seed := range encodingJSONStringSeeds {
		stringValueMatchesEncodingJSON(t, []byte(seed))
	}
}

func FuzzStringValueMatchesEncodingJSON(f *testing.F) {
	for _, seed := range encodingJSONStringSeeds {
		f.Add([]byte(seed))
	}

	f.Fuzz(func(t *testing.T, input []byte) {
		// encoding/json replaces invalid UTF-8 with U+FFFD; jsonvx keeps bytes as is.
		if !utf8.Valid(input) {
			return
		}
		stringValueMatchesEncodingJSON(t, input)
	})
}
