// Package money is the FDOS representation of monetary and quantity amounts
// (RFC-0002, ADR-0008).
//
// Three properties are structural here, not conventional:
//
//   - **No binary floating point.** The usual objection is representation
//     error. The decisive one is that floating-point addition is not
//     associative, so the same events folded in a different order give a
//     different total — which breaks Constitution §9 independently of
//     precision. The `nofloat` analyser enforces this at build time.
//   - **Currency is part of the value.** Adding USD to EUR returns an error
//     rather than a number, and there is no ambient "current currency".
//   - **Division requires an explicit RoundingContext.** There is no default.
//     Addition, subtraction and multiplication of decimals are exact; division
//     is not closed, and the decision about how to close it is where most
//     financial calculation bugs live.
package money

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cockroachdb/apd/v3"
)

// Errors returned by this package. They are values, not strings, so callers can
// distinguish "these cannot be added" from "this text is not a number".
var (
	// ErrCurrencyMismatch is returned by any operation combining two Money
	// values of different currencies.
	ErrCurrencyMismatch = errors.New("money: currency mismatch")

	// ErrInvalidCurrency is returned when a currency code is not three
	// uppercase letters.
	ErrInvalidCurrency = errors.New("money: invalid currency code")

	// ErrMalformed is returned when a decimal string is not in canonical form.
	ErrMalformed = errors.New("money: malformed decimal")

	// ErrDivideByZero is returned by Div when the divisor is zero.
	ErrDivideByZero = errors.New("money: division by zero")

	// ErrInexact is returned when an operation would silently lose precision
	// without a RoundingContext to say how.
	ErrInexact = errors.New("money: inexact result requires a rounding context")
)

// Currency is an ISO 4217 alphabetic code.
//
// A distinct type rather than a bare string so that a currency cannot be passed
// where a unit is expected, and so the zero value is obviously invalid.
type Currency string

// ParseCurrency validates and returns a Currency.
func ParseCurrency(code string) (Currency, error) {
	if len(code) != 3 {
		return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, code)
	}
	for _, r := range code {
		if r < 'A' || r > 'Z' {
			return "", fmt.Errorf("%w: %q", ErrInvalidCurrency, code)
		}
	}
	return Currency(code), nil
}

// String returns the ISO 4217 code.
func (c Currency) String() string { return string(c) }

// Money is an amount in a currency.
//
// The fields are unexported and there is no literal construction: a Money can
// only come from Parse or from an operation on existing Money values. That is
// what makes "currency is part of the value" a property of the type rather than
// a convention callers may forget.
type Money struct {
	amount   apd.Decimal
	currency Currency
}

// Parse builds a Money from a canonical decimal string and a currency code.
//
// Canonical form: an optional leading '-', digits, and an optional '.' followed
// by digits. No exponent, no leading '+', no grouping separators. Trailing
// zeros are significant — they record the precision of the value, which is why
// "1.50" and "1.5" are distinguishable here and compare equal under Cmp.
func Parse(amount string, currency string) (Money, error) {
	cur, err := ParseCurrency(currency)
	if err != nil {
		return Money{}, err
	}
	if invalid := validateCanonical(amount); invalid != nil {
		return Money{}, invalid
	}
	d, _, err := apd.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("%w: %q", ErrMalformed, amount)
	}
	return Money{amount: *d, currency: cur}, nil
}

// MustParse is Parse for constants and tests. It panics on invalid input and
// must never be used on data that came from outside the process.
func MustParse(amount, currency string) Money {
	m, err := Parse(amount, currency)
	if err != nil {
		panic(err)
	}
	return m
}

// Zero returns the zero amount in a currency. Distinct from the Money zero
// value, which has no currency and is not a valid amount.
func Zero(currency Currency) Money {
	return Money{currency: currency}
}

