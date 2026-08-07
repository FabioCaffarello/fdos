package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"pgregory.net/rapid"

	"github.com/FabioCaffarello/fdos/libs/kernel/explained"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/memory"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// The property the whole bitemporal design exists to guarantee (ADR-0009):
//
//	A projection at knowledge time K is unchanged by appending any fact whose
//	knowledge time is after K.
//
// This is what "no look-ahead bias" means mechanically. A backtest that
// violates it reports returns that were never achievable, and the violation is
// invisible — the numbers look plausible.
func TestProjectionIsUnchangedByLaterKnowledge(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		fixture := newFixture(t)

		// What was known by the cutoff.
		earlyCount := rapid.IntRange(1, 4).Draw(t, "early")
		var lastEarly string
		for i := range earlyCount {
			lastEarly = fixture.observe(t, day(1+i), qty(t, i+1))
		}
		cutoff := fixture.clock.Now()

		asOf, err := temporal.NewAsOf(day(10), cutoff)
		if err != nil {
			t.Fatalf("as-of: %v", err)
		}
		before, err := fixture.project(asOf)
		if err != nil {
			t.Fatalf("project before: %v", err)
		}

		// Facts learned afterwards, including ones effective in the past —
		// the case that silently contaminates a naive implementation.
		lateCount := rapid.IntRange(1, 4).Draw(t, "late")
		for i := range lateCount {
			fixture.observe(t, day(1+i), qty(t, 100+i))
		}

		after, err := fixture.project(asOf)
		if err != nil {
			t.Fatalf("project after: %v", err)
		}

		if before.Value().Quantity.String() != after.Value().Quantity.String() {
			t.Fatalf("later knowledge changed a past projection: %s -> %s (last early %s)",
				before.Value().Quantity, after.Value().Quantity, lastEarly)
		}
		if before.Trace().Hash() != after.Trace().Hash() {
			t.Fatalf("later knowledge changed the derivation: %s -> %s",
				before.Trace(), after.Trace())
		}
	})
}

// Constitution §4: history is never rewritten.
//
// The test that used to live here drove `store.Save` with a shorter stream and
// asserted a rejection. There is no longer an operation to drive: ADR-0034
// removed the whole-stream write, so shortening a stream is unrepresentable
// rather than refused. That moved the guarantee from rung 3 to rung 1, and a
// rung-1 rule has nothing left for a test to exercise — the compiler is the
// test.
//
// What replaced it, in `adapters/memory`, are the properties that *are* still
// checkable: a stale read is refused, knowledge time cannot go backwards, and
// two concurrent appends receive consecutive refs rather than the same one.

// A correction is a new fact, not a mutation. Asking as of a knowledge time
// before the correction still returns what FDOS believed then — which is what
// "what did we believe on date D" means, and what an audit requires.
func TestCorrectionLeavesTheOriginalReadable(t *testing.T) {
	fixture := newFixture(nil)

	ref := fixture.observeAt(day(1), money.MustParseQuantity("100", "share"))
	beforeCorrection := fixture.clock.Now()

	if _, err := fixture.ledger.CorrectFact(context.Background(), app.CorrectFactCommand{
		Stream:      streamName,
		Corrects:    ref,
		Kind:        domain.CorrectionKindRetracted,
		Reason:      "statement line was a duplicate",
		Source:      testSource,
		CollectedAt: day(1),
		Interpreter: testInterpreter,
		Confidence:  provenance.ConfidenceAsserted,
	}); err != nil {
		t.Fatalf("correct: %v", err)
	}

	// As of before the correction: the original still stands.
	asOfBefore, _ := temporal.NewAsOf(day(5), beforeCorrection)
	original, err := fixture.project(asOfBefore)
	if err != nil {
		t.Fatalf("project before correction: %v", err)
	}
	if got := original.Value().Quantity.String(); got != "100" {
		t.Errorf("as of before the correction, expected 100, got %s", got)
	}

	// As of now: the retraction is visible and there is nothing left to project.
	asOfAfter, _ := temporal.NewAsOf(day(5), fixture.clock.Now())
	if _, err := fixture.project(asOfAfter); !errors.Is(err, domain.ErrNoFacts) {
		t.Errorf("after retraction expected ErrNoFacts, got %v", err)
	}
}

