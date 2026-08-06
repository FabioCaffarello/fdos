// Package app orchestrates the ledger domain.
//
// This is the imperative shell around the pure core. It is where the things the
// domain refuses to touch live: the clock, `context.Context`, and the ports
// through which storage is reached (ADR-0013).
//
// Ports are declared **here**, not in `domain`. If they lived in the domain they
// would want `context.Context` in their signatures, and the ban on `context`
// there would collapse in the first week — which is exactly why RFC-0003 puts
// them in this layer.
package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// ErrStreamNotFound is returned by a Store when the named stream does not exist.
var ErrStreamNotFound = errors.New("app: stream not found")

// ErrStaleRead is returned when a stream changed between the read a caller
// based its decision on and the append acting on that decision.
//
// Deliberately distinct from an attempt to rewrite history. This one means
// *re-read and try again*; that one means a bug. A caller unable to tell them
// apart will retry the bug forever (ADR-0034).
var ErrStaleRead = errors.New("app: the stream changed since it was read")

// ErrNonMonotonicKnowledge is returned when an append carries a knowledge time
// no later than the stream's last.
//
// Knowledge time is monotonic per stream (ADR-0009). With one writer that holds
// by construction; with two it holds only if something checks, and this is the
// check. It is an honest failure rather than a defensive one: if two facts
// cannot be ordered on the axis that decides what was knowable when, FDOS
// cannot record both and pretend to know which came first.
var ErrNonMonotonicKnowledge = errors.New("app: knowledge time is not after the stream's last")

// Expectation is what a caller believes it read, carried into the append that
// acts on it (ADR-0034).
//
// Some appends depend on a prior read and some do not. Admitting a claim depends
// on nothing — it is judged on its own merits — so it expects nothing. Minting
// an identity resolved against a stream first, and a mint that landed since may
// already answer the claim, so it expects the length it resolved at.
//
// The zero value is [Any], which is the permissive answer. That is deliberate
// for a value type whose whole job is a precondition: a caller that forgets to
// set one gets today's behaviour rather than a spurious conflict, and the
// callers that need a precondition are the ones that had to think about it.
type Expectation struct {
	length  int
	checked bool
}

// Any appends regardless of what has happened to the stream since.
func Any() Expectation { return Expectation{} }

// AtLength appends only if the stream still holds exactly n facts.
func AtLength(n int) Expectation { return Expectation{length: n, checked: true} }

// Length reports the expected length and whether there is one to check.
func (e Expectation) Length() (int, bool) { return e.length, e.checked }

// String renders the expectation for an error message.
func (e Expectation) String() string {
	if !e.checked {
		return "any"
	}
	return fmt.Sprintf("at length %d", e.length)
}

// Clock supplies knowledge time.
//
// A port rather than a call to `time.Now()`, because a domain rule that reads
// the clock cannot be replayed (Constitution §2). Injecting it means the reading
// is an explicit input that becomes recorded provenance, and that a test can
// replay any history it likes.
type Clock interface {
	Now() temporal.Instant
}

// Store persists and retrieves fact streams.
//
// Append-only by construction: there is no Update, no Delete, and — since
// ADR-0034 — no whole-stream write either. A stream is only ever extended, so
// an operation that wrote one entire would have no legitimate caller, and the
// one it did have lost facts.
//
// # Why the store appends rather than accepting a stream
//
// `Ref` is `{stream, sequence}` and `domain.Stream.Append` assigns the sequence
// from the length of the stream it is called on. A caller appending to its own
// copy therefore computes a sequence from a stale length — and two callers
// appending from the same base compute the *same* `Ref`. Measured before this
// changed: two appends, one surviving fact, both carrying `acct-1#1`, so a
// reference already recorded as a derivation input silently re-pointed.
//
// A sequence can only be assigned where writes serialise. That is here.
type Store interface {
	// Load returns the stream, or ErrStreamNotFound.
	Load(ctx context.Context, name string) (domain.Stream, error)

	// Append records one fact and returns the reference the store assigned.
	//
	// The stream is created if it does not exist: a stream is the facts in it,
	// so there is nothing to declare in advance.
	//
	// Returns ErrStaleRead if `expect` is not satisfied, and
	// ErrNonMonotonicKnowledge if the envelope's knowledge time is not after
	// the stream's last.
	Append(
		ctx context.Context,
		name string,
		expect Expectation,
		envelope domain.Envelope,
		kind domain.Kind,
		payload domain.Payload,
	) (domain.Ref, error)
}
