package jsonvx

import (
	"fmt"
	"strings"
	"testing"
)

func TestObjectKeepsKeyOrder(t *testing.T) {
	obj := mustParseObject(t, `{"name": "app", "version": "1.0", "dependencies": {}, "author": "me"}`, nil)

	expected := "name,version,dependencies,author"

	if got := strings.Join(obj.Keys(), ","); got != expected {
		t.Errorf("Keys() = %s, expected %s", got, expected)
	}

	var visited []string
	obj.ForEach(func(key []byte, _ JSON, _ *Object) { visited = append(visited, string(key)) })
	if got := strings.Join(visited, ","); got != expected {
		t.Errorf("ForEach order = %s, expected %s", got, expected)
	}
}

func TestObjectStringKeepsKeyOrder(t *testing.T) {
	obj := mustParseObject(t, `{"z": 1, "a": 2, "m": 3}`, nil)

	if got, expected := obj.String(), "{z: 1, a: 2, m: 3}"; got != expected {
		t.Errorf("String() = %s, expected %s", got, expected)
	}
}

func TestNestedObjectsKeepKeyOrder(t *testing.T) {
	obj := mustParseObject(t, `{"outer": {"z": 1, "a": 2}, "list": [{"y": 1, "b": 2}]}`, nil)

	inner, _ := obj.QueryPath("outer")
	innerObj, _ := AsObject(inner)
	if got := strings.Join(innerObj.Keys(), ","); got != "z,a" {
		t.Errorf("nested Keys() = %s, expected z,a", got)
	}

	element, _ := obj.QueryPath("list", "0")
	elementObj, _ := AsObject(element)
	if got := strings.Join(elementObj.Keys(), ","); got != "y,b" {
		t.Errorf("array element Keys() = %s, expected y,b", got)
	}
}

func TestObjectKeysEmpty(t *testing.T) {
	obj := mustParseObject(t, `{}`, nil)

	if keys := obj.Keys(); len(keys) != 0 {
		t.Errorf("Keys() = %q, expected none", keys)
	}
}

func TestQueryPathLargeObject(t *testing.T) {
	// Keys written in reverse alphabetical order, so nothing depends on sorting.
	const n = 1000
	var b strings.Builder
	b.WriteString("{")
	for i := n - 1; i >= 0; i-- {
		fmt.Fprintf(&b, `"key%04d": %d`, i, i)
		if i > 0 {
			b.WriteString(", ")
		}
	}
	b.WriteString("}")

	obj := mustParseObject(t, b.String(), nil)

	if obj.Len() != n {
		t.Fatalf("Len() = %d, expected %d", obj.Len(), n)
	}

	if keys := obj.Keys(); keys[0] != "key0999" || keys[n-1] != "key0000" {
		t.Errorf("first and last keys = %s, %s, expected key0999, key0000", keys[0], keys[n-1])
	}

	for i := 0; i < n; i++ {
		key := fmt.Sprintf("key%04d", i)
		got, err := obj.QueryPath(key)
		if err != nil {
			t.Fatalf("QueryPath(%q) error: %v", key, err)
		}
		if got.String() != fmt.Sprint(i) {
			t.Fatalf("QueryPath(%q) = %v, expected %d", key, got, i)
		}
	}

	if _, err := obj.QueryPath("missing"); err == nil {
		t.Errorf("QueryPath(\"missing\") should fail")
	}
}

func TestDuplicateKeyKeepsFirstPositionAndLastValue(t *testing.T) {
	obj := mustParseObject(t, `{"a": 1, "b": 2, "a": 3}`, nil)

	if got := strings.Join(obj.Keys(), ","); got != "a,b" {
		t.Errorf("Keys() = %s, expected a,b", got)
	}

	if got, _ := obj.QueryPath("a"); got.String() != "3" {
		t.Errorf("QueryPath(\"a\") = %v, expected 3", got)
	}
}

func numberNode(literal string) *Number {
	return &Number{Token: newTokenPtr(NUMBER, INTEGER, []byte(literal), 1, 1, nil)}
}

