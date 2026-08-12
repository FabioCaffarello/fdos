// Package storetest is the conformance suite every [app.Store] must pass.
//
// It exists because ADR-0034 anticipated a second engine — *one conformance
// suite, two implementations*. A suite living inside an adapter cannot serve
// that: `libs/ledger-postgres` would have to depend on `libs/ledger-sqlite`
// merely to be tested. So the suite lives in the context module, beside the port
// it defines the meaning of.
//
// It is exported rather than internal for the same reason `app.Store` is
// exported. The port is public API; what implementing it *means* is public API
// too, and an out-of-tree adapter that cannot run the suite is an adapter nobody
// can hold to the contract.
//
// The suite imports no adapter. It takes a factory, so it knows nothing about
// what it is testing — which is what lets the in-memory store and a durable one
// be held to a single definition.
package storetest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// NewStore returns a store with no streams in it.
//
// Called once per case, so cases cannot leak into one another. An
// implementation backed by a file should hand back a fresh one — `t.TempDir`
// is the usual way.
type NewStore func(t *testing.T) app.Store

// Run executes the suite against an implementation.
//
//	func TestConformance(t *testing.T) {
//	    storetest.Run(t, func(t *testing.T) app.Store { return memory.NewStore() })
//	}
func Run(t *testing.T, newStore NewStore) {
	t.Helper()

	t.Run("AppendsGetConsecutiveRefs", func(t *testing.T) { appendsGetConsecutiveRefs(t, newStore) })
	t.Run("UnknownStreamIsNotFound", func(t *testing.T) { unknownStreamIsNotFound(t, newStore) })
	t.Run("FirstAppendCreatesTheStream", func(t *testing.T) { firstAppendCreatesTheStream(t, newStore) })
	t.Run("ExpectationHoldsWhenNothingMoved", func(t *testing.T) { expectationHolds(t, newStore) })
	t.Run("StaleReadIsRefusedAndNotApplied", func(t *testing.T) { staleReadIsRefused(t, newStore) })
	t.Run("AnyAppendsRegardless", func(t *testing.T) { anyAppendsRegardless(t, newStore) })
	t.Run("KnowledgeTimeCannotGoBackwards", func(t *testing.T) { knowledgeCannotGoBackwards(t, newStore) })
	t.Run("SubSecondKnowledgeTimesOrderChronologically", func(t *testing.T) {
		subSecondKnowledgeOrders(t, newStore)
	})
	t.Run("FactsReloadIntact", func(t *testing.T) { factsReloadIntact(t, newStore) })
	t.Run("AsOfReadMatchesTheProjection", func(t *testing.T) { asOfMatchesProjection(t, newStore) })
	t.Run("ConcurrentAppendsSerialise", func(t *testing.T) { concurrentAppendsSerialise(t, newStore) })
	t.Run("SerialiseRemovesTheRetry", func(t *testing.T) { serialiseRemovesTheRetry(t, newStore) })
	t.Run("NestedSerialiseIsRefused", func(t *testing.T) { nestedSerialiseIsRefused(t, newStore) })
	t.Run("SerialiseReturnsTheCallbacksError", func(t *testing.T) { serialiseReturnsTheError(t, newStore) })
	t.Run("SerialiseDoesNotAppendOnItsOwn", func(t *testing.T) { serialiseAppendsNothing(t, newStore) })
}

// Case 1. The defect the M10 gate measured: `Ref` is {stream, sequence} and a
// caller appending to its own copy derives the sequence from a stale length, so
// two callers appending from the same base computed the same `Ref` and one fact
// was lost. The store assigns it now, so this is the property that must hold.
func appendsGetConsecutiveRefs(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	first := mustAppend(t, ctx, store, "acct-1", app.Any(), 1, "PETR4")
	second := mustAppend(t, ctx, store, "acct-1", app.Any(), 2, "VALE3")

	if first == second {
		t.Fatalf("two appends were assigned the same ref: %s", first)
	}
	if first.Sequence != 1 || second.Sequence != 2 {
		t.Errorf("sequences are %d and %d, want 1 and 2", first.Sequence, second.Sequence)
	}
}

// Case 2. A stream nobody has written is absent, not empty. "We know nothing"
// and "we hold nothing" are different answers (ADR-0022).
func unknownStreamIsNotFound(t *testing.T, newStore NewStore) {
	_, err := newStore(t).Load(context.Background(), "never-written")
	if !errors.Is(err, app.ErrStreamNotFound) {
		t.Fatalf("want ErrStreamNotFound, got %v", err)
	}
}

