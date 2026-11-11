<p align="center">
  <img src="assets/logo-wordmark.svg" alt="jsonvx logo" width="420">
</p>

<p align="center">
  <b>JSON and JSON5 parsing for Go, strict by default.</b><br>
  Turn on comments, trailing commas, hex numbers and more, one at a time. Errors point to the line and column.
</p>

<p align="center">
  <a href="https://github.com/bube054/jsonvx/actions/workflows/ci.yml"><img src="https://github.com/bube054/jsonvx/actions/workflows/ci.yml/badge.svg" alt="CI"></a>
  <a href="jsontestsuite_test.go"><img src="https://img.shields.io/badge/JSONTestSuite-318%2F318%20passing-brightgreen" alt="JSONTestSuite: 318 of 318 cases passing"></a>
  <a href="https://pkg.go.dev/github.com/bube054/jsonvx"><img src="https://pkg.go.dev/badge/github.com/bube054/jsonvx.svg" alt="Go Reference"></a>
  <a href="https://github.com/bube054/jsonvx/releases"><img src="https://img.shields.io/github/v/release/bube054/jsonvx?label=version" alt="Latest version"></a>
  <a href="https://go.dev/"><img src="https://img.shields.io/github/go-mod/go-version/bube054/jsonvx" alt="Go version"></a>
  <a href="LICENSE.md"><img src="https://img.shields.io/github/license/bube054/jsonvx" alt="License"></a>
</p>

```go
config := []byte(`{
  // cache settings
  "ttl": 300,
  "hosts": ["a.example.com", "b.example.com"],
}`)

_, err := jsonvx.ParseObject(config, nil) // strict JSON by default
fmt.Println(err) // unexpected character in JSON input: "// cache settings…" at line 2, column 3

cfg := jsonvx.NewParserConfig(
	jsonvx.WithAllowLineComments(true),
	jsonvx.WithAllowTrailingCommaObject(true),
)
root, _ := jsonvx.ParseObject(config, cfg)
ttl, _ := root.GetInt("ttl") // 300
```

```sh
go get github.com/bube054/jsonvx
```

---

## Features

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

`jsonvx` is designed for flexibility and correctness, not raw performance, making it a good fit for tools, linters, and config loaders where input may vary or include non-standard extensions.

> **Formatting coming soon:** writing parsed data back out as valid `JSON` (compact or indented, including converting `JSON5` to standard `JSON`) is planned for an upcoming release.

## Why jsonvx?

How the same inputs fare in Go's standard library and two popular relaxed-JSON packages:

