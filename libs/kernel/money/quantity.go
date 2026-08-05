package money

import (
	"fmt"

	"github.com/cockroachdb/apd/v3"
)

// Unit is a canonical unit code: "share", "bond.face", "oz.t".
//
// A governed vocabulary but an open type. Closing it to an enum would make
// every new instrument class a change to this package, which private connectors
// cannot make (ADR-0018 records the same reasoning for the wire form).
type Unit string

// String returns the unit code.
func (u Unit) String() string { return string(u) }

// Quantity is an amount of something that is not money.
//
// Distinct from Money because the operations differ: Money × Quantity is
// meaningful, Money × Money is not, and there is deliberately no method for the
// second. Fractional quantities are permitted — fractional shares exist — so
// this is never an integer type.
type Quantity struct {
	amount apd.Decimal
	unit   Unit
}

// ParseQuantity builds a Quantity from canonical decimal text and a unit.
func ParseQuantity(amount string, unit string) (Quantity, error) {
	if unit == "" {
		return Quantity{}, fmt.Errorf("%w: empty unit", ErrMalformed)
	}
	if err := validateCanonical(amount); err != nil {
		return Quantity{}, err
	}
	d, _, err := apd.NewFromString(amount)
	if err != nil {
		return Quantity{}, fmt.Errorf("%w: %q", ErrMalformed, amount)
	}
	return Quantity{amount: *d, unit: Unit(unit)}, nil
}

// MustParseQuantity is ParseQuantity for constants and tests. It panics on
// invalid input and must never be used on data from outside the process.
func MustParseQuantity(amount, unit string) Quantity {
	q, err := ParseQuantity(amount, unit)
	if err != nil {
		panic(err)
	}
	return q
}

// Unit returns the unit of the quantity.
func (q Quantity) Unit() Unit { return q.unit }

// String returns the canonical decimal text.
func (q Quantity) String() string { return q.amount.Text('f') }

// IsZero reports whether the quantity is zero.
func (q Quantity) IsZero() bool { return q.amount.IsZero() }

// Add returns q + other, which requires the same unit. Exact.
func (q Quantity) Add(other Quantity) (Quantity, error) {
	if q.unit != other.unit {
		return Quantity{}, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, q.unit, other.unit)
	}
	var out apd.Decimal
	if _, err := exactContext().Add(&out, &q.amount, &other.amount); err != nil {
		return Quantity{}, err
	}
	return Quantity{amount: out, unit: q.unit}, nil
}

// Sub returns q - other, which requires the same unit. Exact.
func (q Quantity) Sub(other Quantity) (Quantity, error) {
	if q.unit != other.unit {
		return Quantity{}, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, q.unit, other.unit)
	}
	var out apd.Decimal
	if _, err := exactContext().Sub(&out, &q.amount, &other.amount); err != nil {
		return Quantity{}, err
	}
	return Quantity{amount: out, unit: q.unit}, nil
}

// Cmp compares two quantities of the same unit: -1, 0 or +1.
func (q Quantity) Cmp(other Quantity) (int, error) {
	if q.unit != other.unit {
		return 0, fmt.Errorf("%w: %s vs %s", ErrCurrencyMismatch, q.unit, other.unit)
	}
	return q.amount.Cmp(&other.amount), nil
}