// Case 3. A stream is the facts in it, so there is nothing to declare first.
func firstAppendCreatesTheStream(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	if _, err := store.Append(
		ctx, "acct-1", app.AtLength(0), envelopeAt(t, 1), domain.KindObservation, claim(t, "PETR4"),
	); err != nil {
		t.Fatalf("the first append was refused: %v", err)
	}
	if _, err := store.Load(ctx, "acct-1"); err != nil {
		t.Fatalf("the stream was not created: %v", err)
	}
}

// Case 4. The precondition must not fire when nothing has moved, or every
// caller learns to pass Any() and the mechanism is decoration.
func expectationHolds(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	mustAppend(t, ctx, store, "acct-1", app.AtLength(0), 1, "PETR4")
	mustAppend(t, ctx, store, "acct-1", app.AtLength(1), 2, "VALE3")

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Len() != 2 {
		t.Errorf("stream holds %d facts, want 2", stream.Len())
	}
}

// Case 5, and the one to write carefully.
//
// A store that returns the error *and* appends anyway passes a version of this
// test that only checks the error — which is the M10 defect in a different
// disguise. The second half is the half that matters.
func staleReadIsRefused(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	// Two callers each read the stream at length 0.
	mustAppend(t, ctx, store, "acct-1", app.AtLength(0), 1, "PETR4")

	_, err := store.Append(
		ctx, "acct-1", app.AtLength(0), envelopeAt(t, 2), domain.KindObservation, claim(t, "VALE3"))
	if !errors.Is(err, app.ErrStaleRead) {
		t.Fatalf("want ErrStaleRead, got %v", err)
	}

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Len() != 1 {
		t.Fatalf("the refused append was applied anyway: stream holds %d facts", stream.Len())
	}
	held, ok := stream.Facts()[0].Payload().(domain.HoldingClaimed)
	if !ok || held.Instrument.Value() != "PETR4" {
		t.Errorf("the refused append overwrote the accepted one: %v", stream.Facts()[0].Payload())
	}
}

// Case 6. Any() is not a weaker precondition — it is the right answer for a
// caller whose decision did not depend on a read. Admission uses it, so a
// producer's submission is never rejected because somebody else wrote first.
func anyAppendsRegardless(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	for i, ticker := range []string{"PETR4", "VALE3", "ITUB4"} {
		mustAppend(t, ctx, store, "acct-1", app.Any(), i+1, ticker)
	}

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Len() != 3 {
		t.Errorf("stream holds %d facts, want 3", stream.Len())
	}
}

// Case 7. Knowledge time is monotonic per stream (ADR-0009). One writer
// satisfies that by construction; more than one satisfies it only because the
// store refuses otherwise.
func knowledgeCannotGoBackwards(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)
	mustAppend(t, ctx, store, "acct-1", app.Any(), 5, "PETR4")

	for name, hour := range map[string]int{"earlier": 4, "identical": 5} {
		t.Run(name, func(t *testing.T) {
			_, err := store.Append(
				ctx, "acct-1", app.Any(), envelopeAt(t, hour), domain.KindObservation, claim(t, "VALE3"))
			if !errors.Is(err, app.ErrNonMonotonicKnowledge) {
				t.Fatalf("want ErrNonMonotonicKnowledge, got %v", err)
			}
		})
	}
}

