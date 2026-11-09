package jsonvx

import (
	"errors"
	"testing"
)

func TestStringOnEmptyNodesDoesNotPanic(t *testing.T) {
	var (
		nilString  *String
		nilNumber  *Number
		nilBoolean *Boolean
		nilArray   *Array
		nilObject  *Object
	)

	tests := []struct {
		msg  string
		node JSON
	}{
		{msg: "empty String", node: &String{}},
		{msg: "empty Number", node: &Number{}},
		{msg: "empty Boolean", node: &Boolean{}},
		{msg: "nil String", node: nilString},
		{msg: "nil Number", node: nilNumber},
		{msg: "nil Boolean", node: nilBoolean},
		{msg: "nil Array", node: nilArray},
		{msg: "nil Object", node: nilObject},
	}

	for _, test := range tests {
		t.Run(test.msg, func(t *testing.T) {
			if got := test.node.String(); got != "<nil>" {
				t.Errorf("String() = %q, expected %q", got, "<nil>")
			}
		})
	}
}

func TestValueOnNilNodesReturnsError(t *testing.T) {
	var (
		nilNull    *Null
		nilBoolean *Boolean
		nilString  *String
		nilNumber  *Number
	)

	if _, err := nilNull.Value(); !errors.Is(err, ErrNotNull) {
		t.Errorf("nil Null Value() error = %v, expected %v", err, ErrNotNull)
	}

	if _, err := nilBoolean.Value(); !errors.Is(err, ErrNotBoolean) {
		t.Errorf("nil Boolean Value() error = %v, expected %v", err, ErrNotBoolean)
	}

	if _, err := nilString.Value(); !errors.Is(err, ErrNotString) {
		t.Errorf("nil String Value() error = %v, expected %v", err, ErrNotString)
	}

	if _, err := nilString.Raw(); !errors.Is(err, ErrNotString) {
		t.Errorf("nil String Raw() error = %v, expected %v", err, ErrNotString)
	}

	if _, err := nilNumber.Value(); !errors.Is(err, ErrNotNumber) {
		t.Errorf("nil Number Value() error = %v, expected %v", err, ErrNotNumber)
	}
}

func TestStringOnParsedNodesUnchanged(t *testing.T) {
	obj := mustParseObject(t, `{"s": "text", "n": 1.50, "b": false, "a": [1, 2], "o": {}}`, nil)

	expected := map[string]string{
		"s": "text",
		"n": "1.50",
		"b": "\033[1mfalse\033[0m",
		"a": "[1, 2]",
		"o": "{}",
	}

	for key, want := range expected {
		value, err := obj.QueryPath(key)
		if err != nil {
			t.Fatalf("QueryPath(%q) error: %v", key, err)
		}

		if got := value.String(); got != want {
			t.Errorf("%s: String() = %q, expected %q", key, got, want)
		}
	}
}

func TestQueryPathKeyNotFound(t *testing.T) {
	obj := mustParseObject(t, `{"a": {"b": 1}}`, nil)

	tests := []struct {
		path    []string
		message string
	}{
		{path: []string{"missing"}, message: `key not found: "missing"`},
		{path: []string{"a", "missing"}, message: `key not found: "missing"`},
	}

	for _, test := range tests {
		_, err := obj.QueryPath(test.path...)

		if !errors.Is(err, ErrKeyNotFound) {
			t.Errorf("QueryPath(%q) error = %v, expected %v", test.path, err, ErrKeyNotFound)
		}

		// The message is unchanged from before ErrKeyNotFound existed.
		if err == nil || err.Error() != test.message {
			t.Errorf("QueryPath(%q) message = %q, expected %q", test.path, err, test.message)
		}
	}
}