| | **jsonvx** | [encoding/json](https://pkg.go.dev/encoding/json) | [tailscale/hujson](https://github.com/tailscale/hujson) | [yosuke-furukawa/json5](https://github.com/yosuke-furukawa/json5) |
| --- | --- | --- | --- | --- |
| **Comments, trailing commas** | each one a switch | no | yes | yes |
| **Single quotes, hex, `NaN`, unquoted keys…** | each one a switch | no | no | no, rejects single quotes and hex |
| **Strict mode** | the default, passes [JSONTestSuite](https://github.com/nst/JSONTestSuite) | always strict | no: always accepts comments | no: always relaxed |
| **Error message gives** | line and column | no position | line and column | no position |
| **Decodes into structs** | no: a tree with typed getters | yes | yes, through encoding/json | yes |

**Choose encoding/json** for strict JSON into structs, especially in hot paths: it's faster. On a 0.5 MB document, jsonvx took about 2.6x as long as decoding into a struct; for typical config files the difference is negligible.

**Choose hujson** if all you need is comments and trailing commas, decoded into structs.

**Choose jsonvx** when you need more of JSON5 than that, want to decide exactly which extensions to accept, want a strict mode that's tested against JSONTestSuite, or want to read values by path without defining structs.

## Installing

To start using `jsonvx`, install [Go](https://golang.org) 1.23 or newer and run:

```sh
go get github.com/bube054/jsonvx
```

## Quick Start

Parse your `JSON` with `ParseObject`, then read each value in one call.

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

	root, err := jsonvx.ParseObject(data, nil)
	if err != nil {
		panic(err)
	}

	count, err := root.GetInt("count")
	if err != nil {
		panic(err)
	}
	fmt.Println(count) // 3

	title, _ := root.GetString("isekais", "0", "title")
	fmt.Println(title) // Re:Zero

	party, _ := root.GetArray("isekais", "2", "party")
	for i, member := range party.All() {
		fmt.Println(i, member) // 0 Aqua, 1 Megumin, 2 Darkness
	}
}
```

Use `ParseArray` when the top-level value is an array, or `Parse` when it could be anything. The second argument is the [parser config](#configuring-the-parser); `nil` means strict `JSON`.

## Reading Values

`*Object` and `*Array` have one getter per type. Each takes a [path](#paths) and returns the value already converted:

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
rank, _ := isekais.GetInt("2", "rank") // 3
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

A getter called on a nil `*Object` or `*Array`, for example after ignoring an error from `GetObject`, returns an error instead of panicking.

## Paths

A path is a list of strings, one per step: an object key, or an array index written as a string (`"0"`). Keys are used exactly as written, so a key with dots needs no escaping. Using the Quick Start data:

```go
root.GetString("isekais", "0", "title")       // "Re:Zero"
root.GetString("genres", "1")                  // "fantasy"
root.GetString("best.of.all")                  // "Re:Zero"
root.GetString("isekais", "1", "hero", "last") // "Greyrat"
root.GetString("isekais", "2", "party", "1")   // "Megumin"
root.GetInt("isekais", "2", "rank")            // 3
root.GetArray("genres")                        // [isekai, fantasy, comedy]
root.GetObject("isekais", "1")                 // {rank: 2, title: Mushoku Tensei, ...}
```

On an array, the first step is an index: `isekais.GetString("0", "title")`.

## Working With Each Type

### Objects
- `All()` loops over key-value pairs, `Values()` over just the values, and `Keys()` returns the keys as a `[]string`. All three follow the order the keys were written in. Use `sort.Strings(obj.Keys())` for alphabetical order.
- `Len()` is the number of keys.
- Duplicate keys: the last value wins, matching `encoding/json`, JavaScript's `JSON.parse` and the `JSON5` spec. To reject duplicates instead, use [`DisallowDuplicateKeys`](#disallowduplicatekeys).
- `Equal` compares objects by key, so key order does not affect it.

```go
hero, _ := root.GetObject("isekais", "0", "hero")

for key, value := range hero.All() {
	fmt.Println(key, value)
}
// first Subaru
// last Natsuki

for value := range hero.Values() {
	fmt.Println(value)
}
// Subaru
// Natsuki

fmt.Println(hero.Keys()) // [first last]
fmt.Println(hero.Len())  // 2

dup, _ := jsonvx.ParseObject([]byte(`{"best_girl": "Emilia", "best_girl": "Rem"}`), nil)
bestGirl, _ := dup.GetString("best_girl") // Rem, the last value wins
```

### Arrays
- In a path, an array index is written as a string: `"0"` is the first item.
- `All()` loops over index and item, `Values()` over just the items, and `Len()` is the number of items. `break`, `continue` and `return` work as in any `for range` loop.
- Loop items are `jsonvx.JSON` nodes. Printing one shows its value; to read inside one, cast it to `*jsonvx.Object` or `*jsonvx.Array`.

```go
party, _ := root.GetArray("isekais", "0", "party")
fmt.Println(party.Len()) // 4

firstMember, _ := party.GetString("0") // Emilia

for i, member := range party.All() {
	fmt.Println(i, member)
}
// 0 Emilia
// 1 Rem
// 2 Ram
// 3 Beatrice

for member := range party.Values() {
	if member.String() == "Rem" {
		fmt.Println("Found Rem")
		break
	}
}

// An array of objects: cast each item to read inside it.
isekais, _ := root.GetArray("isekais")
for item := range isekais.Values() {
	show, _ := item.(*jsonvx.Object)
	title, _ := show.GetString("title")
	fmt.Println(title)
}
// Re:Zero
// Mushoku Tensei
// KonoSuba
```

### Strings
- `GetString` decodes escape sequences: `\n`, `\"`, `\u00e9` and emoji pairs like `\ud83d\ude00` become the characters they stand for. With `JSON5` options, `\'`, `\v`, `\0`, `\xHH` and line continuations are decoded too.
- Keys are decoded the same way, so `{"caf\u00e9": 1}` is read with `root.GetInt("café")`.
- To get a string exactly as written, escapes and all, use `Raw()` on the [node](#working-with-nodes).

### Numbers
- `GetInt` is exact for every whole number that fits in an `int64`.
- `GetFloat` returns a `float64`, which holds about 16 significant digits, so longer numbers are rounded: `9007199254740993` becomes `9007199254740992`.
- For whole numbers of any size, or the number exactly as written, use `GetNumber`:

```go
root, _ := jsonvx.ParseObject([]byte(`{"adventurer_card_id": 9007199254740993}`), nil)

approx, _ := root.GetFloat("adventurer_card_id") // 9.007199254740992e+15 (rounded)
exact, _ := root.GetInt("adventurer_card_id")    // 9007199254740993

id, _ := root.GetNumber("adventurer_card_id")
big, _ := id.BigInt() // 9007199254740993, any size
raw, _ := id.Raw()    // "9007199254740993", as written
```

### Booleans and Null
- `GetBool` reads `true` and `false`.
- `null` has no value to return, so there is no getter for it. Check for it with a type assertion on the node:

```go
root, _ := jsonvx.ParseObject([]byte(`{"finished": false, "season4": null}`), nil)

finished, _ := root.GetBool("finished") // false

node, _ := root.QueryPath("season4")
_, isNull := node.(*jsonvx.Null) // true
```

### Comments
- Comments are ignored.
- Input with only comments and no data is an error: `no meaningful content to parse`.

## Configuring The Parser

By default, the parser is strict, matching the [ECMA-404](https://ecma-international.org/publications-and-standards/standards/ecma-404/) specification. To allow relaxed formats, build a config with the options you want and pass it to `Parse`, `ParseObject` or `ParseArray`:

```go
cfg := jsonvx.NewParserConfig(
	jsonvx.WithAllowTrailingCommaArray(true),
	jsonvx.WithAllowLineComments(true),
)
root, err := jsonvx.ParseObject(data, cfg)
```

  ### `AllowExtraWS`:
  Allows extra whitespace characters that are not normally permitted by strict `JSON`, such as vertical tab and form feed.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowExtraWS(true))
  jsonvx.Parse([]byte("{\"hero\":\v\"Subaru\"}"), cfg) // ok: vertical tab
  jsonvx.Parse([]byte("{\"hero\":\f\"Subaru\"}"), cfg) // ok: form feed
  ```
  ### `AllowHexNumbers`:
  Enables hexadecimal numbers such as `0x1F`. Upper and lower case work (`0xff`, `0XFF`), as do negative numbers (`-0x10`) and, with [`AllowLeadingPlus`](#allowleadingplus), a leading plus.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowHexNumbers(true))
  root, _ := jsonvx.ParseObject([]byte(`{"mana": 0xFF}`), cfg)
  mana, _ := root.GetInt("mana") // 255
  ```
  ### `AllowPointEdgeNumbers`:
  Allows numbers like `.5` or `5.` without a digit before or after the decimal point.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowPointEdgeNumbers(true))
  jsonvx.Parse([]byte(".5"), cfg) // ok
  jsonvx.Parse([]byte("5."), cfg) // ok
  ```
  ### `AllowInfinity`:
  Enables `Infinity`, `-Infinity` and `+Infinity` as numbers.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowInfinity(true))
  jsonvx.Parse([]byte("Infinity"), cfg)  // ok
  jsonvx.Parse([]byte("-Infinity"), cfg) // ok
  jsonvx.Parse([]byte("+Infinity"), cfg) // ok only with AllowLeadingPlus
  ```
  ### `AllowNaN`:
  Allows `NaN` (Not-a-Number) as a number.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowNaN(true))
  jsonvx.Parse([]byte("NaN"), cfg)  // ok
  jsonvx.Parse([]byte("-NaN"), cfg) // ok
  jsonvx.Parse([]byte("+NaN"), cfg) // ok only with AllowLeadingPlus
  ```
  ### `AllowLeadingPlus`:
  Permits a leading `+` in numbers.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowLeadingPlus(true))
  jsonvx.Parse([]byte("+9999"), cfg) // ok (EXPLOSION!)
  ```
  ### `AllowUnquoted`:
  Allows unquoted object keys. Values still need quotes.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowUnquoted(true))
  jsonvx.Parse([]byte(`{hero: "Kazuma"}`), cfg) // ok
  jsonvx.Parse([]byte(`{hero: Kazuma}`), cfg)   // error: unquoted value
  ```
  ### `AllowSingleQuotes`:
  Allows strings in single quotes as well as double quotes.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowSingleQuotes(true))
  jsonvx.Parse([]byte(`{'name': 'Megumin'}`), cfg) // ok
  ```
  ### `AllowNewlineInStrings`:
  Allows a line break inside a string when it is escaped with a backslash.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowNewlineInStrings(true))
  jsonvx.Parse([]byte(`"Starting life in \
another world"`), cfg) // ok
  ```
  ### `AllowOtherEscapeChars`:
  Allows escape sequences other than `\"`, `\\`, `\/`, `\b`, `\f`, `\n`, `\r`, `\t` and `\uXXXX`.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowOtherEscapeChars(true))
  jsonvx.Parse([]byte(`"another\qworld"`), cfg) // ok
  ```
  ### `AllowControlCharsInStrings`:
  Allows raw control characters (bytes `0x00`-`0x1F`, such as a tab) inside strings, as `JSON5` does. Strict `JSON` requires them to be escaped (`\t`). Raw line breaks are never allowed inside strings; use [`AllowNewlineInStrings`](#allownewlineinstrings) for an escaped one.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowControlCharsInStrings(true))
  jsonvx.Parse([]byte("\"Rem\tRam\""), cfg) // ok: raw tab
  jsonvx.Parse([]byte("\"Rem\nRam\""), cfg) // error: raw line break
  ```
  ### `AllowTrailingCommaArray`:
  Permits a trailing comma in arrays.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowTrailingCommaArray(true))
  jsonvx.Parse([]byte(`["Aqua", "Megumin", "Darkness", ]`), cfg) // ok
  ```
  ### `AllowTrailingCommaObject`:
  Permits a trailing comma in objects.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowTrailingCommaObject(true))
  jsonvx.Parse([]byte(`{"rank": 1,}`), cfg) // ok
  ```
  ### `AllowLineComments`:
  Allows single-line comments (`// ...`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowLineComments(true))
  jsonvx.Parse([]byte("// Kazuma's luck stat\n99"), cfg) // ok
  ```
  ### `AllowBlockComments`:
  Allows block comments (`/* ... */`).
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithAllowBlockComments(true))
  jsonvx.Parse([]byte(`/* Explosions Megumin casts per day */ 1`), cfg) // ok
  ```
  ### `DisallowDuplicateKeys`:
  Rejects objects that repeat a key with `ErrJSONDuplicateKey`, reporting where the duplicate is. Keys are compared after decoding escapes, so `"a"` and `"\u0061"` count as the same key. When disabled (the default), the last value for a repeated key wins. Useful for linters and validators, where a repeated key is usually a mistake.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithDisallowDuplicateKeys(true))
  jsonvx.Parse([]byte(`{"best_girl": "Rem", "best_boy": "Subaru"}`), cfg)   // ok
  jsonvx.Parse([]byte(`{"best_girl": "Rem", "best_girl": "Emilia"}`), cfg) // error: duplicate key in JSON object: "best_girl" at line 1, column 22
  ```
  ### `AllowJSON5`:
  `JSON5Config()` turns on every option above except `DisallowDuplicateKeys`, so you can read [`JSON5`](https://json5.org).
  ```go
  status, _ := jsonvx.ParseObject([]byte(`{
    // Rudeus Greyrat's status window
    name: 'Rudeus Greyrat',
    title: 'Rudeus "the Quagmire"',
    mana: 0xFF,
    critChance: .25, luckMultiplier: 7.,
    karma: +1,
    spells: ['Stone Cannon', 'Quagmire',],
  }`), jsonvx.JSON5Config())

  name, _ := status.GetString("name") // Rudeus Greyrat
  mana, _ := status.GetInt("mana")    // 255
  ```
  ### `MaxDepth`:
  Sets the maximum nesting depth of arrays and objects (default `1000`, see `jsonvx.DefaultMaxDepth`). Deeper input fails with `ErrJSONTooDeep` instead of overflowing the stack and crashing the program. This applies to every config, including `JSON5Config()`. A value of `0` or less uses the default.
  ```go
  cfg := jsonvx.NewParserConfig(jsonvx.WithMaxDepth(2))
  jsonvx.Parse([]byte(`[["loop 1"]]`), cfg)    // ok: depth 2
  jsonvx.Parse([]byte(`[[["loop 99"]]]`), cfg) // error: JSON nesting too deep (Return by Death can't save you here)
  ```

## Working With Nodes

When you don't know a value's type ahead of time, `QueryPath` returns the node itself, and a type switch tells you what it is. Every node type has a `Value()` method.

```go
node, err := root.QueryPath("isekais", "0", "rank")
if err != nil {
	panic(err)
}

switch v := node.(type) {
case *jsonvx.String:
	s, _ := v.Value() // string
	raw, _ := v.Raw() // as written, escapes kept
	fmt.Println(s, raw)
case *jsonvx.Number:
	n, _ := v.Value() // float64
	fmt.Println(n)
case *jsonvx.Boolean:
	b, _ := v.Value() // bool
	fmt.Println(b)
case *jsonvx.Null:
	fmt.Println("null")
case *jsonvx.Array:
	fmt.Println(v.Len(), "items")
case *jsonvx.Object:
	fmt.Println(v.Keys())
}
```

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for what changed in each release.

## Maintainers

- [bube054](https://github.com/bube054) - **Attah Gbubemi David (author)**

## Other helpful projects

- [ginvalidator](https://github.com/bube054/ginvalidator)
- [validatorgo](https://github.com/bube054/validatorgo)

## License

This project is licensed under the [MIT](https://opensource.org/license/mit). See the [LICENSE](https://github.com/bube054/jsonvx/blob/main/LICENSE.md) file for details.