// Case 7b. The region the suite could not reach, and the defect it hid.
//
// Every other case builds knowledge times from whole hours, which render as a
// fixed-width `…T0N:00:00Z`. A store that compares *formatted* timestamps as
// strings agrees with chronology exactly there and nowhere else: RFC3339 grants
// string-sortability only when all times carry the same number of fractional
// digits (RFC 3339 §5.1), and Go's RFC3339Nano strips trailing zeros, so a
// sub-second instant renders shorter and `.` (0x2E) sorts below `Z` (0x5A).
//
// Measured before this case existed: the SQLite store refused an append the
// in-memory store accepted, because `2026-…T00:00:00.000000001Z` compared as
// *earlier* than `2026-…T00:00:00Z`. Two implementations of one port disagreeing
// about whether time moved forward, with the shared suite green.
//
// A single nanosecond, deliberately: it is the smallest step the temporal type
// admits, so it is the case a formatted comparison is most likely to get wrong,
// and the one an as-of read has to get right.
func subSecondKnowledgeOrders(t *testing.T, newStore NewStore) {
	ctx := context.Background()

	t.Run("a nanosecond later is later", func(t *testing.T) {
		store := newStore(t)
		second := time.Hour

		if _, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAfter(t, second), domain.KindObservation, claim(t, "PETR4"),
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
		if _, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAfter(t, second+time.Nanosecond),
			domain.KindObservation, claim(t, "VALE3"),
		); err != nil {
			t.Fatalf("an append one nanosecond later was refused: %v — the store is comparing "+
				"rendered timestamps rather than instants", err)
		}
	})

	t.Run("a fraction earlier is earlier", func(t *testing.T) {
		store := newStore(t)
		second := time.Hour

		if _, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAfter(t, second+500*time.Millisecond),
			domain.KindObservation, claim(t, "PETR4"),
		); err != nil {
			t.Fatalf("seed: %v", err)
		}
		_, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAfter(t, second), domain.KindObservation, claim(t, "VALE3"))
		if !errors.Is(err, app.ErrNonMonotonicKnowledge) {
			t.Fatalf("an append half a second earlier was accepted (err=%v); knowledge time "+
				"cannot go backwards", err)
		}
	})
}

// Case 8. Nothing is lost or altered in the round trip. For an in-memory store
// this is nearly free; for a durable one it is the whole point, and the
// adapter's own tests must additionally close and reopen the database.
func factsReloadIntact(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	want := []string{"PETR4", "VALE3", "ITUB4"}
	refs := make([]domain.Ref, 0, len(want))
	for i, ticker := range want {
		refs = append(refs, mustAppend(t, ctx, store, "acct-1", app.Any(), i+1, ticker))
	}

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	facts := stream.Facts()
	if len(facts) != len(want) {
		t.Fatalf("stream holds %d facts, want %d", len(facts), len(want))
	}
	for i, f := range facts {
		if f.Ref() != refs[i] {
			t.Errorf("fact %d reloaded as %s, want %s", i, f.Ref(), refs[i])
		}
		held, ok := f.Payload().(domain.HoldingClaimed)
		if !ok {
			t.Fatalf("fact %d is not a HoldingClaimed", i)
		}
		if held.Instrument.Value() != want[i] {
			t.Errorf("fact %d is %s, want %s", i, held.Instrument.Value(), want[i])
		}
		if f.Envelope().Provenance().IsZero() {
			t.Errorf("fact %d reloaded without provenance", i)
		}
		if f.Envelope().Coordinates().Knowledge().IsZero() {
			t.Errorf("fact %d reloaded without knowledge time", i)
		}
	}
}

// Case 9. An as-of read must answer what the facts say, not what an index
// happens to hold. Nearly tautological for a store that keeps a domain.Stream;
// the case exists for the one that keeps rows and an index, where a wrong index
// silently changes the answer rather than failing.
func asOfMatchesProjection(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)
	for i, ticker := range []string{"PETR4", "VALE3", "ITUB4"} {
		mustAppend(t, ctx, store, "acct-1", app.Any(), i+1, ticker)
	}

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}

	// Knowledge time 2 was assigned to the second fact, so a read there sees two.
	visible := stream.VisibleAt(asOf(t, 2))
	if len(visible) != 2 {
		t.Fatalf("as-of read returned %d facts, want the 2 known by then", len(visible))
	}
	// And reading later sees all three: the earlier read was not a truncation.
	if all := stream.VisibleAt(asOf(t, 99)); len(all) != 3 {
		t.Errorf("a later read returned %d facts, want 3", len(all))
	}
}

