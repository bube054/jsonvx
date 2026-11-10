package jsonvx

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

const isekaiData = `{
  "fan": "bube",
  "count": 3,
  "score": 9.5,
  "finished": false,
  "best.of.all": "Re:Zero",
  "big": 9007199254740993,
  "huge": 123456789012345678901234567890,
  "isekais": [
    {"rank": 1, "title": "Re:Zero", "hero": {"first": "Subaru", "last": "Natsuki"}, "party": ["Emilia", "Rem", "Ram", "Beatrice"]},
    {"rank": 2, "title": "Mushoku Tensei", "hero": {"first": "Rudeus", "last": "Greyrat"}, "party": ["Roxy", "Sylphiette", "Eris"]}
  ]
}`

func mustParseIsekai(t *testing.T) *Object {
	t.Helper()

	root, err := ParseObject([]byte(isekaiData), nil)
	if err != nil {
		t.Fatalf("ParseObject: %v", err)
	}

	return root
}

func TestObjectGetters(t *testing.T) {
	root := mustParseIsekai(t)

	if got, err := root.GetString("isekais", "0", "title"); err != nil || got != "Re:Zero" {
		t.Errorf("GetString = %q, %v", got, err)
	}
	if got, err := root.GetString("best.of.all"); err != nil || got != "Re:Zero" {
		t.Errorf("GetString with dotted key = %q, %v", got, err)
	}
	if got, err := root.GetInt("count"); err != nil || got != 3 {
		t.Errorf("GetInt = %d, %v", got, err)
	}
	if got, err := root.GetInt("big"); err != nil || got != 9007199254740993 {
		t.Errorf("GetInt big = %d, %v (expected exact value)", got, err)
	}
	if got, err := root.GetFloat("score"); err != nil || got != 9.5 {
		t.Errorf("GetFloat = %v, %v", got, err)
	}
	if got, err := root.GetFloat("count"); err != nil || got != 3 {
		t.Errorf("GetFloat on an integer = %v, %v", got, err)
	}
	if got, err := root.GetBool("finished"); err != nil || got != false {
		t.Errorf("GetBool = %v, %v", got, err)
	}

	hero, err := root.GetObject("isekais", "1", "hero")
	if err != nil {
		t.Fatalf("GetObject: %v", err)
	}
	if got, err := hero.GetString("first"); err != nil || got != "Rudeus" {
		t.Errorf("GetString on a GetObject result = %q, %v", got, err)
	}

	party, err := root.GetArray("isekais", "0", "party")
	if err != nil {
		t.Fatalf("GetArray: %v", err)
	}
	if party.Len() != 4 {
		t.Errorf("party has %d members, expected 4", party.Len())
	}

	num, err := root.GetNumber("huge")
	if err != nil {
		t.Fatalf("GetNumber: %v", err)
	}
	if big, err := num.BigInt(); err != nil || big.String() != "123456789012345678901234567890" {
		t.Errorf("GetNumber then BigInt = %v, %v", big, err)
	}
}

func TestArrayGetters(t *testing.T) {
	root := mustParseIsekai(t)
	isekais, err := root.GetArray("isekais")
	if err != nil {
		t.Fatal(err)
	}

	if got, err := isekais.GetString("1", "title"); err != nil || got != "Mushoku Tensei" {
		t.Errorf("GetString = %q, %v", got, err)
	}
	if got, err := isekais.GetInt("1", "rank"); err != nil || got != 2 {
		t.Errorf("GetInt = %d, %v", got, err)
	}
	if got, err := isekais.GetFloat("0", "rank"); err != nil || got != 1 {
		t.Errorf("GetFloat = %v, %v", got, err)
	}
	if _, err := isekais.GetObject("0", "hero"); err != nil {
		t.Errorf("GetObject: %v", err)
	}
	if _, err := isekais.GetArray("0", "party"); err != nil {
		t.Errorf("GetArray: %v", err)
	}
	if _, err := isekais.GetNumber("0", "rank"); err != nil {
		t.Errorf("GetNumber: %v", err)
	}

	flags, err := ParseArray([]byte(`[true, false]`), nil)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := flags.GetBool("0"); err != nil || got != true {
		t.Errorf("GetBool = %v, %v", got, err)
	}
}

