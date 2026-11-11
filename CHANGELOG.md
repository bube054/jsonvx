# Changelog

All notable changes to `jsonvx`. Versions follow [semantic versioning](https://semver.org).

## v1.4.2 (2026-11-11)

### Fixed
- Error messages no longer repeat the rest of the input. When parsing stops at an invalid character, the message used to quote everything from there to the end of the input, which on a config file could be hundreds of lines. It now quotes the rest of that line, at most 30 characters, ending in `…` when shortened: `unexpected character in JSON input: "// cache settings…" at line 2, column 3`. The line and column, and the error values for `errors.Is`, are unchanged.
- The same applies to `Number.Int64` errors for very long numbers.

### Other
- CI runs `gofmt`, `go vet` and race-detector tests, including the JSONTestSuite cases, on every push and pull request, on Go 1.23 and the two current releases.

## v1.4.1 (2026-11-10)

Documentation only; no code changes.

### Changed
- The README has Objects and Arrays examples for `All()`, `Values()`, `Keys()` and `Len()`, plus Booleans and Null.
- The install command is `go get github.com/bube054/jsonvx`, and the README states the Go 1.23 minimum.

## v1.4.0 (2026-11-10)

### New
- `Parse`, `ParseObject` and `ParseArray` parse in one call. `ParseObject` and `ParseArray` also check the top-level type and return `ErrWrongType` if it is wrong.
- Typed getters on `*Object` and `*Array`: `GetString`, `GetInt`, `GetFloat`, `GetBool`, `GetNumber`, `GetObject` and `GetArray`. Errors include the path, for example `wrong JSON type: expected string, got number at ["count"]`, and wrap the existing error values so `errors.Is` works.
- `ErrWrongType`.

### Changed
- The README now shows only the getter style. `NewParser`, `Parser.Parse`, `QueryPath` and the `As*` functions are unchanged and still documented on pkg.go.dev.

## v1.3.0 (2026-11-10)

### New
- `All()` and `Values()` on `*Object` and `*Array`: Go 1.23 iterators for plain `for range` loops, with `break` support. Object keys come back as `string`.

### Deprecated
- `Object.ForEach` and `Array.ForEach`. They still work; use `All()` instead.

## v1.2.0 (2026-11-09)

`jsonvx` now passes all 318 [JSONTestSuite](https://github.com/nst/JSONTestSuite) cases in strict mode, and the cases run as part of `go test`.

### Fixed
- Numbers follow JSON's grammar: strict mode rejects `-01`, `2.e3`, `-.5`, `inf`, `infinity` and `nan`, and accepts `0e1` and `0e+1`.
- A NUL byte no longer ends the input and silently drops the rest.
- Raw control characters in strings are rejected in strict mode.
- With `AllowUnquoted`, bare words are only accepted as object keys, not as array items or the whole document.

### New
- `AllowControlCharsInStrings` (enabled by `JSON5Config`) allows raw control characters such as tabs in strings, but never raw line breaks.

### Behavior changes
- Strict mode now rejects raw control characters in strings, the invalid numbers above, and bare words as values. If you relied on raw control characters, enable `AllowControlCharsInStrings`.

## v1.1.0 (2026-11-09)

### Fixed
- Deeply nested input no longer crashes the process. `MaxDepth` (default 1000) returns `ErrJSONTooDeep`.
- Escape sequences in strings and keys are decoded.
- Valid strings such as `"a\\\"b"` are no longer rejected.
- Hex numbers are readable through `Value()`; before, it always returned an error.
- `String()` no longer panics on nil or empty nodes.

### New
- `DisallowDuplicateKeys` rejects objects that repeat a key.
- `Object.Keys()`.
- `Number.Int64()`, `BigInt()` and `Raw()` for exact values such as long IDs.
- `String.Raw()` returns a string exactly as written.

### Performance
- Parsing is about 2-3x faster and allocates 3-8x less memory.

### Behavior changes
- `String.Value()` returns decoded text. Use `Raw()` for the old output.
- Duplicate keys resolve to the last value instead of the first. A repeated key keeps the position where it first appeared.
- Object keys keep input order instead of being sorted alphabetically.
- Whole numbers too large for an `int64` are classified as `Integer` instead of `Float`, and whole numbers too long for a `float64` now parse instead of being rejected.

## v1.0.0 (2025-08-02)

First release.
