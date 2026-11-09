// Package jsonvx is a highly configurable JSON parser and querier for Go. Formatting (writing
// parsed data back out as JSON) is coming soon.
//
// It supports both strict JSON (as defined by [ECMA-404]) and up to a more relaxed variant
// such as [JSON5]. This makes it ideal for parsing user-generated data, configuration files,
// or legacy formats that don't fully comply with the JSON specification.
//
// With a single [ParserConfig] struct, jsonvx gives you fine-grained control over how JSON is parsed.
// You can enable or disable features like:
//
//   - Hexadecimal numbers (0xFF)
//   - NaN and Infinity as numeric values
//   - Leading plus signs in numbers (+42)
//   - Decimal edge cases like .5 or 5.
//   - Unquoted object keys ({key: "value"})
//   - Single-quoted strings ('text')
//   - Newlines inside strings
//   - Escape characters outside the standard set
//   - Trailing commas in arrays or objects
//   - Line comments (// comment) and Block comments (/* comment */)
//   - Extra whitespace in unusual places
//   - Rejecting duplicate object keys (by default the last value wins)
//   - Enable JSON5
//
// It also guards against hostile input with a nesting depth limit ([ParserConfig.MaxDepth],
// [DefaultMaxDepth] by default), so deeply nested data returns [ErrJSONTooDeep] instead of
// crashing your program.
//
// After parsing, jsonvx gives you a clean abstract syntax tree (AST) that you can either traverse manually
// or query using the built-in API. Each node in the tree implements a common JSON interface, so you can safely
// inspect and transform data as needed. Writing it back out as JSON is coming soon.
//
// jsonvx is designed for flexibility and correctness — not raw performance. It prioritizes clarity and
// configurability over speed, making it perfect for tools, linters, and config loaders where
// input may vary or include non-standard extensions.
//
// If you need full control over how JSON is interpreted and a structured way to work with the result,
// jsonvx is for you.
//
// [ECMA-404]: https://ecma-international.org/publications-and-standards/standards/ecma-404/
// [JSON5]: https://json5.org/
package jsonvx

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Common errors for type assertion and query path resolution.
var (
	ErrNotString  = errors.New("value is not a JSON string")
	ErrNotNumber  = errors.New("value is not a JSON number")
	ErrNotBoolean = errors.New("value is not a JSON boolean")
	ErrNotNull    = errors.New("value is not null")

	ErrInvalidQueryKey   = errors.New("invalid query key")
	ErrKeyNotFound       = errors.New("key not found")
	ErrExpectedIndex     = errors.New("invalid query key, expected integer index")
	ErrIndexOutOfRange   = errors.New("index out of range")
	ErrEmptyArray        = errors.New("array is empty")
	ErrInvalidJSONType   = errors.New("invalid JSON type")
	ErrQueryExceedsDepth = errors.New("query exceeds depth for scalar value")
)

// nilString is what String returns for a nil or empty node, matching how fmt prints nil pointers.
const nilString = "<nil>"

// JSON is a common interface implemented by all JSON types (Null, Boolean, etc.).
type JSON interface {
	fmt.Stringer
	Equal(JSON) bool
}

// Null represents a JSON null value.
type Null struct {
	Token *Token
}

// newNull creates a new *Null value, optionally invoking a callback.
func newNull(Token *Token, cb func()) *Null {
	if cb != nil {
		cb()
	}

	return &Null{Token: Token}
}

func (n *Null) String() string {
	return "\033[1mnull\033[0m"
}

// Value returns nil if the token is valid, or an error otherwise.
func (n *Null) Value() (any, error) {
	if n == nil || n.Token == nil {
		return nil, ErrNotNull
	}

	return nil, nil
}

func (n *Null) Equal(n2 JSON) bool {
	if n == nil || n2 == nil {
		return n == n2
	}

	other, ok := AsNull(n2)

	if !ok {
		return false
	}

	return n.Token.Equal(other.Token)
}

// AsNull safely casts a JSON to a *Null.
func AsNull(j JSON) (*Null, bool) {
	null, ok := j.(*Null)
	return null, ok
}

// Boolean represents a JSON boolean (true or false).
type Boolean struct {
	Token *Token
}

// newBoolean creates a new *Boolean value, optionally invoking a callback
func newBoolean(Token *Token, cb func()) *Boolean {
	if cb != nil {
		cb()
	}

	return &Boolean{Token: Token}
}

