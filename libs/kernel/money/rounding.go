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
func (rc RoundingContext) Precision() uint32 { return rc.precision }

// Mode returns the rounding mode.
func (rc RoundingContext) Mode() RoundingMode { return rc.mode }

// String renders the context for a derivation record parameter.
func (rc RoundingContext) String() string {
	return fmt.Sprintf("precision=%d,mode=%s", rc.precision, rc.mode)
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
