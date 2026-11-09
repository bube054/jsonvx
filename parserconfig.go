package jsonvx

import (
	"fmt"
	"strings"
)

// ParserConfig defines the encoding and decoding behavior for the JSON parser.
//
// By default, all fields are false (strict mode). Enabling individual fields allows the parser
// to accept features that are not allowed by the standard [ECMA-404] specification but may appear
// in relaxed formats like [JSON5] or user-generated JSON.
//
// [ECMA-404]: https://ecma-international.org/publications-and-standards/standards/ecma-404/
// [JSON5]: https://json5.org/
type ParserConfig struct {
	AllowExtraWS bool // AllowExtraWS allows extra whitespace characters that are not normally permitted by strict JSON.

	AllowHexNumbers       bool // AllowHexNumbers enables support for hexadecimal numeric literals (e.g., 0xFF).
	AllowPointEdgeNumbers bool // AllowPointEdgeNumbers allows numbers like `.5` or `5.` without requiring a digit before/after the decimal point.

	AllowInfinity    bool // AllowInfinity enables the use of `Infinity` and `-Infinity` as number values.
	AllowNaN         bool // AllowNaN allows `NaN` (Not-a-Number) as a numeric value.
	AllowLeadingPlus bool // AllowLeadingPlus permits a leading '+' in numbers (e.g., `+42`).

	AllowUnquoted         bool // AllowUnquoted enables parsing of unquoted object keys (e.g., `{foo: "bar"}`).
	AllowSingleQuotes     bool // AllowSingleQuotes allows strings to be enclosed in single quotes (' ') in addition to double quotes.
	AllowNewlineInStrings bool // AllowNewlineInStrings permits multiple new line characters being escaped.
	AllowOtherEscapeChars bool // AllowOtherEscapeChars enables support for escape sequences other than \\, \/, \b, \n, \f, \r, \t and Unicode escapes (\uXXXX).

	// AllowControlCharsInStrings permits raw control characters (bytes 0x00-0x1F, such as a
	// tab) inside strings, as JSON5 does. Raw line breaks (LF and CR) are never allowed in
	// strings, as in both JSON and JSON5; use AllowNewlineInStrings for an escaped line break.
	AllowControlCharsInStrings bool

	AllowTrailingCommaArray  bool // AllowTrailingCommaArray permits a trailing comma in array literals (e.g., `[1, 2, ]`).
	AllowTrailingCommaObject bool // AllowTrailingCommaObject permits a trailing comma in object literals (e.g., `{"a": 1,}`).

	// DisallowDuplicateKeys rejects objects that repeat a key, such as {"a": 1, "a": 2},
	// with ErrJSONDuplicateKey. Keys are compared after decoding escapes, so "a" and
	// "\u0061" count as the same key.
	//
	// When false (the default), the last value for a repeated key wins, matching
	// encoding/json, JavaScript's JSON.parse and the JSON5 spec.
	DisallowDuplicateKeys bool

	AllowLineComments  bool // AllowLineComments enables the use of single-line comments (// ...).
	AllowBlockComments bool // AllowBlockComments enables the use of block comments (/* ... */).

	// MaxDepth is the maximum nesting depth of arrays and objects the parser accepts.
	// Input nested deeper than this is rejected with ErrJSONTooDeep instead of being parsed.
	// This protects against stack overflows, which crash the whole program, on hostile input.
	// A value of 0 or less means DefaultMaxDepth.
	MaxDepth int
}

// DefaultMaxDepth is the nesting depth limit used when ParserConfig.MaxDepth is 0 or less.
const DefaultMaxDepth = 1000

// maxDepth returns the effective nesting depth limit.
func (c *ParserConfig) maxDepth() int {
	if c.MaxDepth <= 0 {
		return DefaultMaxDepth
	}

	return c.MaxDepth
}

// NewParserConfig creates a new ParserConfig instance, optionally applying one or more configuration options.
// Options are applied in the order provided.
func NewParserConfig(opts ...func(*ParserConfig)) *ParserConfig {
	cfg := &ParserConfig{}

	for _, o := range opts {
		o(cfg)
	}

	return cfg
}