func (b *Boolean) String() string {
	if b == nil || b.Token == nil {
		return nilString
	}

	if b.Token.SubKind == TRUE {
		return "\033[1mtrue\033[0m"
	} else {
		return "\033[1mfalse\033[0m"
	}
}

// Value returns the boolean value or an error if the token is invalid.
func (b *Boolean) Value() (bool, error) {
	if b == nil || b.Token == nil {
		return false, ErrNotBoolean
	}

	val := b.Token.Value()

	if val == nil {
		return false, ErrNotBoolean
	}

	boolVal, ok := val.(bool)

	if !ok {
		return false, ErrNotBoolean
	}

	return boolVal, nil
}

func (b *Boolean) Equal(b2 JSON) bool {
	if b == nil || b2 == nil {
		return b == b2
	}

	other, ok := AsBoolean(b2)

	if !ok {
		return false
	}

	return b.Token.Equal(other.Token)
}

// AsBoolean safely casts a JSON to a *Boolean.
func AsBoolean(j JSON) (*Boolean, bool) {
	boolean, ok := j.(*Boolean)
	return boolean, ok
}

// String represents a JSON string.
type String struct {
	Token *Token
}

// newString creates a new *String value, optionally invoking a callback
func newString(Token *Token, cb func()) *String {
	if cb != nil {
		cb()
	}

	return &String{Token: Token}
}

func (s *String) String() string {
	value, err := s.Value()
	if err != nil {
		return nilString
	}

	return value
}

// Value returns the decoded string, with escape sequences such as \n, \" and \u00e9
// turned into the characters they stand for, or an error if invalid.
// Use Raw to get the string exactly as written in the input.
func (s *String) Value() (string, error) {
	if s == nil || s.Token == nil {
		return "", ErrNotString
	}

	val := s.Token.Value()

	if val == nil {
		return "", ErrNotString
	}

	strVal, ok := val.(string)

	if !ok {
		return "", ErrNotString
	}

	return strVal, nil
}

// Raw returns the string exactly as written in the input, without the surrounding
// quotes and without decoding escape sequences, so "caf\u00e9" stays `caf\u00e9`.
// Before v1.1.0, Value returned this raw form.
func (s *String) Raw() (string, error) {
	if s == nil || s.Token == nil || s.Token.Kind != STRING {
		return "", ErrNotString
	}

	if s.Token.SubKind == IDENT {
		return string(s.Token.Literal), nil
	}

	return string(stringContent(s.Token.Literal)), nil
}

func (s *String) Equal(s2 JSON) bool {
	if s == nil || s2 == nil {
		return s == s2
	}

	other, ok := AsString(s2)

	if !ok {
		return false
	}

	return s.Token.Equal(other.Token)
}

// AsString safely casts a JSON to a *String.
func AsString(j JSON) (*String, bool) {
	str, ok := j.(*String)
	return str, ok
}

// Number represents a JSON number (integer, float, hex, etc.).
type Number struct {
	Token *Token
}

// newNumber creates a new *Number value, optionally invoking a callback
func newNumber(Token *Token, cb func()) *Number {
	if cb != nil {
		cb()
	}

	return &Number{Token: Token}
}

func (n *Number) String() string {
	if n == nil || n.Token == nil {
		return nilString
	}

	return string(n.Token.Literal)
}

// Value returns the number as a float64, depending on its subtype.
//
// A float64 holds about 16 significant digits, so longer numbers are rounded to the
// nearest float64: 9007199254740993 becomes 9007199254740992. When every digit
// matters, such as for IDs, use Int64, BigInt or Raw instead.
func (n *Number) Value() (float64, error) {
	if n == nil || n.Token == nil {
		return 0, ErrNotNumber
	}

	switch n.Token.SubKind {
	case INTEGER:
		numVal, err := ToInt(n.Token.Literal)
		if err == nil {
			return float64(numVal), nil
		}
		// Too large for int64: use the nearest float64. Int64 and BigInt give exact values.
		floatVal, err := ToFloat(n.Token.Literal)
		if err != nil {
			return 0, ErrNotNumber
		}
		return floatVal, nil
	case HEX:
		// Hex literals (e.g. 0x1F, -0xFF) can have any number of digits. Like other
		// numbers, values too long for float64 to hold exactly are rounded to the
		// nearest float64; values beyond float64's range are an error.
		floatVal, ok := hexToFloat64(n.Token.Literal)
		if !ok || math.IsInf(floatVal, 0) {
			return 0, ErrNotNumber
		}
		return floatVal, nil
	case FLOAT, SCI_NOT:
		numVal, err := ToFloat(n.Token.Literal)
		if err != nil {
			return 0, ErrNotNumber
		}
		return numVal, nil
	case INF:
		if bytes.Equal(n.Token.Literal, []byte("Infinity")) ||
			bytes.Equal(n.Token.Literal, []byte("+Infinity")) {
			return math.Inf(1), nil
		}
		if bytes.Equal(n.Token.Literal, []byte("-Infinity")) {
			return math.Inf(-1), nil
		}
		return 0, ErrNotNumber
	case NaN:
		if bytes.Equal(n.Token.Literal, []byte("NaN")) ||
			bytes.Equal(n.Token.Literal, []byte("+NaN")) ||
			bytes.Equal(n.Token.Literal, []byte("-NaN")) {
			return math.NaN(), nil
		}
		return 0, ErrNotNumber

	default:
		return 0, ErrNotNumber
	}
}

