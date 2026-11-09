# JSONVX

`jsonvx` is a highly configurable `JSON` `parser`, `querier`, and `formatter` for Go.

It supports both strict `JSON` (as defined by [ECMA-404](https://datatracker.ietf.org/doc/html/rfc7159)) and up to a more relaxed variant such as [JSON5](https://json5.org/). This makes it ideal for parsing user-generated data, configuration files, or legacy formats that don't fully comply with the JSON specification.

With a single [`ParserConfig`](#configuring-the-parser) struct, `jsonvx` gives you fine-grained control over how `JSON` is parsed. You can enable or disable features like:

- [Hexadecimal numbers `(0xFF)`](#allowextraws)
- [`NaN` and `Infinity` as numeric values](#allownan)
- [Leading plus signs in numbers `(+42)`](#allowleadingplus)
- [Decimal edge cases like `.5` or `5.`](#allowpointedgenumbers)
- [Unquoted object keys `({key: "value"})`](#allowunquoted)
- [Single-quoted strings `('text')`](#allowsinglequotes)
- [Newlines inside strings](#allownewlineinstrings)
- [Escape characters outside the standard set](#allowotherescapechars)
- [Trailing commas in arrays or objects](#allowtrailingcommaarray)
- [Line comments `(// comment)` and Block comments `(/* comment */)`](#allowlinecomments)
- [Extra whitespace in unusual places](#allowextraws)
- [Enable `JSON5`](#allowjson5)

After parsing, `jsonvx` gives you a clean abstract syntax tree `(AST)` that you can either traverse manually or query using the built-in API. Each node in the tree implements a common `JSON interface`, so you can safely `inspect`, `transform`, or `stringify` data as needed.

`jsonvx` is designed for flexibility and correctness — not raw performance. It prioritizes clarity and configurability over speed, making it perfect for tools, linters, formatters, and config loaders where input may vary or include non-standard extensions.

If you need full control over how `JSON` is interpreted and a structured way to work with the result, `jsonvx` is for you.

## Installing

To start using `jsonvx`, install [Go](https://golang.org) and run `go get`:

```sh
$ go get -u github.com/bube054/jsonvx
```

## Quick Start

Get up and running with `jsonvx` in seconds. Just create a new `Parser`, parse your `JSON`, and access fields using a [simple path query system](#query-path-syntax).

```go
package main

import (
	"fmt"

	"github.com/bube054/jsonvx"
)

func main() {
	data := []byte(`{
		"name": {"first": "Tom", "last": "Anderson"},
		"age": 37,
		"children": ["Sara", "Alex", "Jack"],
		"fav.movie": "Deer Hunter",
		"friends": [
			{"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
			{"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
			{"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
		]
	}`)

	parser := jsonvx.NewParser(data, nil)

	node, err := parser.Parse()
	if err != nil {
		panic(fmt.Sprintf("failed to parse JSON: %s", err))
	}

	rootObj, ok := jsonvx.AsObject(node)
	if !ok {
		panic("expected root node to be an object")
	}

	ageNode, err := rootObj.QueryPath("age")
	if err != nil {
		panic(fmt.Sprintf("failed to query 'age' field: %s", err))
	}

	ageNum, ok := jsonvx.AsNumber(ageNode)
	if !ok {
		panic("expected 'age' to be a number")
	}

	ageValue, err := ageNum.Value()
	if err != nil {
		panic(fmt.Sprintf("failed to convert 'age' to numeric value: %s", err))
	}

	fmt.Println(ageValue) // 37
}
```

## Query Path Syntax

Access deeply nested fields in your parsed `JSON` `array` or `object` structure using the `QueryPath` method, which accepts a variadic list of strings to represent the path segments.

Using the `json` data above, we can query for specific values using the `QueryPath` method:

```go
// Get first name
strNode, _ := rootObj.QueryPath("name", "first") // => "Tom"

// get children array
arrayNode, _ := rootObj.QueryPath("children") // => ["Sara", "Alex", "Jack"]

// Get second child
strNode, _ = rootObj.QueryPath("children", "1") // => "Alex"

// Get favorite movie (with dot in key name)
strNode, _ = rootObj.QueryPath("fav.movie") // => "Deer Hunter"

// Get first friend object
objNode, _ := rootObj.QueryPath("friends", "0") // => {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]}

// Get last name of first friend
strNode, _ = rootObj.QueryPath("friends", "0", "last") // => "Murphy"

// Get second social network of second friend
strNode, _ = rootObj.QueryPath("friends", "1", "nets", "1") // => "tw"

// Get age of third friend
numNode, _ := rootObj.QueryPath("friends", "2", "age") // => 47
```

## Configuring The Parser

You can configure the `Parser` using the functional options pattern, allowing you to enable relaxed JSON features individually. By default, the parser is strict (all options disabled), matching the [ECMA-404](https://datatracker.ietf.org/doc/html/rfc7159) specification. To allow non-standard or user-friendly formats (like [JSON5](https://json5.org)), pass options when creating the config:

  ### `AllowExtraWS`:
  Allows extra whitespace characters that are not normally permitted by strict `JSON`.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowExtraWS(true))
  parser := jsonvx.NewParser([]byte("{\"key\":\v\"value\"}"), cfg) // valid Vertical Tab
  parser := jsonvx.NewParser([]byte("{\"key\":\f\"value\"}"), cfg) // valid Form Feed
  parser := jsonvx.NewParser([]byte("{\"key\":\u0085\"value\"}"), cfg) // valid Next Line
  parser := jsonvx.NewParser([]byte("{\"key\":\u00A0\"value\"}"), cfg) // valid No-Break Space
  ```
  ### `AllowHexNumbers`:
  Enables support for hexadecimal numeric literals (e.g., `0x1F`). Upper and lower case work (`0xff`, `0XFF`), as do negative numbers (`-0x10`) and, with [`AllowLeadingPlus`](#allowleadingplus), a leading plus. `Value()` returns the number as a `float64`; like decimal numbers, hex numbers longer than about 16 digits are rounded to the nearest `float64`.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowHexNumbers(true))
  parser := jsonvx.NewParser([]byte("0x1F"), cfg) // valid 31 in hex

  node, _ := parser.Parse()
  num, _ := jsonvx.AsNumber(node)
  value, _ := num.Value() // 31
  ```

  > **Fixed in v1.1.0:** before v1.1.0, hex numbers parsed but `Value()` always returned an error.
  ### `AllowPointEdgeNumbers`:
  Allows numbers like `.5` or `5.` without requiring a digit before/after the decimal point.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowPointEdgeNumbers(true))
  parser := jsonvx.NewParser([]byte(".5"), cfg) // valid pre decimal point number
  parser := jsonvx.NewParser([]byte("5."), cfg) // valid post decimal point number
  ```
  ### `AllowInfinity`:
  Enables the use of `Infinity`, `-Infinity` and `+Infinity` as number values.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowInfinity(true))
  parser := jsonvx.NewParser([]byte("Infinity"), cfg) // valid positive infinity
  parser := jsonvx.NewParser([]byte("-Infinity"), cfg) // valid negative infinity
  parser := jsonvx.NewParser([]byte("+Infinity"), cfg) // valid only if AllowLeadingPlus is enabled
  ```
  ### `AllowNaN`:
  Allows `NaN` (Not-a-Number) as a numeric value.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowNaN(true))
  parser := jsonvx.NewParser([]byte("NaN"), cfg) // valid NaN
  parser := jsonvx.NewParser([]byte("-NaN"), cfg) // valid NaN
  parser := jsonvx.NewParser([]byte("+NaN"), cfg) // valid NaN only if AllowLeadingPlus is enabled
  ```
  ### `AllowLeadingPlus`:
  Permits a leading '+' in numbers (e.g., `+42`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowLeadingPlus(true))
  parser := jsonvx.NewParser([]byte("+99"), cfg) // valid positive number
  ```
  ### `AllowUnquoted`:
  Enables parsing of unquoted object keys (e.g., `{foo: "bar"}`)
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowUnquoted(true))
  parser := jsonvx.NewParser([]byte(`{foo: "bar"}`), cfg) // valid only for unquoted keys and not for unquoted values
  ```
  ### `AllowSingleQuotes`:
  Allows strings to be enclosed in single quotes (`' '`) in addition to double quotes.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowSingleQuotes(true))
  parser := jsonvx.NewParser([]byte(`{'name': 'Tom'}`), cfg) // valid single-quoted string
  ```
  ### `AllowNewlineInStrings`:
  Permits multiple new line characters (`\n`) being escaped.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowNewlineInStrings(true))
  parser := jsonvx.NewParser([]byte(`"hello \
   world"`), cfg) // valid escaped new line
  ```
  ### `AllowOtherEscapeChars`:
  Enables support for escape sequences other than `\\`, `\/`, `\b`, `\n`, `\f`, `\r`, `\t` and Unicode escapes (`\uXXXX`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowOtherEscapeChars(true))
  parser := jsonvx.NewParser([]byte(`"hello\qworld"`), cfg) // valid other escape character
  ```
  ### `AllowTrailingCommaArray`:
  Permits a trailing comma in array literals (e.g., `[1, 2, ]`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowTrailingCommaArray(true))
  parser := jsonvx.NewParser([]byte(`[1, 2, 3, ]`), cfg) // valid array with trailing comma
  ```
  ### `AllowTrailingCommaObject`:
  Permits a trailing comma in object literals (e.g., `{"a": 1,}`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowTrailingCommaObject(true))
  parser := jsonvx.NewParser([]byte(`{"a": 1,}`), cfg) // valid object with trailing comma
  ```
  ### `AllowLineComments`:
  Enables the use of single-line comments (// ...).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowLineComments(true))
  parser := jsonvx.NewParser([]byte(`// comment
  123`), cfg) // valid number after liner comment
  ```
  ### `AllowBlockComments`:
  Enables the use of block comments (/_ ... _/).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowBlockComments(true))
  parser := jsonvx.NewParser([]byte(`/* comment */ 123`), cfg)
  ```
  ### `AllowJSON5`:
  Enables the use of [`JSON5`](https://json5.org) syntax.
  ```go
  parser := jsonvx.NewParser([]byte(`
{
  // comments
  unquoted: 'and you can quote me on that',
  singleQuotes: 'I can use "double quotes" here',
  lineBreaks: "Look, Mom! \
No \\n's!",
  hexadecimal: 0xdecaf,
  leadingDecimalPoint: .8675309, andTrailing: 8675309.,
  positiveSign: +1,
  trailingComma: 'in objects', andIn: ['arrays',],
  "backwardsCompatible": "with JSON",
}
`), jsonvx.JSON5Config())
  ```

## Parsing behavior
Below are examples of how to `parse`, `traverse` and `retrieve` values from the parsed `JSON` input using the `Parser`.

### Object
- `Objects` are stored as a `slice` of `jsonvx.KeyValue` struct.
- keys are stored as `[]byte` and `sorted` `lexicographically` for `log(n)` value retrieval.
- values are stored as `jsonvx.JSON`
- Each key-value pair is accessible using the `ForEach` method.
- Mixed value types (e.g., `numbers`, `strings`, `objects`) are obviously supported.
- Duplicate keys are included, but the first key-value pair is gotten with the `QueryPath` method.

This example shows how to parse a `JSON` object using `jsonvx`, cast it to an `jsonvx.Object` node, and iterate over key-value pairs using the built-in `ForEach` method.

```go
parser := jsonvx.NewParser([]byte(`{"name": "Alice", "age": 30}`), jsonvx.NewParserConfig())

// Parse the JSON input
node, err := parser.Parse()
if err != nil {
	panic(fmt.Sprintf("failed to parse JSON: %s", err))
}

// Cast the parsed node to an ObjectNode
objNode, ok := jsonvx.AsObject(node)
if !ok {
	panic("expected root node to be an object")
}

// Iterate over key-value pairs using ForEach
objNode.ForEach(func(key []byte, value jsonvx.JSON, object *jsonvx.Object) {
	fmt.Printf("Key: %s, Value: %v\n", string(key), value)
})
```

### Array

- Items are stored as a `slice` of `jsonvx.JSON`
- Each item is accessible using the `ForEach` method.

This example demonstrates how to parse a `JSON` `array` using `jsonvx`, cast it to an `jsonvx.Array` node, and iterate over its elements using the built-in `ForEach` method.

```go
parser := jsonvx.NewParser([]byte(`[1, 2, 3]`), jsonvx.NewParserConfig())

// Parse the JSON input
node, err := parser.Parse()
if err != nil {
	panic(fmt.Sprintf("failed to parse JSON: %s", err))
}

// Cast the parsed node to an ArrayNode
arrNode, ok := jsonvx.AsArray(node)
if !ok {
	panic("expected root node to be an array")
}

// Iterate over each item using ForEach
arrNode.ForEach(func(item jsonvx.JSON, index int, array *jsonvx.Array) {
	fmt.Printf("Item %d: %v\n", index, item)
})
```

### String

- JSON string values are parsed as `jsonvx.String`.
- `Value()` returns the raw Go` string value.

This example demonstrates how to parse a `JSON` `string` using `jsonvx`, cast it to an `jsonvx.String` node, and get its `string` value using the built-in `Value` method.

```go
parser := jsonvx.NewParser([]byte(`"Hello, World!"`), jsonvx.NewParserConfig())

// Parse the JSON input
node, err := parser.Parse()
if err != nil {
	panic(fmt.Sprintf("failed to parse JSON: %s", err))
}

// Cast the parsed node to a StringNode
strNode, ok := jsonvx.AsString(node)
if !ok {
	panic("expected root node to be a string")
}

// Extract the underlying Go string value
strValue, err := strNode.Value()
if err != nil {
	panic(fmt.Sprintf("failed to extract string value: %s", err))
}

fmt.Println(strValue) // Output: Hello, World!
```

### Number

- JSON numbers are parsed as a `jsonvx.Number`.
- `Value()` returns the number as a Go `float64`. A `float64` holds about 16 significant digits, so longer numbers are rounded: `9007199254740993` becomes `9007199254740992`.
- When every digit matters, such as for IDs, use the exact methods instead:
  - `Int64()` returns an exact `int64`. It returns `ErrNotInteger` for numbers written with a decimal point or exponent (`1.5`, `1.0`, `1e3`), and `ErrNumberOutOfRange` for whole numbers that don't fit in an `int64`.
  - `BigInt()` returns an exact `*big.Int` for whole numbers of any size.
  - `Raw()` returns the number exactly as written, such as `"1.50"` or `"0x1F"`.
- `Integer`, `Float`, `SciNot`, `Hex`, `Infinity` and ironically `NaN` number types are supported. Whole numbers of any length are `Integer`.

This example demonstrates how to parse a `JSON` `number` using `jsonvx`, cast it to an `jsonvx.Number` node, and get `float64` its value using the built-in `Value` method.

```go
parser := jsonvx.NewParser([]byte("123456"), jsonvx.NewParserConfig())

// Parse the JSON input
node, err := parser.Parse()
if err != nil {
	panic(fmt.Sprintf("failed to parse JSON: %s", err))
}

// Cast the parsed node to a NumberNode
numNode, ok := jsonvx.AsNumber(node)
if !ok {
	panic("expected root node to be a number")
}

// Extract the underlying Go float64 value
numValue, err := numNode.Value()
if err != nil {
	panic(fmt.Sprintf("failed to extract number value: %s", err))
}

fmt.Println(numValue) // Output: 123456
```

For large IDs, `Value()` rounds while `Int64()`, `BigInt()` and `Raw()` keep every digit:

```go
parser := jsonvx.NewParser([]byte(`{"user_id": 9007199254740993}`), nil)
node, _ := parser.Parse()
obj, _ := jsonvx.AsObject(node)
idNode, _ := obj.QueryPath("user_id")
id, _ := jsonvx.AsNumber(idNode)

approx, _ := id.Value() // 9.007199254740992e+15 (rounded)
exact, _ := id.Int64()  // 9007199254740993
big, _ := id.BigInt()   // 9007199254740993
raw, _ := id.Raw()      // "9007199254740993"
```

> **Changed in v1.1.0:** `Int64()`, `BigInt()` and `Raw()` are new. Whole numbers too large for an `int64` are now classified as `Integer` instead of `Float` (their `Value()` is unchanged), and whole numbers too long for a `float64` now parse instead of being rejected.

### Boolean

- JSON booleans are parsed as a `jsonvx.Boolean`.
- `Value()` returns the raw Go `bool` value.

This example demonstrates how to parse a `JSON` `boolean` using `jsonvx`, cast it to an `jsonvx.Boolean` node, and get `bool` its value using the built-in `Value` method.

```go
parser := jsonvx.NewParser([]byte("true"), jsonvx.NewParserConfig())

// Parse the JSON input
node, err := parser.Parse()
if err != nil {
	panic(fmt.Sprintf("failed to parse JSON: %s", err))
}

// Cast the parsed node to a BooleanNode
boolNode, ok := jsonvx.AsBoolean(node)
if !ok {
	panic("expected root node to be a boolean")
}

// Extract the underlying Go value
trueValue, err := boolNode.Value()
if err != nil {
	panic(fmt.Sprintf("failed to extract boolean value: %s", err))
}

fmt.Println(trueValue) // Output: true
```

### Null

- JSON `null` values are parsed as a `jsonvx.Null`.
- `Value()` returns the raw Go `nil` value.

This example demonstrates how to parse a `JSON` `null` using `jsonvx`, cast it to an `jsonvx.Null` node, and get `nil` its value using the built-in `Value` method.

```go
parser := jsonvx.NewParser([]byte("null"), jsonvx.NewParserConfig())

// Parse the JSON input
node, err := parser.Parse()
if err != nil {
	panic("failed to parse JSON: %s", err)
}

// Cast the parsed node to a NullNode
nullNode, ok := jsonvx.AsNull(node)
if !ok {
  panic("expected root node to be null")
}

// Extract the underlying Go value
nilValue, _ := nullNode.Value()
fmt.Println(nilValue) // Output: <nil>
```

### Comments

- `Comments` are ignored during parsing.
- A `JSON` input with only comments and no data will result in a parse error.

```go
parser := jsonvx.NewParser([]byte("/* Block Comment */"), jsonvx.NewParserConfig(
	jsonvx.WithAllowBlockComments(true), jsonvx.WithAllowLineComments(true),
))

// parse the JSON
node, err := parser.Parse()
if err != nil {
	panic(fmt.Sprintf("failed to parse JSON: %s", err))
}
```

## Maintainers

- [bube054](https://github.com/bube054) - **Attah Gbubemi David (author)**

## Other helpful projects

- [ginvalidator](https://github.com/bube054/ginvalidator)
- [validatorgo](https://github.com/bube054/validatorgo)

## License

This project is licensed under the [MIT](https://opensource.org/license/mit). See the [LICENSE](https://github.com/bube054/jsonvx/blob/master/LICENSE) file for details.