// validateCanonical rejects the forms apd would otherwise accept silently:
// exponents, leading '+', and whitespace. A ledger that accepts "1e3" for a
// thousand is a ledger nobody can audit by reading.
func validateCanonical(s string) error {
	if s == "" {
		return fmt.Errorf("%w: empty", ErrMalformed)
	}
	if strings.ContainsAny(s, "eE+ \t") {
		return fmt.Errorf("%w: %q uses exponent, sign or whitespace", ErrMalformed, s)
	}
	body := strings.TrimPrefix(s, "-")
	if body == "" {
		return fmt.Errorf("%w: %q", ErrMalformed, s)
	}
	if strings.Count(body, ".") > 1 {
		return fmt.Errorf("%w: %q", ErrMalformed, s)
	}
	for _, r := range body {
		if r != '.' && (r < '0' || r > '9') {
			return fmt.Errorf("%w: %q", ErrMalformed, s)
		}
	}
	if strings.HasSuffix(body, ".") || strings.HasPrefix(body, ".") {
		return fmt.Errorf("%w: %q", ErrMalformed, s)
	}
	return nil
}

// Currency returns the currency of the amount.
func (m Money) Currency() Currency { return m.currency }

// String returns the canonical decimal text. This is the wire form and the
// audit form: what is stored is what a human reads.
func (m Money) String() string { return m.amount.Text('f') }

// IsZero reports whether the amount is zero, regardless of precision.
func (m Money) IsZero() bool { return m.amount.IsZero() }

// Add returns m + other. Exact: no rounding context is needed because decimal
// addition never loses precision.
func (m Money) Add(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, err
	}
	var out apd.Decimal
	if _, err := exactContext().Add(&out, &m.amount, &other.amount); err != nil {
		return Money{}, err
	}
	return Money{amount: out, currency: m.currency}, nil
}

// Sub returns m - other. Exact.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return Money{}, err
	}
	var out apd.Decimal
	if _, err := exactContext().Sub(&out, &m.amount, &other.amount); err != nil {
		return Money{}, err
	}
	return Money{amount: out, currency: m.currency}, nil
}

// Neg returns -m.
func (m Money) Neg() Money {
	var out apd.Decimal
	out.Neg(&m.amount)
	return Money{amount: out, currency: m.currency}
}

// Cmp compares two amounts of the same currency: -1, 0 or +1.
//
// Comparison is by value, so "1.50" and "1.5" are equal. Precision is recorded
// in the text form, not in the ordering.
func (m Money) Cmp(other Money) (int, error) {
	if err := m.assertSameCurrency(other); err != nil {
		return 0, err
	}
	return m.amount.Cmp(&other.amount), nil
}

// Mul multiplies by a Quantity, returning Money.
//
// Money × Quantity is meaningful; Money × Money is not, and there is no method
// for it. Exact: decimal multiplication never loses precision, though the
// result carries the sum of the operands' scales.
func (m Money) Mul(q Quantity) (Money, error) {
	var out apd.Decimal
	if _, err := exactContext().Mul(&out, &m.amount, &q.amount); err != nil {
		return Money{}, err
	}
	return Money{amount: out, currency: m.currency}, nil
}

// Div divides by a Quantity under an explicit RoundingContext.
//
// The context is required, not defaulted. Division is not closed over the
// decimals — 1/3 has no terminating representation — and choosing how to close
// it is jurisdiction- and instrument-specific. A library that picked a default
// would encode a legal assumption, which is exactly how the decision goes
// unexamined.
//
// The returned bool reports whether the result was inexact, so a caller that
// must record rounding in a derivation (ADR-0010) can tell.
func (m Money) Div(q Quantity, rc RoundingContext) (result Money, inexact bool, err error) {
	if q.amount.IsZero() {
		return Money{}, false, ErrDivideByZero
	}
	ctx := rc.apdContext()
	var out apd.Decimal
	res, err := ctx.Quo(&out, &m.amount, &q.amount)
	if err != nil {
		return Money{}, false, err
	}
	return Money{amount: out, currency: m.currency}, res.Inexact(), nil
}

func (m Money) assertSameCurrency(other Money) error {
	if m.currency != other.currency {
		return fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, m.currency, other.currency)
	}
	return nil
}

// exactContext refuses to round. Any operation that would need to is an error
// rather than a silent approximation — the caller must reach for Div with a
// RoundingContext and say what it wants.
func exactContext() *apd.Context {
	c := apd.BaseContext.WithPrecision(maxPrecision)
	c.Traps = apd.InvalidOperation | apd.DivisionByZero | apd.Overflow | apd.Underflow
	return c
}

// maxPrecision bounds intermediate results. Generous enough for any realistic
// instrument, finite so that a pathological input cannot consume unbounded
// memory in a pure function.
const maxPrecision = 96