// Constitution §8: a computed financial value travels with its derivation. The
// trace names every fact consumed, so the explanation is data rather than a
// story told about a number afterwards.
func TestProjectionCarriesItsDerivation(t *testing.T) {
	fixture := newFixture(nil)
	fixture.observeAt(day(1), money.MustParseQuantity("10", "share"))
	fixture.observeAt(day(2), money.MustParseQuantity("25", "share"))

	asOf, _ := temporal.NewAsOf(day(3), fixture.clock.Now())
	result, err := fixture.project(asOf)
	if err != nil {
		t.Fatalf("project: %v", err)
	}

	if result.Trace().IsZero() {
		t.Fatal("projection produced no derivation reference")
	}
	if got := len(result.Record().Inputs()); got != 2 {
		t.Errorf("expected 2 inputs in the derivation, got %d", got)
	}
	if result.Record().Method().Name() != "ledger.ProjectPosition" {
		t.Errorf("unexpected method: %s", result.Record().Method())
	}

	// The as-of is a recorded parameter: a trace that does not say which
	// coordinates produced the number cannot be used to reproduce it.
	var sawAsOf bool
	for _, p := range result.Record().Parameters() {
		if p.Name == "as_of" {
			sawAsOf = true
		}
	}
	if !sawAsOf {
		t.Error("derivation does not record the as-of coordinates")
	}
}

// The same inputs must produce the same content address, or nothing downstream
// can deduplicate or verify a derivation.
func TestDerivationIsDeterministic(t *testing.T) {
	build := func() string {
		f := newFixture(nil)
		f.observeAt(day(1), money.MustParseQuantity("10", "share"))
		f.observeAt(day(2), money.MustParseQuantity("25", "share"))
		asOf, _ := temporal.NewAsOf(day(3), f.clock.Now())
		result, err := f.project(asOf)
		if err != nil {
			t.Fatalf("project: %v", err)
		}
		return result.Trace().Hash()
	}
	if first, second := build(), build(); first != second {
		t.Fatalf("derivation is not deterministic: %s vs %s", first, second)
	}
}

// --- fixture -----------------------------------------------------------------

const streamName = "acct-1"

var (
	testAccount     = identity.MustDerive(identity.KindAccount, "acct-1")
	testInstrument  = identity.MustDerive(identity.KindInstrument, "BRPETRACNOR9")
	testSource      = mustSource("broker.statement")
	testInterpreter = mustInterpreter("statement-parser", "1.0.0")
)

type explainedPosition = explained.Value[domain.Position]

type fixture struct {
	ledger *app.Ledger
	clock  *clock.Sequence
	seq    int
}

func newFixture(t *rapid.T) *fixture {
	// Knowledge time starts well after any effective time used here, so
	// "learned later than it was true" is the normal case rather than an edge.
	seq := clock.NewSequence(temporal.MustAt(time.Unix(1_000_000, 0).UTC()), time.Hour)
	ledger, err := app.NewLedger(memory.NewStore(), seq, identity.Canonicalisation())
	if err != nil {
		if t != nil {
			t.Fatalf("new ledger: %v", err)
		}
		panic(err)
	}
	return &fixture{ledger: ledger, clock: seq}
}

func (f *fixture) observe(t *rapid.T, effective temporal.Instant, quantity money.Quantity) string {
	ref, err := f.appendObservation(effective, quantity)
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	return ref.String()
}

func (f *fixture) observeAt(effective temporal.Instant, quantity money.Quantity) domain.Ref {
	ref, err := f.appendObservation(effective, quantity)
	if err != nil {
		panic(err)
	}
	return ref
}

func (f *fixture) appendObservation(effective temporal.Instant, quantity money.Quantity) (domain.Ref, error) {
	f.seq++
	return f.ledger.ObserveHolding(context.Background(), app.ObserveHoldingCommand{
		Stream:      streamName,
		Account:     testAccount,
		Instrument:  testInstrument,
		Quantity:    quantity,
		Effective:   mustOpen(effective),
		Source:      testSource,
		CollectedAt: effective,
		Interpreter: testInterpreter,
		Confidence:  provenance.ConfidenceAsserted,
	})
}

func (f *fixture) project(asOf temporal.AsOf) (explainedPosition, error) {
	return f.ledger.ProjectPosition(context.Background(), app.ProjectPositionQuery{
		Stream:     streamName,
		Account:    testAccount,
		Instrument: testInstrument,
		AsOf:       asOf,
	})
}

func qty(t *rapid.T, n int) money.Quantity {
	q, err := money.ParseQuantity(itoa(n), "share")
	if err != nil {
		t.Fatalf("quantity: %v", err)
	}
	return q
}

func day(n int) temporal.Instant {
	return temporal.MustAt(time.Date(2026, time.January, n, 0, 0, 0, 0, time.UTC))
}

func mustOpen(from temporal.Instant) temporal.Interval {
	iv, err := temporal.OpenFrom(from)
	if err != nil {
		panic(err)
	}
	return iv
}

func mustSource(v string) provenance.Source {
	s, err := provenance.NewSource(v)
	if err != nil {
		panic(err)
	}
	return s
}

func mustInterpreter(name, version string) provenance.Interpreter {
	i, err := provenance.NewInterpreter(name, version)
	if err != nil {
		panic(err)
	}
	return i
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