func TestGetterErrors(t *testing.T) {
	root := mustParseIsekai(t)

	tests := []struct {
		name    string
		call    func() error
		target  error
		message string
	}{
		{
			name:    "wrong type",
			call:    func() error { _, err := root.GetString("count"); return err },
			target:  ErrWrongType,
			message: `wrong JSON type: expected string, got number at ["count"]`,
		},
		{
			name:    "wrong type deep",
			call:    func() error { _, err := root.GetInt("isekais", "0", "title"); return err },
			target:  ErrWrongType,
			message: `wrong JSON type: expected number, got string at ["isekais" "0" "title"]`,
		},
		{
			name:    "object asked for, array found",
			call:    func() error { _, err := root.GetObject("isekais"); return err },
			target:  ErrWrongType,
			message: `wrong JSON type: expected object, got array at ["isekais"]`,
		},
		{
			name:    "missing key",
			call:    func() error { _, err := root.GetString("isekais", "0", "villain"); return err },
			target:  ErrKeyNotFound,
			message: `key not found: "villain" at ["isekais" "0" "villain"]`,
		},
		{
			name:   "index out of range",
			call:   func() error { _, err := root.GetString("isekais", "9", "title"); return err },
			target: ErrIndexOutOfRange,
		},
		{
			name:   "not an integer",
			call:   func() error { _, err := root.GetInt("score"); return err },
			target: ErrNotInteger,
		},
		{
			name:   "too big for int64",
			call:   func() error { _, err := root.GetInt("huge"); return err },
			target: ErrNumberOutOfRange,
		},
		{
			name:    "empty path returns the root itself",
			call:    func() error { _, err := root.GetString(); return err },
			target:  ErrWrongType,
			message: `wrong JSON type: expected string, got object at root`,
		},
	}

	for _, test := range tests {
		err := test.call()
		if !errors.Is(err, test.target) {
			t.Errorf("%s: got %v, expected an error wrapping %v", test.name, err, test.target)
			continue
		}
		if test.message != "" && err.Error() != test.message {
			t.Errorf("%s: message\n  got      %s\n  expected %s", test.name, err, test.message)
		}
	}
}

func TestGettersOnNil(t *testing.T) {
	root := mustParseIsekai(t)

	// The usual mistake: ignore an error, then call a getter on the nil result.
	hero, _ := root.GetObject("isekais", "0", "villain")
	if _, err := hero.GetString("first"); !errors.Is(err, ErrInvalidJSONType) {
		t.Errorf("GetString on nil *Object: got %v, expected ErrInvalidJSONType", err)
	}

	party, _ := root.GetArray("nope")
	if _, err := party.GetString("0"); !errors.Is(err, ErrInvalidJSONType) {
		t.Errorf("GetString on nil *Array: got %v, expected ErrInvalidJSONType", err)
	}
}

func TestParseHelpers(t *testing.T) {
	if node, err := Parse([]byte(`[1, 2]`), nil); err != nil || node.String() != "[1, 2]" {
		t.Errorf("Parse = %v, %v", node, err)
	}

	if _, err := ParseObject([]byte(`[1, 2]`), nil); !errors.Is(err, ErrWrongType) ||
		err.Error() != "wrong JSON type: expected object, got array at root" {
		t.Errorf("ParseObject on an array: %v", err)
	}

	if _, err := ParseArray([]byte(`{"a": 1}`), nil); !errors.Is(err, ErrWrongType) {
		t.Errorf("ParseArray on an object: %v", err)
	}

	// Parse errors come back unchanged.
	if _, err := ParseObject([]byte(`{"a": 1,,}`), nil); err == nil || errors.Is(err, ErrWrongType) {
		t.Errorf("ParseObject on invalid JSON: %v", err)
	}

	// The config is used.
	status, err := ParseObject([]byte(`{name: 'Rudeus', mana: 0xFF, /* comment */}`), JSON5Config())
	if err != nil {
		t.Fatalf("ParseObject with JSON5Config: %v", err)
	}
	if mana, err := status.GetInt("mana"); err != nil || mana != 255 {
		t.Errorf("GetInt on hex = %d, %v", mana, err)
	}

	// A nil config means strict JSON.
	if _, err := ParseObject([]byte(`{name: 'Rudeus'}`), nil); err == nil {
		t.Error("expected strict mode to reject JSON5 with a nil config")
	}
}

func TestGettersMatchLongForm(t *testing.T) {
	root := mustParseIsekai(t)

	node, _ := root.QueryPath("isekais", "1", "hero", "last")
	str, _ := AsString(node)
	long, _ := str.Value()

	short, err := root.GetString("isekais", "1", "hero", "last")
	if err != nil || short != long {
		t.Errorf("GetString = %q, %v; long form gave %q", short, err, long)
	}
}

func TestWrongTypeNamesEveryType(t *testing.T) {
	root, err := ParseObject([]byte(`{"n": null, "b": true, "x": 1, "s": "s", "a": [], "o": {}}`), nil)
	if err != nil {
		t.Fatal(err)
	}

	for key, want := range map[string]string{"n": "null", "b": "boolean", "x": "number", "s": "string", "a": "array", "o": "object"} {
		if key == "b" {
			continue // the one key GetBool succeeds on
		}
		_, err := root.GetBool(key)
		if err == nil || !strings.Contains(err.Error(), "got "+want+" ") {
			t.Errorf("GetBool(%q): %v, expected it to say got %s", key, err, want)
		}
	}
}

func ExampleParseObject() {
	data := []byte(`{"isekais": [{"title": "Re:Zero", "hero": {"first": "Subaru"}}], "count": 1}`)

	root, err := ParseObject(data, nil)
	if err != nil {
		panic(err)
	}

	title, _ := root.GetString("isekais", "0", "title")
	hero, _ := root.GetString("isekais", "0", "hero", "first")
	count, _ := root.GetInt("count")
	fmt.Println(title, hero, count)

	_, err = root.GetString("count")
	fmt.Println(err)
	// Output:
	// Re:Zero Subaru 1
	// wrong JSON type: expected string, got number at ["count"]
}
