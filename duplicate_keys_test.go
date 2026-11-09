package jsonvx

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func mustParseObject(t *testing.T, input string, cfg *ParserConfig) *Object {
	t.Helper()

	parser := NewParser([]byte(input), cfg)
	node, err := parser.Parse()
	if err != nil {
		t.Fatalf("unexpected parse error for %q: %v", input, err)
	}

	obj, ok := AsObject(node)
	if !ok {
		t.Fatalf("expected *Object for %q, got %T", input, node)
	}

	return obj
}

func TestDuplicateKeysLastWins(t *testing.T) {
	var manyKeys strings.Builder
	manyKeys.WriteString(`{"a": "first", `)
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&manyKeys, `"k%02d": %d, `, i, i)
	}
	manyKeys.WriteString(`"a": "last"}`)

	tests := []struct {
		msg      string
		input    string
		cfg      *ParserConfig
		path     []string
		expected string
		length   int
	}{
		{msg: "adjacent duplicates", input: `{"a": "first", "a": "last"}`, path: []string{"a"}, expected: "last", length: 1},
		{msg: "duplicates 40 keys apart", input: manyKeys.String(), path: []string{"a"}, expected: "last", length: 41},
		{msg: "three copies", input: `{"a": 1, "a": 2, "b": 0, "a": 3}`, path: []string{"a"}, expected: "3", length: 2},
		{msg: "escaped duplicate", input: `{"a": "first", "\u0061": "last"}`, path: []string{"a"}, expected: "last", length: 1},
		{msg: "unquoted and quoted duplicate", input: `{a: "first", "a": "last"}`, cfg: NewParserConfig(WithAllowUnquoted(true)), path: []string{"a"}, expected: "last", length: 1},
		{msg: "JSON5 config keeps last-wins", input: `{a: 'first', a: 'last',}`, cfg: JSON5Config(), path: []string{"a"}, expected: "last", length: 1},
		{msg: "value type can change", input: `{"a": [1], "a": {"b": 2}}`, path: []string{"a", "b"}, expected: "2", length: 1},
		{msg: "duplicate inside nested object", input: `{"o": {"x": 1, "x": 2}}`, path: []string{"o", "x"}, expected: "2", length: 1},
		{msg: "similar keys are distinct", input: `{"a": 1, "A": 2, "a ": 3}`, path: []string{"a"}, expected: "1", length: 3},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			obj := mustParseObject(t, test.input, test.cfg)

			if obj.Len() != test.length {
				t.Errorf("Len() = %d, expected %d", obj.Len(), test.length)
			}

			got, err := obj.QueryPath(test.path...)
			if err != nil {
				t.Fatalf("QueryPath(%q) error: %v", test.path, err)
			}

			if got.String() != test.expected {
				t.Errorf("QueryPath(%q) = %v, expected %s", test.path, got, test.expected)
			}
		})
	}
}

func TestDuplicateKeysForEachSeesEachKeyOnce(t *testing.T) {
	obj := mustParseObject(t, `{"b": 1, "a": 2, "b": 3, "a": 4}`, nil)

	var visited []string
	obj.ForEach(func(key []byte, value JSON, _ *Object) {
		visited = append(visited, fmt.Sprintf("%s=%v", key, value))
	})

	// Each key keeps the position where it first appeared, with its last value.
	expected := []string{"b=3", "a=4"}
	if strings.Join(visited, ",") != strings.Join(expected, ",") {
		t.Errorf("ForEach visited %v, expected %v", visited, expected)
	}
}

func TestDisallowDuplicateKeys(t *testing.T) {
	disallow := NewParserConfig(WithDisallowDuplicateKeys(true))
	disallowJSON5 := JSON5Config()
	disallowJSON5.DisallowDuplicateKeys = true

	tests := []struct {
		msg         string
		input       string
		cfg         *ParserConfig
		expectedErr error
	}{
		{msg: "adjacent duplicates rejected", input: `{"a": 1, "a": 2}`, cfg: disallow, expectedErr: ErrJSONDuplicateKey},
		{msg: "escaped duplicate rejected", input: `{"a": 1, "\u0061": 2}`, cfg: disallow, expectedErr: ErrJSONDuplicateKey},
		{msg: "nested duplicate rejected", input: `{"o": {"x": 1, "x": 2}}`, cfg: disallow, expectedErr: ErrJSONDuplicateKey},
		{msg: "duplicate in array element rejected", input: `[{"x": 1}, {"y": 1, "y": 2}]`, cfg: disallow, expectedErr: ErrJSONDuplicateKey},
		{msg: "unquoted duplicate rejected", input: `{a: 1, 'a': 2}`, cfg: disallowJSON5, expectedErr: ErrJSONDuplicateKey},
		{msg: "same key in different objects allowed", input: `{"a": {"a": 1}}`, cfg: disallow, expectedErr: nil},
		{msg: "same key across array elements allowed", input: `[{"a": 1}, {"a": 2}]`, cfg: disallow, expectedErr: nil},
		{msg: "unique keys allowed", input: `{"a": 1, "b": 2, "A": 3}`, cfg: disallow, expectedErr: nil},
		{msg: "empty object allowed", input: `{}`, cfg: disallow, expectedErr: nil},
		{msg: "default config allows duplicates", input: `{"a": 1, "a": 2}`, cfg: nil, expectedErr: nil},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			parser := NewParser([]byte(test.input), test.cfg)
			node, err := parser.Parse()

			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("got error %v, expected %v", err, test.expectedErr)
			}

			if test.expectedErr != nil && node != nil {
				t.Fatalf("got node %v alongside error %v", node, err)
			}
		})
	}
}

func TestDisallowDuplicateKeysErrorPosition(t *testing.T) {
	disallow := NewParserConfig(WithDisallowDuplicateKeys(true))

	tests := []struct {
		msg      string
		input    string
		expected string
	}{
		{msg: "single line", input: `{"a": 1, "a": 2}`, expected: `duplicate key in JSON object: "a" at line 1, column 10`},
		{msg: "multi line", input: "{\n  \"a\": 1,\n  \"a\": 2\n}", expected: `duplicate key in JSON object: "a" at line 3, column 3`},
		{msg: "escaped key reports decoded name", input: `{"a": 1, "\u0061": 2}`, expected: `duplicate key in JSON object: "a" at line 1, column 10`},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			parser := NewParser([]byte(test.input), disallow)
			_, err := parser.Parse()

			if err == nil || err.Error() != test.expected {
				t.Fatalf("got error %q, expected %q", err, test.expected)
			}
		})
	}
}