func (n *Number) Equal(n2 JSON) bool {
	if n == nil || n2 == nil {
		return n == n2
	}

	other, ok := AsNumber(n2)

	if !ok {
		return false
	}

	return n.Token.Equal(other.Token)
}

// AsNumber safely casts a JSON to a *Number.
func AsNumber(j JSON) (*Number, bool) {
	number, ok := j.(*Number)
	return number, ok
}

// Array represents a JSON array.
type Array struct {
	Items []JSON
}

// newArray creates a new *Array value, optionally invoking a callback
func newArray(items []JSON, cb func()) *Array {
	if cb != nil {
		cb()
	}

	return &Array{Items: items}
}

func (a *Array) String() string {
	if a == nil {
		return nilString
	}

	var builder strings.Builder
	builder.WriteString("[")
	for i, item := range a.Items {
		if i == len(a.Items)-1 {
			builder.WriteString(fmt.Sprintf("%v", item))
		} else {
			builder.WriteString(fmt.Sprintf("%v, ", item))
		}
	}
	builder.WriteString("]")
	return builder.String()
}

func (a *Array) Len() int {
	return len(a.Items)
}

// QueryPath retrieves a nested item using a slice of string indices.
func (a *Array) QueryPath(paths ...string) (JSON, error) {
	if len(paths) == 0 {
		return a, nil
	}

	indexStr := paths[0]
	index, err := strconv.Atoi(indexStr)
	if err != nil {
		return nil, fmt.Errorf("%w: %q", ErrExpectedIndex, indexStr)
	}

	if a.Len() == 0 {
		return nil, ErrEmptyArray
	}

	if index < 0 || index >= a.Len() {
		return nil, fmt.Errorf("%w: %d", ErrIndexOutOfRange, index)
	}

	item := a.Items[index]

	rest := paths[1:]

	switch val := item.(type) {
	case *Null, *Boolean, *Number, *String:
		if len(rest) > 0 {
			return nil, ErrQueryExceedsDepth
		}

		return val, nil
	case *Array:
		return val.QueryPath(rest...)
	case *Object:
		return val.QueryPath(rest...)
	default:
		return nil, ErrInvalidJSONType
	}
}

// ArrayCallback defines the function signature for iterating over items in a JSON array.
// - item: the current element
// - index: the index of the element
// - array: the array being iterated
type ArrayCallback func(item JSON, index int, array *Array)

// ForEach calls the given callback for each item in the array.
// It provides the item, its index, and the array itself.
func (a *Array) ForEach(cb ArrayCallback) {
	for i, item := range a.Items {
		cb(item, i, a)
	}
}

func (a *Array) Equal(a2 JSON) bool {
	if a == nil || a2 == nil {
		return a == a2
	}

	other, ok := AsArray(a2)

	if !ok {
		return false
	}

	if a.Len() != other.Len() {
		return false
	}

	for i, item := range a.Items {
		item2 := other.Items[i]

		if !item.Equal(item2) {
			return false
		}

	}

	return true
}

// AsArray safely casts a JSON to a *Array.
func AsArray(j JSON) (*Array, bool) {
	arr, ok := j.(*Array)
	return arr, ok
}

// KeyValue represents a key-value pair in a JSON object.
type KeyValue struct {
	key   []byte
	value JSON
}

func newKeyValue(key []byte, value JSON) KeyValue {
	return KeyValue{key: key, value: value}
}

