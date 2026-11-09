package jsonvx

import (
	"errors"
	"strings"
	"testing"
)

// nestedArrays returns depth nested arrays, e.g. nestedArrays(3) == "[[[]]]".
func nestedArrays(depth int) []byte {
	return []byte(strings.Repeat("[", depth) + strings.Repeat("]", depth))
}

// nestedObjects returns depth nested objects, e.g. nestedObjects(2) == `{"a":{"a":{}}}` minus one level.
func nestedObjects(depth int) []byte {
	return []byte(strings.Repeat(`{"a":`, depth-1) + "{}" + strings.Repeat("}", depth-1))
}

func TestJSONParserMaxDepth(t *testing.T) {
	tests := []struct {
		msg         string
		input       []byte
		cfg         *ParserConfig
		expectedErr error
	}{
		{msg: "default config allows DefaultMaxDepth arrays", input: nestedArrays(DefaultMaxDepth), cfg: nil, expectedErr: nil},
		{msg: "default config rejects DefaultMaxDepth+1 arrays", input: nestedArrays(DefaultMaxDepth + 1), cfg: nil, expectedErr: ErrJSONTooDeep},
		{msg: "default config allows DefaultMaxDepth objects", input: nestedObjects(DefaultMaxDepth), cfg: nil, expectedErr: nil},
		{msg: "default config rejects DefaultMaxDepth+1 objects", input: nestedObjects(DefaultMaxDepth + 1), cfg: nil, expectedErr: ErrJSONTooDeep},
		{msg: "zero-value config uses DefaultMaxDepth", input: nestedArrays(DefaultMaxDepth + 1), cfg: &ParserConfig{}, expectedErr: ErrJSONTooDeep},
		{msg: "negative MaxDepth uses DefaultMaxDepth", input: nestedArrays(DefaultMaxDepth + 1), cfg: NewParserConfig(WithMaxDepth(-1)), expectedErr: ErrJSONTooDeep},
		{msg: "JSON5 config uses DefaultMaxDepth", input: nestedArrays(DefaultMaxDepth + 1), cfg: JSON5Config(), expectedErr: ErrJSONTooDeep},
		{msg: "custom limit allows exactly MaxDepth", input: []byte(`[[1]]`), cfg: NewParserConfig(WithMaxDepth(2)), expectedErr: nil},
		{msg: "custom limit rejects MaxDepth+1 arrays", input: []byte(`[[[1]]]`), cfg: NewParserConfig(WithMaxDepth(2)), expectedErr: ErrJSONTooDeep},
		{msg: "custom limit rejects MaxDepth+1 objects", input: []byte(`{"a": {"b": {}}}`), cfg: NewParserConfig(WithMaxDepth(2)), expectedErr: ErrJSONTooDeep},
		{msg: "custom limit counts mixed arrays and objects", input: []byte(`[{"a": [1]}]`), cfg: NewParserConfig(WithMaxDepth(2)), expectedErr: ErrJSONTooDeep},
		{msg: "siblings do not add up to depth", input: []byte(`[[1], [2], {"a": [3]}, [[4]]]`), cfg: NewParserConfig(WithMaxDepth(3)), expectedErr: nil},
		{msg: "higher custom limit allows deeper input", input: nestedArrays(5000), cfg: NewParserConfig(WithMaxDepth(5000)), expectedErr: nil},
		{msg: "scalars are not affected by depth", input: []byte(`"just a string"`), cfg: NewParserConfig(WithMaxDepth(1)), expectedErr: nil},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			parser := NewParser(test.input, test.cfg)
			node, err := parser.Parse()

			if !errors.Is(err, test.expectedErr) {
				t.Fatalf("got error %v, expected %v", err, test.expectedErr)
			}

			if test.expectedErr == nil && node == nil {
				t.Fatalf("got nil node with no error")
			}

			if test.expectedErr != nil && node != nil {
				t.Fatalf("got node %v alongside error %v", node, err)
			}
		})
	}
}

func TestJSONParserMaxDepthErrorPosition(t *testing.T) {
	parser := NewParser([]byte(`[[[1]]]`), NewParserConfig(WithMaxDepth(2)))
	_, err := parser.Parse()

	expected := `JSON nesting too deep: "[" exceeds maximum depth of 2 at line 1, column 3`
	if err == nil || err.Error() != expected {
		t.Fatalf("got error %q, expected %q", err, expected)
	}
}

func TestJSONParserReusableAfterDepthError(t *testing.T) {
	parser := NewParser([]byte(`[[[1]]]`), NewParserConfig(WithMaxDepth(3)))

	for i := 0; i < 2; i++ {
		if _, err := parser.Parse(); err != nil {
			t.Fatalf("parse %d: unexpected error %v", i+1, err)
		}
	}
}

// TestJSONParserHostileNesting reproduces the original crash: before the depth limit,
// this input overflowed the 1 GB goroutine stack and killed the process with
// "fatal error: stack overflow", which recover() cannot catch.
func TestJSONParserHostileNesting(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping 10 MB hostile input in short mode")
	}

	parser := NewParser(nestedArrays(5_000_000), nil)
	node, err := parser.Parse()

	if !errors.Is(err, ErrJSONTooDeep) {
		t.Fatalf("got error %v, expected %v", err, ErrJSONTooDeep)
	}

	if node != nil {
		t.Fatalf("got node alongside error")
	}
}
