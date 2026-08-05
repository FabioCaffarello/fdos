// Package clock supplies knowledge time to the application layer.
//
// An adapter, because reading the clock is I/O in the sense that matters: its
// result is not a function of the program's inputs. The domain cannot call
// `time.Now()` — `nondet` refuses it — so the reading enters here and becomes
// an explicit, recordable input (ADR-0009).
package clock

import (
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// System reads the wall clock.
type System struct{}

// Now returns the current instant in UTC.
func (System) Now() temporal.Instant {
	return temporal.MustAt(time.Now().UTC())
}

// Fixed returns a constant instant.
//
// Not a test double bolted on afterwards: replaying a history requires
// controlling knowledge time, and a ledger that cannot be replayed cannot be
// audited. The system clock and this are the same port.
type Fixed struct{ At temporal.Instant }

// Now returns the fixed instant.
func (f Fixed) Now() temporal.Instant { return f.At }

// Sequence hands out instants in order, advancing by a fixed step.
//
// Models the one property the real clock has that matters to the ledger:
// knowledge time is monotonic per stream. A test that appends three facts gets
// three distinct, increasing knowledge times without coordinating timestamps by
// hand.
type Sequence struct {
	next time.Time
	step time.Duration
}

// NewSequence starts a sequence at `start`, advancing by `step` per reading.
func NewSequence(start temporal.Instant, step time.Duration) *Sequence {
	return &Sequence{next: start.Time(), step: step}
}

// Now returns the next instant in the sequence.
//
// Not safe for concurrent use, deliberately: a test that needs deterministic
// knowledge times is a test that must control the order of readings, and
// making this concurrency-safe would hide a race in the test rather than fix
// it.
func (s *Sequence) Now() temporal.Instant {
	at := temporal.MustAt(s.next)
	s.next = s.next.Add(s.step)
	return at
}