// String returns a formatted string representing all configuration options in the ParserConfig.
// Each field is listed with its corresponding boolean value.
func (c *ParserConfig) String() string {
	var b strings.Builder
	b.WriteString("ParserConfig{\n")

	configFields := []struct {
		name  string
		value bool
	}{
		{"AllowExtraWS", c.AllowExtraWS},
		{"AllowHexNumbers", c.AllowHexNumbers},
		{"AllowPointEdgeNumbers", c.AllowPointEdgeNumbers},
		{"AllowInfinity", c.AllowInfinity},
		{"AllowNaN", c.AllowNaN},
		{"AllowLeadingPlus", c.AllowLeadingPlus},
		{"AllowUnquoted", c.AllowUnquoted},
		{"AllowSingleQuotes", c.AllowSingleQuotes},
		{"AllowNewlineInStrings", c.AllowNewlineInStrings},
		{"AllowOtherEscapeChars", c.AllowOtherEscapeChars},
		{"AllowControlCharsInStrings", c.AllowControlCharsInStrings},
		{"AllowTrailingCommaArray", c.AllowTrailingCommaArray},
		{"AllowTrailingCommaObject", c.AllowTrailingCommaObject},
		{"DisallowDuplicateKeys", c.DisallowDuplicateKeys},
		{"AllowLineComments", c.AllowLineComments},
		{"AllowBlockComments", c.AllowBlockComments},
	}

	for _, f := range configFields {
		b.WriteString(fmt.Sprintf("  %s: %v,\n", f.name, f.value))
	}

	b.WriteString(fmt.Sprintf("  MaxDepth: %d,\n", c.maxDepth()))

	b.WriteString("}")
	return b.String()
}

// WithAllowExtraWS is the functional option setters for the AllowExtraWS flag.
func WithAllowExtraWS(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowExtraWS = allow
	}
}

// Functional option setters for the AllowHexNumbers flag.
func WithAllowHexNumbers(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowHexNumbers = allow
	}
}

// Functional option setters for the AllowHexNumbers flag.
func WithAllowPointEdgeNumbers(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowPointEdgeNumbers = allow
	}
}

func WithAllowInfinity(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowInfinity = allow
	}
}

func WithAllowNaN(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowNaN = allow
	}
}

func WithAllowLeadingPlus(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowLeadingPlus = allow
	}
}

func WithAllowUnquoted(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowUnquoted = allow
	}
}

func WithAllowSingleQuotes(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowSingleQuotes = allow
	}
}

func WithAllowNewlineInStrings(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowNewlineInStrings = allow
	}
}

func WithAllowOtherEscapeChars(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowOtherEscapeChars = allow
	}
}

// WithAllowControlCharsInStrings sets whether raw control characters other than line
// breaks (such as a tab) are allowed inside strings.
func WithAllowControlCharsInStrings(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowControlCharsInStrings = allow
	}
}

func WithAllowTrailingCommaArray(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowTrailingCommaArray = allow
	}
}

func WithAllowTrailingCommaObject(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowTrailingCommaObject = allow
	}
}

// WithDisallowDuplicateKeys sets whether objects that repeat a key are rejected
// with ErrJSONDuplicateKey instead of letting the last value win.
func WithDisallowDuplicateKeys(disallow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.DisallowDuplicateKeys = disallow
	}
}

func WithAllowLineComments(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowLineComments = allow
	}
}

func WithAllowBlockComments(allow bool) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.AllowBlockComments = allow
	}
}

// WithMaxDepth sets the maximum nesting depth of arrays and objects.
// A value of 0 or less means DefaultMaxDepth.
func WithMaxDepth(depth int) func(*ParserConfig) {
	return func(c *ParserConfig) {
		c.MaxDepth = depth
	}
}

// JSON5Config returns a ParserConfig with all features enabled for JSON5 compatibility.
func JSON5Config() *ParserConfig {
	return &ParserConfig{
		AllowExtraWS:               true,
		AllowHexNumbers:            true,
		AllowPointEdgeNumbers:      true,
		AllowInfinity:              true,
		AllowNaN:                   true,
		AllowLeadingPlus:           true,
		AllowUnquoted:              true,
		AllowSingleQuotes:          true,
		AllowNewlineInStrings:      true,
		AllowOtherEscapeChars:      true,
		AllowControlCharsInStrings: true,
		AllowTrailingCommaArray:    true,
		AllowTrailingCommaObject:   true,
		AllowLineComments:          true,
		AllowBlockComments:         true,
	}
}