func (kv *KeyValue) Equal(kv2 *KeyValue) bool {
	if kv == nil || kv2 == nil {
		return kv == kv2
	}

	if !bytes.Equal(kv.key, kv2.key) {
		return false
	}

	return kv.value.Equal(kv2.value)
}

// Object represents a JSON object.
//
// Properties are kept in the order they appear in the input. A repeated key keeps
// the position where it first appeared and holds the last value written for it.
type Object struct {
	Properties []KeyValue

	// index maps each key to its position in Properties, so lookups do not have to
	// search. It is built by the parser. If it is nil, or no longer matches
	// Properties because the slice was changed, lookups fall back to a linear scan.
	index map[string]int
}

// newObject creates a new *Object value, optionally invoking a callback
func newObject(properties []KeyValue, index map[string]int, cb func()) *Object {
	if cb != nil {
		cb()
	}

	return &Object{Properties: properties, index: index}
}

// lookup returns the position of key in Properties, or -1 if it is not present.
func (o *Object) lookup(key []byte) int {
	if i, ok := o.index[string(key)]; ok && i < len(o.Properties) && bytes.Equal(o.Properties[i].key, key) {
		return i
	}

	// No usable index: scan from the end, so the last copy of a repeated key wins,
	// matching how the parser resolves duplicates.
	for i := len(o.Properties) - 1; i >= 0; i-- {
		if bytes.Equal(o.Properties[i].key, key) {
			return i
		}
	}

	return -1
}

// Keys returns the object's keys in the order they appear in the input.
// Use sort.Strings on the result for alphabetical order.
func (o *Object) Keys() []string {
	keys := make([]string, len(o.Properties))
	for i, prop := range o.Properties {
		keys[i] = string(prop.key)
	}

	return keys
}

func (o *Object) String() string {
	if o == nil {
		return nilString
	}

	var builder strings.Builder
	builder.WriteString("{")
	length := len(o.Properties)
	for ind, kv := range o.Properties {
		if ind == length-1 {
			builder.WriteString(fmt.Sprintf("%s: %v", kv.key, kv.value))
		} else {
			builder.WriteString(fmt.Sprintf("%s: %v, ", kv.key, kv.value))
		}
	}
	builder.WriteString("}")
	return builder.String()
}

func (o *Object) Len() int {
	return len(o.Properties)
}

// QueryPath retrieves a nested value via key-based path traversal.
func (o *Object) QueryPath(paths ...string) (JSON, error) {
	if len(paths) == 0 {
		return o, nil
	}

	keyStr := paths[0]

	index := o.lookup([]byte(keyStr))
	if index < 0 {
		return nil, fmt.Errorf("%w: %q", ErrKeyNotFound, keyStr)
	}

	item := o.Properties[index]
	rest := paths[1:]

	switch val := item.value.(type) {
	case *Null, *Boolean, *Number, *String:
		if len(rest) > 0 {
			return nil, ErrQueryExceedsDepth
		}
		return val, nil
	case *Array:
		return val.QueryPath(rest...)
	case *Object:
		return val.QueryPath(rest...)
	default:
		return nil, ErrInvalidJSONType
	}
}

// ObjectCallback defines the function signature for iterating over properties in a JSON object.
// - key: the property's key as a byte slice
// - value: the property's value
// - object: the object being iterated
type ObjectCallback func(key []byte, value JSON, object *Object)

// ForEach calls the given callback for each key-value pair in the object,
// in the order they appear in the input.
// It provides the key, value, and the object itself.
func (o *Object) ForEach(cb ObjectCallback) {
	for _, prop := range o.Properties {
		cb(prop.key, prop.value, o)
	}
}

// Equal reports whether o2 is an object with the same keys and equal values.
// Key order does not matter.
func (o *Object) Equal(o2 JSON) bool {
	if o == nil || o2 == nil {
		return JSON(o) == o2
	}

	other, ok := AsObject(o2)

	if !ok || other == nil {
		return false
	}

	if o.Len() != other.Len() {
		return false
	}

	for _, prop := range o.Properties {
		i := other.lookup(prop.key)
		if i < 0 {
			return false
		}

		prop2 := other.Properties[i]
		if !prop.Equal(&prop2) {
			return false
		}
	}

	return true
}

// AsObject safely casts a JSON to a *Object.
func AsObject(j JSON) (*Object, bool) {
	obj, ok := j.(*Object)
	return obj, ok
}
