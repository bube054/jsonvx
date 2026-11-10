package jsonvx

import (
	"fmt"
	"slices"
	"testing"
)

func TestObjectAll(t *testing.T) {
	obj, _ := AsObject(mustParse(t, `{"name": "Megumin", "spell": "Explosion", "a": 1, "a": 2}`, nil))

	var got []string
	for key, value := range obj.All() {
		got = append(got, fmt.Sprintf("%s=%v", key, value))
	}

	// Input order, and a repeated key keeps its first position with its last value.
	want := []string{"name=Megumin", "spell=Explosion", "a=2"}
	if !slices.Equal(got, want) {
		t.Errorf("All() = %v, expected %v", got, want)
	}
}

func TestObjectValues(t *testing.T) {
	obj, _ := AsObject(mustParse(t, `{"name": "Megumin", "spell": "Explosion"}`, nil))

	var got []string
	for value := range obj.Values() {
		got = append(got, value.String())
	}

	if want := []string{"Megumin", "Explosion"}; !slices.Equal(got, want) {
		t.Errorf("Values() = %v, expected %v", got, want)
	}
}

func TestArrayAll(t *testing.T) {
	arr, _ := AsArray(mustParse(t, `["Aqua", "Megumin", "Darkness"]`, nil))

	var got []string
	for i, item := range arr.All() {
		got = append(got, fmt.Sprintf("%d=%v", i, item))
	}

	if want := []string{"0=Aqua", "1=Megumin", "2=Darkness"}; !slices.Equal(got, want) {
		t.Errorf("All() = %v, expected %v", got, want)
	}
}

func TestArrayValues(t *testing.T) {
	arr, _ := AsArray(mustParse(t, `["Aqua", "Megumin", "Darkness"]`, nil))

	got := slices.Collect(arr.Values())
	if len(got) != 3 || got[2].String() != "Darkness" {
		t.Errorf("Values() = %v, expected [Aqua Megumin Darkness]", got)
	}
}

func TestIteratorsBreakEarly(t *testing.T) {
	obj, _ := AsObject(mustParse(t, `{"a": 1, "b": 2, "c": 3}`, nil))
	arr, _ := AsArray(mustParse(t, `[1, 2, 3]`, nil))

	// Breaking out of the loop must stop the iterator; if it kept yielding after
	// yield returned false, Go would panic.
	count := 0
	for range obj.All() {
		count++
		break
	}
	for range obj.Values() {
		count++
		break
	}
	for range arr.All() {
		count++
		break
	}
	for range arr.Values() {
		count++
		break
	}

	if count != 4 {
		t.Errorf("expected each loop to run once, got %d iterations in total", count)
	}
}

func TestIteratorsEmptyAndNil(t *testing.T) {
	emptyObj, _ := AsObject(mustParse(t, `{}`, nil))
	emptyArr, _ := AsArray(mustParse(t, `[]`, nil))
	var nilObj *Object
	var nilArr *Array

	for _, obj := range []*Object{emptyObj, nilObj} {
		for range obj.All() {
			t.Error("expected no iterations over an empty or nil object")
		}
		for range obj.Values() {
			t.Error("expected no iterations over an empty or nil object")
		}
	}

	for _, arr := range []*Array{emptyArr, nilArr} {
		for range arr.All() {
			t.Error("expected no iterations over an empty or nil array")
		}
		for range arr.Values() {
			t.Error("expected no iterations over an empty or nil array")
		}
	}
}

func TestIteratorsMatchForEach(t *testing.T) {
	obj, _ := AsObject(mustParse(t, `{"x": [1, {"y": null}], "z": true}`, nil))

	var fromForEach, fromAll []string
	obj.ForEach(func(key []byte, value JSON, _ *Object) {
		fromForEach = append(fromForEach, string(key)+"="+value.String())
	})
	for key, value := range obj.All() {
		fromAll = append(fromAll, key+"="+value.String())
	}

	if !slices.Equal(fromForEach, fromAll) {
		t.Errorf("All() = %v, ForEach gave %v", fromAll, fromForEach)
	}
}

func ExampleObject_All() {
	parser := NewParser([]byte(`{"name": "Megumin", "spell": "Explosion"}`), nil)
	node, _ := parser.Parse()
	obj, _ := AsObject(node)

	for key, value := range obj.All() {
		fmt.Printf("%s: %v\n", key, value)
	}
	// Output:
	// name: Megumin
	// spell: Explosion
}

func ExampleArray_All() {
	parser := NewParser([]byte(`["Aqua", "Megumin", "Darkness"]`), nil)
	node, _ := parser.Parse()
	arr, _ := AsArray(node)

	for i, item := range arr.All() {
		fmt.Printf("%d: %v\n", i, item)
	}
	// Output:
	// 0: Aqua
	// 1: Megumin
	// 2: Darkness
}
