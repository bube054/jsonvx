package jsonvx

import (
	"errors"
	"fmt"
	"math/big"
)

// Errors returned by the exact number accessors.
var (
	ErrNotInteger       = errors.New("value is not a JSON integer")
	ErrNumberOutOfRange = errors.New("number out of range")
)

// Int64 returns the number as an exact int64.
//
// Unlike Value, it never rounds: 9007199254740993 stays 9007199254740993. It
// accepts numbers written as whole numbers, in decimal (123, -5) or hex (0x1F,
// with AllowHexNumbers). It returns ErrNotInteger for numbers written with a
// decimal point or exponent (1.5, 1.0, 1e3), Infinity and NaN, and
// ErrNumberOutOfRange for whole numbers outside the int64 range; use BigInt for those.
func (n *Number) Int64() (int64, error) {
	value, err := n.BigInt()
	if err != nil {
		return 0, err
	}

	if !value.IsInt64() {
		return 0, fmt.Errorf("%w: %s does not fit in int64", ErrNumberOutOfRange, errorSnippet(n.Token.Literal))
	}

	return value.Int64(), nil
}

// BigInt returns the exact value of a whole number of any size, in decimal or hex.
// It returns ErrNotInteger for numbers written with a decimal point or exponent,
// Infinity and NaN.
func (n *Number) BigInt() (*big.Int, error) {
	if n == nil || n.Token == nil {
		return nil, ErrNotNumber
	}

	literal := n.Token.Literal

	switch n.Token.SubKind {
	case INTEGER:
		value, ok := new(big.Int).SetString(string(literal), 10)
		if !ok {
			return nil, ErrNotNumber
		}
		return value, nil
	case HEX:
		value, ok := hexToBigInt(literal)
		if !ok {
			return nil, ErrNotNumber
		}
		return value, nil
	case FLOAT, SCI_NOT, INF, NaN:
		return nil, fmt.Errorf("%w: %s", ErrNotInteger, literal)
	default:
		return nil, ErrNotNumber
	}
}

// Raw returns the number exactly as written in the input, such as "1.50",
// "9007199254740993" or "0x1F". Use it to keep every digit, to pass the number to
// another parser, or to store it as text.
func (n *Number) Raw() (string, error) {
	if n == nil || n.Token == nil || n.Token.Kind != NUMBER {
		return "", ErrNotNumber
	}

	return string(n.Token.Literal), nil
}