func TestQueryPathWithoutIndex(t *testing.T) {
	// An Object built by hand has no index, so lookups scan Properties.
	obj := &Object{Properties: []KeyValue{
		newKeyValue([]byte("z"), numberNode("1")),
		newKeyValue([]byte("a"), numberNode("2")),
		newKeyValue([]byte("z"), numberNode("3")),
	}}

	if got, err := obj.QueryPath("a"); err != nil || got.String() != "2" {
		t.Errorf("QueryPath(\"a\") = (%v, %v), expected 2", got, err)
	}

	// With repeated keys and no index, the last copy wins, as when parsing.
	if got, err := obj.QueryPath("z"); err != nil || got.String() != "3" {
		t.Errorf("QueryPath(\"z\") = (%v, %v), expected 3", got, err)
	}

	if _, err := obj.QueryPath("missing"); err == nil {
		t.Errorf("QueryPath(\"missing\") should fail")
	}
}

func TestQueryPathAfterPropertiesChanged(t *testing.T) {
	// Properties is exported, so callers can reorder or shrink it after parsing.
	// Lookups must notice the index no longer matches and still give right answers.
	obj := mustParseObject(t, `{"a": 1, "b": 2, "c": 3}`, nil)

	// Reverse the order: the index now points at the wrong positions.
	p := obj.Properties
	p[0], p[2] = p[2], p[0]

	for key, expected := range map[string]string{"a": "1", "b": "2", "c": "3"} {
		if got, err := obj.QueryPath(key); err != nil || got.String() != expected {
			t.Errorf("after reorder, QueryPath(%q) = (%v, %v), expected %s", key, got, err, expected)
		}
	}

	// Drop the last property: its index entry now points past the end.
	obj.Properties = obj.Properties[:2]

	if _, err := obj.QueryPath("a"); err == nil {
		t.Errorf("after removal, QueryPath(\"a\") should fail")
	}

	if got, err := obj.QueryPath("c"); err != nil || got.String() != "3" {
		t.Errorf("after removal, QueryPath(\"c\") = (%v, %v), expected 3", got, err)
	}
}

func TestObjectEqualIgnoresKeyOrder(t *testing.T) {
	one, two := numberNode("1"), numberNode("2")

	ab := &Object{Properties: []KeyValue{newKeyValue([]byte("a"), one), newKeyValue([]byte("b"), two)}}
	ba := &Object{Properties: []KeyValue{newKeyValue([]byte("b"), two), newKeyValue([]byte("a"), one)}}
	differentValue := &Object{Properties: []KeyValue{newKeyValue([]byte("a"), two), newKeyValue([]byte("b"), one)}}
	differentKey := &Object{Properties: []KeyValue{newKeyValue([]byte("a"), one), newKeyValue([]byte("c"), two)}}
	shorter := &Object{Properties: []KeyValue{newKeyValue([]byte("a"), one)}}

	if !ab.Equal(ba) || !ba.Equal(ab) {
		t.Errorf("objects with the same keys in a different order should be equal")
	}

	if ab.Equal(differentValue) {
		t.Errorf("objects with different values should not be equal")
	}

	if ab.Equal(differentKey) {
		t.Errorf("objects with different keys should not be equal")
	}

	if ab.Equal(shorter) || shorter.Equal(ab) {
		t.Errorf("objects with different lengths should not be equal")
	}

	if ab.Equal(&Array{}) {
		t.Errorf("an object should not equal an array")
	}
}

func TestObjectEqualNil(t *testing.T) {
	var nilObj *Object
	obj := &Object{}

	if !nilObj.Equal(nilObj) {
		t.Errorf("nil object should equal itself")
	}

	if obj.Equal(nilObj) || nilObj.Equal(obj) {
		t.Errorf("nil and non-nil objects should not be equal")
	}

	if obj.Equal(nil) {
		t.Errorf("object should not equal a nil JSON value")
	}
}
