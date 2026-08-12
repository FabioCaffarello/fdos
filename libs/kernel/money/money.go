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
	if err := exactly(&out, func(c *apd.Context, o *apd.Decimal) (apd.Condition, error) {
		return c.Add(o, &m.amount, &other.amount)
	}); err != nil {
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
	if err := exactly(&out, func(c *apd.Context, o *apd.Decimal) (apd.Condition, error) {
		return c.Sub(o, &m.amount, &other.amount)
	}); err != nil {
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
	if err := exactly(&out, func(c *apd.Context, o *apd.Decimal) (apd.Condition, error) {
		return c.Mul(o, &m.amount, &q.amount)
	}); err != nil {
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

// ErrNoScale is returned by Quantize when the context fixes no decimal places.
//
// A Quantize without a scale has nothing to round to, and defaulting one would
// be the privileged default ADR-0008 forbids — here it would additionally be a
// guess about a currency.
var ErrNoScale = errors.New("money: rounding context fixes no scale")

// Quantize rounds the amount to the decimal places the context fixes.
//
// **This is the money operation.** Rounding to a currency's minor units is what
// ISO 4217 publishes and what Council Regulation (EC) No 1103/97 Article 5
// requires; a significant-digit budget cannot express it, which is why the
// context carries both concepts (ADR-0040). Reach for this rather than for
// [Money.Div]'s precision when the question is "how many cents".
//
// The returned bool reports whether the result was inexact, matching Div, so a
// caller that must record rounding in a derivation (ADR-0010) can tell.
//
// **Precision still binds, and that is a domain error rather than a NaN.** The
// decimal specification makes quantize total on scale and *partial* on
// precision: it raises Invalid Operation when the quantized coefficient would
// exceed the context's precision, so `35236450.6` to two places fails at
// precision 9. Callers must size precision from the largest amount they will
// hold times the currency's minor units. Letting that surface as a NaN a caller
// could propagate is the trap this signature exists to close — the same trap the
// exact context had before its conditions were trapped.
func (m Money) Quantize(rc RoundingContext) (result Money, inexact bool, err error) {
	scale, ok := rc.Scale()
	if !ok {
		return Money{}, false, ErrNoScale
	}

	// apd takes the target *exponent*, which is the negated scale: two decimal
	// places is exponent -2.
	var out apd.Decimal
	res, err := rc.apdContext().Quantize(&out, &m.amount, -scale)
	if err != nil {
		if res.InvalidOperation() {
			return Money{}, false, fmt.Errorf(
				"%w: %s cannot be held at scale %d within precision %d",
				ErrInexact, m, scale, rc.Precision())
		}
		return Money{}, false, err
	}
	return Money{amount: out, currency: m.currency}, res.Inexact(), nil
}

// exactly runs an exact-context operation and gives precision loss the sentinel
// ErrInexact already documents — "returned when an operation would silently lose
// precision without a RoundingContext to say how".
//
// Without this the trap surfaces an apd condition, which a caller can neither
// match with errors.Is nor act on without parsing a string. ADR-0008 requires
// the loss to be *signalled*, and a signal nothing can receive is not one.
// Failures that are not precision loss — overflow, invalid operation, division
// by zero — pass through untouched so they keep their own meaning.
func exactly(out *apd.Decimal, op func(*apd.Context, *apd.Decimal) (apd.Condition, error)) error {
	res, err := op(exactContext(), out)
	if err == nil {
		return nil
	}
	if res.Inexact() || res.Rounded() {
		return fmt.Errorf("%w: %w", ErrInexact, err)
	}
	return err
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
//
// Inexact and Rounded are trapped, and that is the whole point of the type.
// Without them this context inherited apd.BaseContext's rounding — half-up — and
// silently discarded a unit past maxPrecision: `10^96 + 1` compared equal to
// `10^96` with a nil error. That contradicted ADR-0008 twice, since it requires
// that "precision loss is signalled and recorded in the computation trace" and
// that no rounding mode be privileged.
//
// Rounded is trapped alongside Inexact even though no measured case fires it
// alone. It can in principle fire when digits are discarded without changing the
// value, which in this system is still a loss: trailing zeros are significant
// because they record the precision of the value (fdos.kernel.v1.Decimal).
//
// No rounding mode is set, deliberately. With both conditions trapped the mode
// is unreachable, and choosing one here would be the privileged default ADR-0008
// forbids.
func exactContext() *apd.Context {
	c := apd.BaseContext.WithPrecision(maxPrecision)
	c.Traps = apd.InvalidOperation | apd.DivisionByZero | apd.Overflow |
		apd.Underflow | apd.Inexact | apd.Rounded
	return c
}

// maxPrecision bounds intermediate results. Generous enough for any realistic
// instrument, finite so that a pathological input cannot consume unbounded
// memory in a pure function.
const maxPrecision = 96
