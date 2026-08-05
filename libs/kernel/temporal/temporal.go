// Package temporal is the FDOS bitemporal vocabulary (RFC-0003, ADR-0009).
//
// Two axes, and the distinction between them is what prevents look-ahead bias:
//
//   - **Effective time** — when a fact was true in the world. Supplied by the
//     source or by the domain.
//   - **Knowledge time** — when FDOS could first have acted on it. Assigned at
//     append, monotonic per stream, and never supplied by a caller.
//
// A caller cannot construct Coordinates with a knowledge time of its choosing:
// the constructor takes it from the append operation. That is what makes
// backdating unrepresentable rather than merely discouraged.
//
// There is no `time.Now()` here or anywhere in the kernel. The clock is
// injected at the application boundary, where the reading becomes recorded
// provenance. The `nondet` analyser enforces it.
package temporal

import (
	"errors"
	"fmt"
	"time"
)

// Errors returned by this package.
var (
	// ErrInvalidInterval is returned when an effective interval ends before it
	// begins.
	ErrInvalidInterval = errors.New("temporal: interval ends before it begins")

	// ErrZeroInstant is returned when a required instant is the zero time.
	ErrZeroInstant = errors.New("temporal: instant is unset")
)

// Instant is a point in time, UTC, at nanosecond resolution.
//
// A distinct type over time.Time so that an effective instant cannot be passed
// where a knowledge instant is expected without going through a constructor
// that says which is which.
type Instant struct {
	t time.Time
}

// At builds an Instant, normalising to UTC. Local zones are a presentation
// concern; a ledger stores one timeline.
func At(t time.Time) (Instant, error) {
	if t.IsZero() {
		return Instant{}, ErrZeroInstant
	}
	return Instant{t: t.UTC()}, nil
}

// MustAt is At for constants and tests.
func MustAt(t time.Time) Instant {
	i, err := At(t)
	if err != nil {
		panic(err)
	}
	return i
}

// Time returns the underlying UTC time.
func (i Instant) Time() time.Time { return i.t }

// IsZero reports whether the instant is unset.
func (i Instant) IsZero() bool { return i.t.IsZero() }

// Before reports whether i is strictly before other.
func (i Instant) Before(other Instant) bool { return i.t.Before(other.t) }

// After reports whether i is strictly after other.
func (i Instant) After(other Instant) bool { return i.t.After(other.t) }

// Equal reports whether two instants denote the same moment.
func (i Instant) Equal(other Instant) bool { return i.t.Equal(other.t) }

// String renders RFC 3339 with nanoseconds.
func (i Instant) String() string { return i.t.Format(time.RFC3339Nano) }

// Interval is when a fact was true: half-open [from, to).
//
// Half-open so adjacent intervals compose with neither gap nor overlap. An
// instantaneous fact is the degenerate case where to equals from; a fact still
// true has an open end. One representation for both, so no consumer has to
// handle two shapes.
type Interval struct {
	from Instant
	to   Instant // zero means open
}

// NewInterval builds a closed interval [from, to).
func NewInterval(from, to Instant) (Interval, error) {
	if from.IsZero() {
		return Interval{}, fmt.Errorf("%w: from is unset", ErrZeroInstant)
	}
	if !to.IsZero() && to.Before(from) {
		return Interval{}, fmt.Errorf("%w: [%s, %s)", ErrInvalidInterval, from, to)
	}
	return Interval{from: from, to: to}, nil
}

// OpenFrom builds an interval that has not ended: [from, ∞).
func OpenFrom(from Instant) (Interval, error) {
	return NewInterval(from, Instant{})
}

// IntervalAt builds the degenerate interval for an instantaneous fact:
// [at, at]. Contains reports true for exactly that instant.
func IntervalAt(at Instant) (Interval, error) {
	return NewInterval(at, at)
}

// From returns the start of the interval.
func (iv Interval) From() Instant { return iv.from }

// To returns the end, or the zero Instant when the interval is open.
func (iv Interval) To() Instant { return iv.to }

// IsOpen reports whether the interval has no end.
func (iv Interval) IsOpen() bool { return iv.to.IsZero() }

// Contains reports whether an instant falls inside [from, to).
//
// The degenerate interval [at, at] contains exactly `at`, which would otherwise
// be empty under a strictly half-open reading.
func (iv Interval) Contains(i Instant) bool {
	if i.Before(iv.from) {
		return false
	}
	if iv.IsOpen() {
		return true
	}
	if iv.to.Equal(iv.from) {
		return i.Equal(iv.from)
	}
	return i.Before(iv.to)
}

// String renders the interval.
func (iv Interval) String() string {
	if iv.IsOpen() {
		return fmt.Sprintf("[%s, ∞)", iv.from)
	}
	return fmt.Sprintf("[%s, %s)", iv.from, iv.to)
}

// Coordinates place a fact on both axes.
//
// The constructor is deliberately awkward for a caller that wants to invent a
// knowledge time: it is not a parameter a domain caller supplies, it comes from
// the append boundary. See Assign.
type Coordinates struct {
	effective Interval
	knowledge Instant
}

// Assign places a fact at an effective interval and a knowledge instant.
//
// Called by the ledger at append with the injected clock's reading — never by a
// domain rule. Allowing a caller to choose the knowledge time would reintroduce
// backdating, which is the contamination bitemporality exists to prevent.
func Assign(effective Interval, knowledge Instant) (Coordinates, error) {
	if effective.from.IsZero() {
		return Coordinates{}, fmt.Errorf("%w: effective interval is unset", ErrZeroInstant)
	}
	if knowledge.IsZero() {
		return Coordinates{}, fmt.Errorf("%w: knowledge time is unset", ErrZeroInstant)
	}
	return Coordinates{effective: effective, knowledge: knowledge}, nil
}

// Effective returns the interval over which the fact was true.
func (c Coordinates) Effective() Interval { return c.effective }

// Knowledge returns when FDOS could first have acted on the fact.
func (c Coordinates) Knowledge() Instant { return c.knowledge }

// AsOf is the pair a projection must be asked for. There is no default and no
// zero-value shortcut: a query that omits either does not compile, which is the
// mechanism that makes look-ahead bias structural (ADR-0009).
type AsOf struct {
	effective Instant
	knowledge Instant
}

// NewAsOf builds a query coordinate. Both arguments are required.
//
// The common failure this prevents is a knowledge time defaulting to "now": it
// is invisible, it is almost always what a developer wants interactively, and
// it is almost always wrong in an analysis.
func NewAsOf(effective, knowledge Instant) (AsOf, error) {
	if effective.IsZero() {
		return AsOf{}, fmt.Errorf("%w: as-of effective time", ErrZeroInstant)
	}
	if knowledge.IsZero() {
		return AsOf{}, fmt.Errorf("%w: as-of knowledge time", ErrZeroInstant)
	}
	return AsOf{effective: effective, knowledge: knowledge}, nil
}

// Effective returns the effective coordinate of the query.
func (a AsOf) Effective() Instant { return a.effective }

// Knowledge returns the knowledge coordinate of the query.
func (a AsOf) Knowledge() Instant { return a.knowledge }

// String renders the query coordinate.
func (a AsOf) String() string {
	return fmt.Sprintf("effective=%s knowledge=%s", a.effective, a.knowledge)
}

// Visible reports whether a fact at these coordinates is visible to a query.
//
// Both conditions must hold: the fact was already known, and it was in force.
// The knowledge test is what excludes information that arrived after the
// moment being analysed.
func (c Coordinates) Visible(a AsOf) bool {
	if c.knowledge.After(a.knowledge) {
		return false
	}
	return c.effective.Contains(a.effective)
}