// Case 10. The serialisation point must serialise.
//
// The retry loop is not test scaffolding — it is the caller contract this design
// implies, and writing it here is how that became visible. Concurrent writers
// contend on **two** things: the Expectation, and knowledge-time monotonicity
// (ADR-0009). A writer whose knowledge time is overtaken between reading the
// clock and reaching the store is refused, and must take a later one and retry.
//
// RFC-0014 left "what is the retry contract" open. This is the shape of it.
func concurrentAppendsSerialise(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	const writers = 8
	var tick atomic.Int64
	tick.Store(1)

	var wg sync.WaitGroup
	errs := make(chan error, writers)

	for i := range writers {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			// Bounded: a livelock must fail the test rather than hang it.
			for attempt := range 200 {
				_, err := store.Append(ctx, "acct-1", app.Any(),
					envelopeAt(t, int(tick.Add(1))), domain.KindObservation, claim(t, fmt.Sprintf("T%d", n)))
				if err == nil {
					return
				}
				if !errors.Is(err, app.ErrNonMonotonicKnowledge) {
					errs <- fmt.Errorf("writer %d: %w", n, err)
					return
				}
				_ = attempt
			}
			errs <- fmt.Errorf("writer %d never landed in 200 attempts", n)
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Len() != writers {
		t.Fatalf("stream holds %d facts, want %d — appends were lost", stream.Len(), writers)
	}
	seen := map[uint64]bool{}
	for _, f := range stream.Facts() {
		if seen[f.Ref().Sequence] {
			t.Fatalf("sequence %d was assigned twice", f.Ref().Sequence)
		}
		seen[f.Ref().Sequence] = true
	}
	for want := uint64(1); want <= writers; want++ {
		if !seen[want] {
			t.Errorf("sequence %d was never assigned; the sequence has a hole", want)
		}
	}
}

// --- fixture -----------------------------------------------------------------

// epoch is fixed rather than read from a clock: a suite that read the wall clock
// could not be replayed, which is the property it exists to protect.
var epoch = time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)

func mustAppend(
	t *testing.T,
	ctx context.Context,
	store app.Store,
	stream string,
	expect app.Expectation,
	hour int,
	ticker string,
) domain.Ref {
	t.Helper()
	ref, err := store.Append(
		ctx, stream, expect, envelopeAt(t, hour), domain.KindObservation, claim(t, ticker))
	if err != nil {
		t.Fatalf("append %s at hour %d: %v", ticker, hour, err)
	}
	return ref
}

func claim(t *testing.T, ticker string) domain.HoldingClaimed {
	t.Helper()
	return domain.HoldingClaimed{
		Account:    identity.MustClaim("account_number", "0001234-5"),
		Instrument: identity.MustClaim("ticker", ticker),
		Quantity:   money.MustParseQuantity("100", "share"),
	}
}

// envelopeAt builds an envelope whose knowledge time is `hour` hours past the
// epoch, so a case can order — or deliberately misorder — appends.
//
// Whole hours, which is convenient and was also a blind spot: every knowledge
// time it produces renders as a fixed-width `…T0N:00:00Z` with no fractional
// part, and that is the one region where lexicographic and chronological order
// coincide. A store comparing formatted timestamps as strings passes every case
// built from this helper while being wrong about any instant with a fraction.
// Use [envelopeAfter] to reach that region.
func envelopeAt(t *testing.T, hour int) domain.Envelope {
	t.Helper()
	return envelopeAfter(t, time.Duration(hour)*time.Hour)
}

