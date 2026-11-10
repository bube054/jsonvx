<p align="center">
  <img src="assets/logo-wordmark.svg" alt="jsonvx logo" width="420">
</p>

# JSONVX

`jsonvx` is a highly configurable `JSON` `parser` and `querier` for Go.

> **Formatting coming soon:** writing parsed data back out as valid `JSON` (compact or indented, including converting `JSON5` to standard `JSON`) is planned for an upcoming release.

It supports both strict `JSON` (as defined by [ECMA-404](https://ecma-international.org/publications-and-standards/standards/ecma-404/)) and up to a more relaxed variant such as [JSON5](https://json5.org/). This makes it ideal for parsing user-generated data, configuration files, or legacy formats that don't fully comply with the JSON specification.

With a single [`ParserConfig`](#configuring-the-parser) struct, `jsonvx` gives you fine-grained control over how `JSON` is parsed. You can enable or disable features like:

- [Hexadecimal numbers `(0xFF)`](#allowhexnumbers)
- [`NaN` and `Infinity` as numeric values](#allownan)
- [Leading plus signs in numbers `(+42)`](#allowleadingplus)
- [Decimal edge cases like `.5` or `5.`](#allowpointedgenumbers)
- [Unquoted object keys `({key: "value"})`](#allowunquoted)
- [Single-quoted strings `('text')`](#allowsinglequotes)
- [Newlines inside strings](#allownewlineinstrings)
- [Escape characters outside the standard set](#allowotherescapechars)
- [Raw control characters such as tabs inside strings](#allowcontrolcharsinstrings)
- [Trailing commas in arrays or objects](#allowtrailingcommaarray)
- [Line comments `(// comment)` and Block comments `(/* comment */)`](#allowlinecomments)
- [Extra whitespace in unusual places](#allowextraws)
- [Rejecting duplicate object keys](#disallowduplicatekeys)
- [Enable `JSON5`](#allowjson5)

It also guards against hostile input with a [nesting depth limit](#maxdepth), on by default, so deeply nested data returns an error instead of crashing your program.

In its default strict mode, `jsonvx` passes every accept and reject case in [JSONTestSuite](https://github.com/nst/JSONTestSuite), the standard conformance suite for JSON parsers; the cases run as part of `go test`.

After parsing, `jsonvx` gives you a clean abstract syntax tree `(AST)` that you can either traverse manually or query using the built-in API. Each node in the tree implements a common `JSON interface`, so you can safely `inspect` and `transform` data as needed. Writing it back out as `JSON` is [coming soon](#jsonvx).

`jsonvx` is designed for flexibility and correctness — not raw performance. It prioritizes clarity and configurability over speed, making it perfect for tools, linters, and config loaders where input may vary or include non-standard extensions.

If you need full control over how `JSON` is interpreted and a structured way to work with the result, `jsonvx` is for you.

## Installing

To start using `jsonvx`, install [Go](https://golang.org) and run `go get`:

```sh
$ go get -u github.com/bube054/jsonvx
```

## Quick Start

Get up and running with `jsonvx` in seconds. Parse your `JSON` with `ParseObject`, then read values in one call with the [typed getters](#typed-getters).

```go
package main

import (
	"fmt"

	"github.com/bube054/jsonvx"
)

func main() {
	// bube's favourite isekais, ranked.
	data := []byte(`{
		"fan": "bube",
		"count": 3,
		"genres": ["isekai", "fantasy", "comedy"],
		"best.of.all": "Re:Zero",
		"isekais": [
			{"rank": 1, "title": "Re:Zero", "hero": {"first": "Subaru", "last": "Natsuki"}, "party": ["Emilia", "Rem", "Ram", "Beatrice"]},
			{"rank": 2, "title": "Mushoku Tensei", "hero": {"first": "Rudeus", "last": "Greyrat"}, "party": ["Roxy", "Sylphiette", "Eris"]},
			{"rank": 3, "title": "KonoSuba", "hero": {"first": "Kazuma", "last": "Satou"}, "party": ["Aqua", "Megumin", "Darkness"]}
		]
	}`)

	// Parse, and check the top-level value is an object.
	root, err := jsonvx.ParseObject(data, nil)
	if err != nil {
		panic(err)
	}

	// Read values in one call each.
	count, err := root.GetInt("count")
	if err != nil {
		panic(err)
	}
	fmt.Println(count) // 3

	title, _ := root.GetString("isekais", "0", "title")
	fmt.Println(title) // Re:Zero

	// Loop over an array.
	party, _ := root.GetArray("isekais", "2", "party")
	for i, member := range party.All() {
		fmt.Println(i, member) // 0 Aqua, 1 Megumin, 2 Darkness
	}
}
```

Pass a config as the second argument for relaxed formats, for example `jsonvx.ParseObject(data, jsonvx.JSON5Config())`. Use `ParseArray` when the top-level value is an array, or `Parse` when it could be anything.

> **New in v1.4.0:** `Parse`, `ParseObject`, `ParseArray` and the `Get*` methods. Before v1.4.0, the same program needed `NewParser`, `Parse`, `AsObject`, `QueryPath`, `AsNumber` and `Value`. That longer form still works and is shown below.

## Typed Getters

`*Object` and `*Array` have one getter per type. Each takes the same path as [`QueryPath`](#query-path-syntax) and returns the value already converted:

| Method | Returns | Notes |
| --- | --- | --- |
| `GetString(path...)` | `string` | Escapes are decoded. |
| `GetInt(path...)` | `int64` | Exact. Errors on `1.5` or `1e3` (`ErrNotInteger`) and on numbers past int64 (`ErrNumberOutOfRange`). |
| `GetFloat(path...)` | `float64` | Very large integers may be rounded. |
| `GetBool(path...)` | `bool` | |
| `GetNumber(path...)` | `*Number` | For `BigInt()`, `Raw()` and other exact access. |
| `GetObject(path...)` | `*Object` | To read several values from one place. |
| `GetArray(path...)` | `*Array` | To loop with `All()` or `Values()`. |

```go
hero, _ := root.GetObject("isekais", "1", "hero")
first, _ := hero.GetString("first") // Rudeus
last, _ := hero.GetString("last")   // Greyrat

isekais, _ := root.GetArray("isekais")
rank, _ := isekais.GetInt("2", "rank") // 3, on an array the first path element is an index
```

Errors say what went wrong and where, and work with `errors.Is`:

```go
_, err := root.GetString("count")
// wrong JSON type: expected string, got number at ["count"]
errors.Is(err, jsonvx.ErrWrongType) // true

_, err = root.GetString("isekais", "0", "villain")
// key not found: "villain" at ["isekais" "0" "villain"]
errors.Is(err, jsonvx.ErrKeyNotFound) // true
```

Calling a getter on a nil `*Object` or `*Array`, for example after ignoring an error from `GetObject`, returns an error wrapping `ErrInvalidJSONType` instead of panicking.

## Query Path Syntax

A path is a list of strings, one per step: an object key, or an array index written as a string (`"0"`). Keys are used exactly as written, so a key with dots like `"best.of.all"` needs no escaping. The typed getters use these paths, and so does `QueryPath`, which returns the raw `jsonvx.JSON` node instead of a converted value.

Using the `json` data above, here is what each path points to:

```go
// Get the title of the number one isekai
strNode, _ := root.QueryPath("isekais", "0", "title") // => "Re:Zero"

// Get the genres array
arrayNode, _ := root.QueryPath("genres") // => ["isekai", "fantasy", "comedy"]

// Get the second genre
strNode, _ = root.QueryPath("genres", "1") // => "fantasy"

// Get the best of all (with dots in the key name)
strNode, _ = root.QueryPath("best.of.all") // => "Re:Zero"

// Get the second isekai
objNode, _ := root.QueryPath("isekais", "1") // => {"rank": 2, "title": "Mushoku Tensei", "hero": {"first": "Rudeus", "last": "Greyrat"}, "party": ["Roxy", "Sylphiette", "Eris"]}

// Get the last name of the second isekai's hero
strNode, _ = root.QueryPath("isekais", "1", "hero", "last") // => "Greyrat"

// Get the second party member of the third isekai
strNode, _ = root.QueryPath("isekais", "2", "party", "1") // => "Megumin"

// Get the rank of the third isekai
numNode, _ := root.QueryPath("isekais", "2", "rank") // => 3
```

`QueryPath` is useful when you don't know a value's type ahead of time. To get a Go value out of a node, cast it with `AsString`, `AsNumber` and the other `As*` functions, then call `Value()`. This is the long form the typed getters replace:

```go
parser := jsonvx.NewParser(data, nil)
node, err := parser.Parse()
if err != nil {
	panic(err)
}
root, _ := jsonvx.AsObject(node)

countNode, _ := root.QueryPath("count")
countNum, _ := jsonvx.AsNumber(countNode)
count, _ := countNum.Int64() // 3, same as root.GetInt("count")
```

## Configuring The Parser

You can configure the `Parser` using the functional options pattern, allowing you to enable relaxed JSON features individually. By default, the parser is strict (all options disabled), matching the [ECMA-404](https://ecma-international.org/publications-and-standards/standards/ecma-404/) specification. To allow non-standard or user-friendly formats (like [JSON5](https://json5.org)), pass options when creating the config:

  ### `AllowExtraWS`:
  Allows extra whitespace characters that are not normally permitted by strict `JSON`.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowExtraWS(true))
  parser := jsonvx.NewParser([]byte("{\"hero\":\v\"Subaru\"}"), cfg) // valid Vertical Tab
  parser := jsonvx.NewParser([]byte("{\"hero\":\f\"Subaru\"}"), cfg) // valid Form Feed
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
  parser := jsonvx.NewParser([]byte("+9999"), cfg) // valid positive number (EXPLOSION!)
  ```
  ### `AllowUnquoted`:
  Enables parsing of unquoted object keys (e.g., `{hero: "Kazuma"}`)
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowUnquoted(true))
  parser := jsonvx.NewParser([]byte(`{hero: "Kazuma"}`), cfg) // valid only for unquoted keys and not for unquoted values
  ```
  ### `AllowSingleQuotes`:
  Allows strings to be enclosed in single quotes (`' '`) in addition to double quotes.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowSingleQuotes(true))
  parser := jsonvx.NewParser([]byte(`{'name': 'Megumin'}`), cfg) // valid single-quoted string
  ```
  ### `AllowNewlineInStrings`:
  Permits multiple new line characters (`\n`) being escaped.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowNewlineInStrings(true))
  parser := jsonvx.NewParser([]byte(`"Starting life in \
another world"`), cfg) // valid escaped new line
  ```
  ### `AllowOtherEscapeChars`:
  Enables support for escape sequences other than `\\`, `\/`, `\b`, `\n`, `\f`, `\r`, `\t` and Unicode escapes (`\uXXXX`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowOtherEscapeChars(true))
  parser := jsonvx.NewParser([]byte(`"another\qworld"`), cfg) // valid other escape character
  ```
  ### `AllowControlCharsInStrings`:
  Allows raw control characters (bytes `0x00`-`0x1F`, such as a tab) inside strings, as `JSON5` does. Strict `JSON` requires them to be escaped (`\t`). Raw line breaks are never allowed inside strings, in `JSON` or `JSON5`; use [`AllowNewlineInStrings`](#allownewlineinstrings) for an escaped line break.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowControlCharsInStrings(true))
  parser := jsonvx.NewParser([]byte("\"Rem\tRam\""), cfg) // valid, raw tab inside the string
  parser := jsonvx.NewParser([]byte("\"Rem\nRam\""), cfg) // invalid, raw line break inside the string
  ```

  > **Changed in v1.2.0:** before v1.2.0, strict mode accepted raw control characters inside strings. If you relied on that, enable this option.
  ### `AllowTrailingCommaArray`:
  Permits a trailing comma in array literals (e.g., `[1, 2, ]`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowTrailingCommaArray(true))
  parser := jsonvx.NewParser([]byte(`["Aqua", "Megumin", "Darkness", ]`), cfg) // valid array with trailing comma
  ```
  ### `AllowTrailingCommaObject`:
  Permits a trailing comma in object literals (e.g., `{"rank": 1,}`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowTrailingCommaObject(true))
  parser := jsonvx.NewParser([]byte(`{"rank": 1,}`), cfg) // valid object with trailing comma
  ```
  ### `AllowLineComments`:
  Enables the use of single-line comments (// ...).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowLineComments(true))
  parser := jsonvx.NewParser([]byte(`// Kazuma's luck stat
  99`), cfg) // valid number after line comment
  ```
  ### `AllowBlockComments`:
  Enables the use of block comments (`/* ... */`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowBlockComments(true))
  parser := jsonvx.NewParser([]byte(`/* Explosions Megumin casts per day */ 1`), cfg) // valid number after block comment
  ```
  ### `DisallowDuplicateKeys`:
  Rejects objects that repeat a key with `ErrJSONDuplicateKey`, reporting where the duplicate is. Keys are compared after decoding escapes, so `"a"` and `"\u0061"` count as the same key. When disabled (the default), the last value for a repeated key wins. Useful for linters and validators, where a repeated key is usually a mistake.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithDisallowDuplicateKeys(true))
  parser := jsonvx.NewParser([]byte(`{"best_girl": "Rem", "best_boy": "Subaru"}`), cfg)   // valid, keys are unique
  parser := jsonvx.NewParser([]byte(`{"best_girl": "Rem", "best_girl": "Emilia"}`), cfg) // error: duplicate key in JSON object: "best_girl" at line 1, column 22
  ```
  ### `AllowJSON5`:
  Enables the use of [`JSON5`](https://json5.org) syntax.
  ```go
  parser := jsonvx.NewParser([]byte(`
{
  // Rudeus Greyrat's status window
  name: 'Rudeus Greyrat',
  title: 'Rudeus "the Quagmire"',
  lore: "Reborn in another world, \
determined to live without regrets. No \\n needed!",
  mana: 0xFF,
  critChance: .25, luckMultiplier: 7.,
  karma: +1,
  spells: ['Stone Cannon', 'Quagmire',],
  "teacher": "Roxy Migurdia",
}
`), jsonvx.JSON5Config())
  ```
  ### `MaxDepth`:
  Sets the maximum nesting depth of arrays and objects (default `1000`, see `jsonvx.DefaultMaxDepth`). Deeper input fails with `ErrJSONTooDeep` instead of overflowing the stack, which would crash the whole program. This applies to every config, including `JSON5Config()`. A value of `0` or less uses the default.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithMaxDepth(2))
  parser := jsonvx.NewParser([]byte(`[["loop 1"]]`), cfg)    // valid, depth 2
  parser := jsonvx.NewParser([]byte(`[[["loop 99"]]]`), cfg) // error: JSON nesting too deep: "[" exceeds maximum depth of 2 at line 1, column 3 (Return by Death can't save you here)
  ```

## Parsing behavior
Below are examples of how to `parse`, `traverse` and `retrieve` values from the parsed `JSON` input using the `Parser`.

### Object
- `Objects` are stored as a `slice` of `jsonvx.KeyValue` struct.
- Keys keep the order they were written in the input. `All()`, `Values()`, `Keys()` and `String()` all follow that order, which matters for formatters and linters. Use `sort.Strings(obj.Keys())` if you want alphabetical order.
- keys are stored as `[]byte`. Each object also keeps a lookup table from key to position, so `QueryPath` jumps straight to a key instead of searching for it.
- values are stored as `jsonvx.JSON`
- Loop over key-value pairs with `for key, value := range obj.All()`, or over just the values with `obj.Values()`. `Keys()` returns the keys as `[]string`.
- `Equal` compares objects by key, so key order does not affect it.
- Mixed value types (e.g., `numbers`, `strings`, `objects`) are obviously supported.
- Duplicate keys: the last value wins, matching `encoding/json`, JavaScript's `JSON.parse` and the `JSON5` spec. `{"a": 1, "a": 2}` is stored as `{"a": 2}`, so `Len()`, `All()` and `QueryPath` all see each key once. Keys are compared after decoding escapes, so `"a"` and `"\u0061"` are the same key. To reject duplicates instead, use [`DisallowDuplicateKeys`](#disallowduplicatekeys).

> **Changed in v1.1.0:** before v1.1.0, duplicate keys were all kept and `QueryPath` was documented to return the first one. That was not reliable for larger objects, because the key sort did not keep equal keys in order. A repeated key now keeps the position where it first appeared, with the last value, as in JavaScript.
>
> **Changed in v1.1.0:** before v1.1.0, keys were sorted alphabetically, so `ForEach` and `String()` did not follow the input. They now keep the input order.
>
> **New in v1.3.0:** `All()` and `Values()` replace `ForEach`, which still works but is deprecated. They are Go 1.23 iterators, so you loop with plain `for ... range`, can `break` early, and get keys as `string` instead of `[]byte`.

This example shows how to parse a `JSON` object using `jsonvx`, cast it to a `jsonvx.Object` node, and loop over its key-value pairs with `All()`.

```go
parser := jsonvx.NewParser([]byte(`{"name": "Megumin", "spell": "Explosion"}`), jsonvx.NewParserConfig())

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

// Loop over key-value pairs, in input order
for key, value := range objNode.All() {
	fmt.Printf("Key: %s, Value: %v\n", key, value)
}
// Key: name, Value: Megumin
// Key: spell, Value: Explosion

// Just the values
for value := range objNode.Values() {
	fmt.Println(value)
}
// Megumin
// Explosion

fmt.Println(objNode.Keys()) // [name spell]
```

### Array

- Items are stored as a `slice` of `jsonvx.JSON`
- Loop over items with `for i, item := range arr.All()`, or over just the items with `arr.Values()`. `ForEach` still works but is deprecated.

This example demonstrates how to parse a `JSON` `array` using `jsonvx`, cast it to a `jsonvx.Array` node, and loop over its items with `All()`.

```go
parser := jsonvx.NewParser([]byte(`["Aqua", "Megumin", "Darkness"]`), jsonvx.NewParserConfig())

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

// Loop over each item with its index
for i, item := range arrNode.All() {
	fmt.Printf("Item %d: %v\n", i, item)
}
// Item 0: Aqua
// Item 1: Megumin
// Item 2: Darkness

// Stop early with break
for item := range arrNode.Values() {
	if item.String() == "Megumin" {
		fmt.Println("Found Megumin. EXPLOSION!")
		break
	}
}
// Found Megumin. EXPLOSION!
```

### String

- JSON string values are parsed as `jsonvx.String`.
- `Value()` returns the decoded Go `string` value: escape sequences such as `\n`, `\"`, `\u00e9` and emoji pairs like `\ud83d\ude00` become the characters they stand for. With `JSON5` options enabled, `\'`, `\v`, `\0`, `\xHH` and line continuations (a backslash at the end of a line) are decoded too.
- `Raw()` returns the string exactly as written in the input, without the surrounding quotes and without decoding escapes.
- Object keys are decoded the same way, so `{"caf\u00e9": 1}` is queried with `QueryPath("café")`.

> **Changed in v1.1.0:** before v1.1.0, `Value()` returned the string with escape sequences left undecoded (`"a\nb"` gave a backslash and an `n`). If you relied on that, use `Raw()`.

This example demonstrates how to parse a `JSON` `string` using `jsonvx`, cast it to a `jsonvx.String` node, and get its `string` value using the built-in `Value` method.

```go
parser := jsonvx.NewParser([]byte(`"Starting life in another world!"`), jsonvx.NewParserConfig())

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

fmt.Println(strValue) // Output: Starting life in another world!
```

Escape sequences are decoded by `Value()`, while `Raw()` keeps them as written:

```go
parser := jsonvx.NewParser([]byte(`"caf\u00e9\nsay \"hi\""`), nil)
node, _ := parser.Parse()
strNode, _ := jsonvx.AsString(node)

value, _ := strNode.Value() // café
                            // say "hi"
raw, _ := strNode.Raw()     // caf\u00e9\nsay \"hi\"
```

### Number

- JSON numbers are parsed as a `jsonvx.Number`.
- `Value()` returns the number as a Go `float64`. A `float64` holds about 16 significant digits, so longer numbers are rounded: `9007199254740993` becomes `9007199254740992`.
- When every digit matters, such as for IDs, use the exact methods instead:
  - `Int64()` returns an exact `int64`. It returns `ErrNotInteger` for numbers written with a decimal point or exponent (`1.5`, `1.0`, `1e3`), and `ErrNumberOutOfRange` for whole numbers that don't fit in an `int64`.
  - `BigInt()` returns an exact `*big.Int` for whole numbers of any size.
  - `Raw()` returns the number exactly as written, such as `"1.50"` or `"0x1F"`.
- `Integer`, `Float`, `SciNot`, `Hex`, `Infinity` and ironically `NaN` number types are supported. Whole numbers of any length are `Integer`.

This example demonstrates how to parse a `JSON` `number` using `jsonvx`, cast it to a `jsonvx.Number` node, and get its `float64` value using the built-in `Value` method.

```go
parser := jsonvx.NewParser([]byte("9999"), jsonvx.NewParserConfig())

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

fmt.Println(numValue) // Output: 9999
```

For large IDs, `Value()` rounds while `Int64()`, `BigInt()` and `Raw()` keep every digit:

```go
parser := jsonvx.NewParser([]byte(`{"adventurer_card_id": 9007199254740993}`), nil)
node, _ := parser.Parse()
obj, _ := jsonvx.AsObject(node)
idNode, _ := obj.QueryPath("adventurer_card_id")
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

This example demonstrates how to parse a `JSON` `boolean` using `jsonvx`, cast it to a `jsonvx.Boolean` node, and get its `bool` value using the built-in `Value` method.

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

This example demonstrates how to parse a `JSON` `null` using `jsonvx`, cast it to a `jsonvx.Null` node, and get its `nil` value using the built-in `Value` method.

```go
parser := jsonvx.NewParser([]byte("null"), jsonvx.NewParserConfig())

// Parse the JSON input
node, err := parser.Parse()
if err != nil {
	panic(fmt.Sprintf("failed to parse JSON: %s", err))
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
cfg := jsonvx.NewParserConfig(jsonvx.WithAllowBlockComments(true), jsonvx.WithAllowLineComments(true))

// Comments around data are ignored
parser := jsonvx.NewParser([]byte(`/* Kazuma's luck stat */ 99 // suspiciously high`), cfg)
node, err := parser.Parse()
fmt.Println(node, err) // Output: 99 <nil>

// Only comments, no data: an error
parser = jsonvx.NewParser([]byte(`/* Aqua's intelligence stat */`), cfg)
node, err = parser.Parse()
fmt.Println(node, err) // Output: <nil> no meaningful content to parse
```

## Maintainers

- [bube054](https://github.com/bube054) - **Attah Gbubemi David (author)**

## Other helpful projects

- [ginvalidator](https://github.com/bube054/ginvalidator)
- [validatorgo](https://github.com/bube054/validatorgo)

## License

This project is licensed under the [MIT](https://opensource.org/license/mit). See the [LICENSE](https://github.com/bube054/jsonvx/blob/main/LICENSE.md) file for details.
