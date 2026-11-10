package jsonvx

import (
	"errors"
	"fmt"
)

// ErrWrongType is returned by the Get* methods and ParseObject/ParseArray when a
// value exists but has a different JSON type than the one asked for.
var ErrWrongType = errors.New("wrong JSON type")

// queryable is implemented by *Object and *Array.
type queryable interface {
	QueryPath(paths ...string) (JSON, error)
}

// getAs finds the value at path and checks that it has type T.
//
// Errors from the lookup (missing key, bad index, ...) are wrapped with the full
// path, so errors.Is still matches ErrKeyNotFound and the others. A value of the
// wrong type gives ErrWrongType.
func getAs[T JSON](root queryable, path []string, want string) (T, error) {
	var zero T

	switch r := root.(type) {
	case *Object:
		if r == nil {
			return zero, fmt.Errorf("%w: Get called on a nil *Object", ErrInvalidJSONType)
		}
	case *Array:
		if r == nil {
			return zero, fmt.Errorf("%w: Get called on a nil *Array", ErrInvalidJSONType)
		}
	}

	node, err := root.QueryPath(path...)
	if err != nil {
		return zero, fmt.Errorf("%w %s", err, describePath(path))
	}

	typed, ok := node.(T)
	if !ok {
		return zero, wrongTypeError(want, node, path)
	}

	return typed, nil
}

func wrongTypeError(want string, got JSON, path []string) error {
	return fmt.Errorf("%w: expected %s, got %s %s", ErrWrongType, want, typeName(got), describePath(path))
}

// describePath formats a path for error messages. Each key is quoted, because keys
// may contain dots or spaces: at ["isekais" "0" "title"].
func describePath(path []string) string {
	if len(path) == 0 {
		return "at root"
	}

	return fmt.Sprintf("at %q", path)
}

// typeName returns the JSON name of a node's type: "string", "number", and so on.
func typeName(node JSON) string {
	switch node.(type) {
	case *Null:
		return "null"
	case *Boolean:
		return "boolean"
	case *Number:
		return "number"
	case *String:
		return "string"
	case *Array:
		return "array"
	case *Object:
		return "object"
	default:
		return "unknown"
	}
}

func getString(root queryable, path []string) (string, error) {
	node, err := getAs[*String](root, path, "string")
	if err != nil {
		return "", err
	}

	return node.Value()
}

func getInt(root queryable, path []string) (int64, error) {
	node, err := getAs[*Number](root, path, "number")
	if err != nil {
		return 0, err
	}

	value, err := node.Int64()
	if err != nil {
		return 0, fmt.Errorf("%w %s", err, describePath(path))
	}

	return value, nil
}

func getFloat(root queryable, path []string) (float64, error) {
	node, err := getAs[*Number](root, path, "number")
	if err != nil {
		return 0, err
	}

	return node.Value()
}

func getBool(root queryable, path []string) (bool, error) {
	node, err := getAs[*Boolean](root, path, "boolean")
	if err != nil {
		return false, err
	}

	return node.Value()
}

// GetString returns the string at path, with escapes decoded.
//
//	title, err := root.GetString("isekais", "0", "title")
//
// It returns an error wrapping ErrKeyNotFound, ErrIndexOutOfRange and so on if the
// path does not exist, or ErrWrongType if the value is not a string.
func (o *Object) GetString(path ...string) (string, error) { return getString(o, path) }

// GetInt returns the whole number at path as an exact int64.
//
//	count, err := root.GetInt("count")
//
// Besides the path errors and ErrWrongType, it returns ErrNotInteger for numbers
// such as 1.5 or 1e3, and ErrNumberOutOfRange if the number does not fit in int64.
func (o *Object) GetInt(path ...string) (int64, error) { return getInt(o, path) }

// GetFloat returns the number at path as a float64. Very large integers may be
// rounded; use GetInt or GetNumber for exact values.
func (o *Object) GetFloat(path ...string) (float64, error) { return getFloat(o, path) }

// GetBool returns the boolean at path.
func (o *Object) GetBool(path ...string) (bool, error) { return getBool(o, path) }

// GetNumber returns the *Number at path, for BigInt, Raw and the other exact
// accessors.
func (o *Object) GetNumber(path ...string) (*Number, error) {
	return getAs[*Number](o, path, "number")
}

// GetObject returns the object at path, so you can read several values from it.
//
//	hero, err := root.GetObject("isekais", "0", "hero")
func (o *Object) GetObject(path ...string) (*Object, error) {
	return getAs[*Object](o, path, "object")
}

// GetArray returns the array at path, ready to loop over with All or Values.
//
//	party, err := root.GetArray("isekais", "0", "party")
func (o *Object) GetArray(path ...string) (*Array, error) {
	return getAs[*Array](o, path, "array")
}

// GetString returns the string at path, with escapes decoded. The first path
// element is an index into this array: arr.GetString("0", "title").
func (a *Array) GetString(path ...string) (string, error) { return getString(a, path) }

// GetInt returns the whole number at path as an exact int64.
func (a *Array) GetInt(path ...string) (int64, error) { return getInt(a, path) }

// GetFloat returns the number at path as a float64.
func (a *Array) GetFloat(path ...string) (float64, error) { return getFloat(a, path) }

// GetBool returns the boolean at path.
func (a *Array) GetBool(path ...string) (bool, error) { return getBool(a, path) }

// GetNumber returns the *Number at path.
func (a *Array) GetNumber(path ...string) (*Number, error) {
	return getAs[*Number](a, path, "number")
}

// GetObject returns the object at path.
func (a *Array) GetObject(path ...string) (*Object, error) {
	return getAs[*Object](a, path, "object")
}

// GetArray returns the array at path.
func (a *Array) GetArray(path ...string) (*Array, error) {
	return getAs[*Array](a, path, "array")
}

// Parse parses data into a JSON value. A nil config means strict JSON.
// It is shorthand for NewParser(data, config) followed by Parse.
func Parse(data []byte, config *ParserConfig) (JSON, error) {
	parser := NewParser(data, config)
	return parser.Parse()
}

// ParseObject parses data and checks that the top-level value is an object.
//
//	root, err := jsonvx.ParseObject(data, nil)
//	title, err := root.GetString("isekais", "0", "title")
//
// It returns the parse error for invalid input, or ErrWrongType if the input is
// valid JSON but not an object.
func ParseObject(data []byte, config *ParserConfig) (*Object, error) {
	node, err := Parse(data, config)
	if err != nil {
		return nil, err
	}

	obj, ok := node.(*Object)
	if !ok {
		return nil, wrongTypeError("object", node, nil)
	}

	return obj, nil
}

// ParseArray parses data and checks that the top-level value is an array.
// It returns the parse error for invalid input, or ErrWrongType if the input is
// valid JSON but not an array.
func ParseArray(data []byte, config *ParserConfig) (*Array, error) {
	node, err := Parse(data, config)
	if err != nil {
		return nil, err
	}

	arr, ok := node.(*Array)
	if !ok {
		return nil, wrongTypeError("array", node, nil)
	}

	return arr, nil
}