// envelopeAfter builds an envelope whose knowledge time is `d` past the epoch,
// at whatever precision `d` carries.
func envelopeAfter(t *testing.T, d time.Duration) domain.Envelope {
	t.Helper()
	effective := temporal.MustAt(epoch)
	knowledge := temporal.MustAt(epoch.Add(d))

	interval, err := temporal.OpenFrom(effective)
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := temporal.Assign(interval, knowledge)
	if err != nil {
		t.Fatal(err)
	}
	source, err := provenance.NewSource(
		"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	prov, err := provenance.Observed(source, effective, provenance.Unmediated(), provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.NewEnvelope(coordinates, prov, nil)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func asOf(t *testing.T, hour int) temporal.AsOf {
	t.Helper()
	a, err := temporal.NewAsOf(
		temporal.MustAt(epoch.Add(time.Hour)), temporal.MustAt(epoch.Add(time.Duration(hour)*time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// Case 12. What `Serialise` buys, stated as the difference from case 11
// (ADR-0041).
//
// Case 11 has eight writers retrying up to 200 times each, and passes. That is
// the honest picture of the port *without* a region: every writer computes what
// it will append from something it read, the read goes stale, and the append is
// refused. Retrying works and hides that the ledger refuses most of the traffic
// it was built to accept — measured at 3 to 9 admitted of 32 before ADR-0036,
// and 106 of 128 across sixteen processes before this decision.
//
// Here every writer gets **one attempt**. The knowledge time each appends is
// computed from the stream's length read inside its own region, which is the
// analogue of reading the clock: two writers that read the same length compute
// the same instant, and the second is refused as non-monotonic. So a store that
// does not serialise fails this immediately, and no amount of retrying is
// involved in passing it.
func serialiseRemovesTheRetry(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	const writers = 8
	var wg sync.WaitGroup
	errs := make(chan error, writers)

	for i := range writers {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			err := store.Serialise(ctx, "acct-1", func(ctx context.Context, s app.Store) error {
				// Read-then-append, the shape the region exists to make atomic.
				// An absent stream is length zero: nothing to have gone stale.
				length := 0
				if stream, err := s.Load(ctx, "acct-1"); err == nil {
					length = stream.Len()
				} else if !errors.Is(err, app.ErrStreamNotFound) {
					return err
				}
				_, err := s.Append(ctx, "acct-1", app.AtLength(length),
					envelopeAt(t, length+1), domain.KindObservation, claim(t, fmt.Sprintf("T%d", n)))
				return err
			})
			if err != nil {
				errs <- fmt.Errorf("writer %d landed no fact on its only attempt: %w", n, err)
			}
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Error(err)
	}

	stream, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Len() != writers {
		t.Fatalf("stream holds %d facts, want %d", stream.Len(), writers)
	}
	// Sequences 1..N with no gap is the same integrity property case 1 asserts,
	// re-checked here because a region that let two writers compute one sequence
	// would satisfy the count above while losing a fact.
	for i, f := range stream.Facts() {
		if want := uint64(i + 1); f.Ref().Sequence != want { //nolint:gosec // i is a slice index
			t.Fatalf("fact %d carries sequence %d, want %d", i, f.Ref().Sequence, want)
		}
	}
}

// Case 13. Entering a region from inside one is refused rather than hung.
//
// The honest outcome of nesting is a deadlock against a lock the caller already
// holds, and a deadlock is a hang with no message — in a suite, a timeout with
// no cause. An implementation is required to say so instead.
func nestedSerialiseIsRefused(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	err := store.Serialise(ctx, "acct-1", func(ctx context.Context, s app.Store) error {
		return s.Serialise(ctx, "acct-1", func(context.Context, app.Store) error {
			t.Error("a nested region ran; it must be refused")
			return nil
		})
	})
	if !errors.Is(err, app.ErrNestedSerialise) {
		t.Fatalf("nested Serialise returned %v, want ErrNestedSerialise", err)
	}
}

// Case 14. The callback's error survives the trip out.
//
// A store that wrapped errors opaquely would break every caller's ability to
// tell ErrStaleRead — *re-read and try again* — from a bug, which ADR-0034 made
// a distinct error precisely so a caller would not retry the bug forever.
func serialiseReturnsTheError(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	sentinel := errors.New("storetest: the callback said no")
	err := store.Serialise(ctx, "acct-1", func(context.Context, app.Store) error { return sentinel })
	if !errors.Is(err, sentinel) {
		t.Fatalf("Serialise returned %v, which does not wrap the callback's error", err)
	}

	// And a real one from inside, so the path a caller actually hits is covered
	// rather than only a sentinel that never touched the store.
	if _, seedErr := store.Append(ctx, "acct-1", app.Any(),
		envelopeAt(t, 1), domain.KindObservation, claim(t, "PETR4")); seedErr != nil {
		t.Fatal(seedErr)
	}
	err = store.Serialise(ctx, "acct-1", func(ctx context.Context, s app.Store) error {
		_, appendErr := s.Append(ctx, "acct-1", app.AtLength(99),
			envelopeAt(t, 2), domain.KindObservation, claim(t, "VALE3"))
		return appendErr
	})
	if !errors.Is(err, app.ErrStaleRead) {
		t.Fatalf("Serialise returned %v, want it to wrap ErrStaleRead", err)
	}
}

// Case 15. A region that appends nothing appends nothing.
//
// Obvious, and the reason to assert it is not. An implementation binding the
// region to a transaction has somewhere to accidentally commit an empty one, or
// to create the stream as a side effect of being asked to serialise against its
// name — which would make "we know nothing" and "we hold nothing" the same
// answer, the distinction ADR-0022 turns on.
func serialiseAppendsNothing(t *testing.T, newStore NewStore) {
	ctx, store := context.Background(), newStore(t)

	if err := store.Serialise(ctx, "acct-1", func(context.Context, app.Store) error {
		return nil
	}); err != nil {
		t.Fatalf("empty region: %v", err)
	}

	if _, err := store.Load(ctx, "acct-1"); !errors.Is(err, app.ErrStreamNotFound) {
		t.Fatalf("after an empty region the stream loaded with %v, want ErrStreamNotFound", err)
	}
}
