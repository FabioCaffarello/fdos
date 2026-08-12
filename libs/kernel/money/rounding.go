package money

import (
	"fmt"

	"github.com/cockroachdb/apd/v3"
)

// RoundingMode selects how an inexact result is resolved.
//
// No mode is privileged and there is deliberately no package-level default. The
// correct choice is jurisdiction- and instrument-specific, and a library that
// picked one would encode a legal assumption where nobody would look for it.
type RoundingMode uint8

// Rounding modes. The zero value is invalid on purpose: a RoundingContext
// constructed by struct literal without thought will not silently round
// half-even, it will fail.
const (
	RoundingModeUnspecified RoundingMode = iota
	RoundingModeHalfEven                 // banker's rounding
	RoundingModeHalfUp
	RoundingModeDown // toward zero
	RoundingModeUp   // away from zero
	RoundingModeCeiling
	RoundingModeFloor
)

// String returns the mode name, matching the wire enum in
// fdos.kernel.v1.RoundingMode.
func (r RoundingMode) String() string {
	switch r {
	case RoundingModeHalfEven:
		return "half_even"
	case RoundingModeHalfUp:
		return "half_up"
	case RoundingModeDown:
		return "down"
	case RoundingModeUp:
		return "up"
	case RoundingModeCeiling:
		return "ceiling"
	case RoundingModeFloor:
		return "floor"
	default:
		return "unspecified"
	}
}

func (r RoundingMode) apdRounder() (apd.Rounder, error) {
	switch r {
	case RoundingModeHalfEven:
		return apd.RoundHalfEven, nil
	case RoundingModeHalfUp:
		return apd.RoundHalfUp, nil
	case RoundingModeDown:
		return apd.RoundDown, nil
	case RoundingModeUp:
		return apd.RoundUp, nil
	case RoundingModeCeiling:
		return apd.RoundCeiling, nil
	case RoundingModeFloor:
		return apd.RoundFloor, nil
	default:
		return "", fmt.Errorf("%w: rounding mode is unspecified", ErrInexact)
	}
}

// RoundingContext is precision plus mode, required by every division.
//
// It exists as a value so it can be recorded in a derivation record
// (ADR-0010): a rounding decision that does not appear in the trace is a
// rounding decision nobody can audit.
type RoundingContext struct {
	precision uint32
	mode      RoundingMode

	// scale is decimal places, and hasScale is its presence. Two fields rather
	// than a pointer because a RoundingContext is a value that gets copied into
	// derivation records, and a pointer would make two contexts that agree
	// compare unequal.
	//
	// Presence matters and is not pedantry: absent means "no scale constraint",
	// and `scale = 0` means round to whole units — which is JPY, not a
	// placeholder (ADR-0040).
	scale    int32
	hasScale bool
}

// NewRoundingContext builds a context. Both arguments are required, which is
// the whole point of the type.
func NewRoundingContext(precision uint32, mode RoundingMode) (RoundingContext, error) {
	if precision == 0 {
		return RoundingContext{}, fmt.Errorf("%w: precision must be positive", ErrInexact)
	}
	if precision > maxPrecision {
		return RoundingContext{}, fmt.Errorf("%w: precision %d exceeds %d", ErrInexact, precision, maxPrecision)
	}
	if _, err := mode.apdRounder(); err != nil {
		return RoundingContext{}, err
	}
	return RoundingContext{precision: precision, mode: mode}, nil
}

// MustRoundingContext is NewRoundingContext for constants and tests.
func MustRoundingContext(precision uint32, mode RoundingMode) RoundingContext {
	rc, err := NewRoundingContext(precision, mode)
	if err != nil {
		panic(err)
	}
	return rc
}

// Precision returns the significant digits the context allows.
//
// Significant digits, governing intermediates — not decimal places. The two are
// different concepts and this type now carries both, which is the answer every
// mature decimal system reached independently (ADR-0040). For a result rounded
// to a currency's minor units, see [RoundingContext.WithScale] and
// [Money.Quantize].
func (rc RoundingContext) Precision() uint32 { return rc.precision }

// WithScale returns a copy of the context that also fixes decimal places.
//
// A method rather than a second constructor: precision and mode stay required,
// and a scale is an addition to a context that was already valid without one.
// Negative rounds to tens or hundreds, which is as legitimate as rounding to
// cents.
func (rc RoundingContext) WithScale(scale int32) RoundingContext {
	rc.scale = scale
	rc.hasScale = true
	return rc
}

// Scale returns the decimal places the context fixes, and whether it fixes any.
//
// The bool is the whole point of the pair: a caller cannot mistake "no scale
// constraint" for "round to whole units", because zero is a real answer.
func (rc RoundingContext) Scale() (int32, bool) { return rc.scale, rc.hasScale }

// Mode returns the rounding mode.
func (rc RoundingContext) Mode() RoundingMode { return rc.mode }

// String renders the context for a derivation record parameter.
//
// The scale appears only when present, so a context without one renders exactly
// as it did before this field existed. That keeps a derivation address stable
// across the addition for every caller that does not use a scale — and this
// string is the pre-image, so a cosmetic change here moves real addresses.
func (rc RoundingContext) String() string {
	if !rc.hasScale {
		return fmt.Sprintf("precision=%d,mode=%s", rc.precision, rc.mode)
	}
	return fmt.Sprintf("precision=%d,mode=%s,scale=%d", rc.precision, rc.mode, rc.scale)
}

func (rc RoundingContext) apdContext() *apd.Context {
	rounder, err := rc.mode.apdRounder()
	if err != nil {
		// Unreachable: NewRoundingContext validates the mode, and the zero
		// value cannot escape it. Fall back to the strictest behaviour rather
		// than silently choosing a rounding mode for the caller.
		c := apd.BaseContext.WithPrecision(1)
		c.Traps = apd.InvalidOperation
		return c
	}
	c := apd.BaseContext.WithPrecision(rc.precision)
	c.Rounding = rounder
	c.Traps = apd.InvalidOperation | apd.DivisionByZero | apd.Overflow | apd.Underflow
	return c
}
