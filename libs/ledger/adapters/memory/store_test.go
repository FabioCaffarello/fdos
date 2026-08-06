package memory_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/memory"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// The defect that produced ADR-0034, now unrepresentable rather than checked.
//
// Before the port change, two callers appending from the same base each computed
// `acct-1#1` — `domain.Stream.Append` derives the sequence from the length of
// the stream it is called on, and both had read length 0. One fact was lost and
// a Ref already handed out addressed a different fact.
//
// The store now assigns the sequence from the stream it holds, under its own
// lock, so concurrent appends receive consecutive refs and neither is lost.
func TestConcurrentAppendsGetConsecutiveRefsAndBothSurvive(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	refA, err := store.Append(ctx, "acct-1", app.Any(), envelopeAt(t, 1), domain.KindObservation, claimFor(t, "PETR4"))
	if err != nil {
		t.Fatalf("append A: %v", err)
	}
	refB, err := store.Append(ctx, "acct-1", app.Any(), envelopeAt(t, 2), domain.KindObservation, claimFor(t, "VALE3"))
	if err != nil {
		t.Fatalf("append B: %v", err)
	}

	if refA == refB {
		t.Fatalf("two appends were assigned the same ref: %s", refA)
	}
	if refA.Sequence != 1 || refB.Sequence != 2 {
		t.Errorf("sequences are %d and %d, want 1 and 2", refA.Sequence, refB.Sequence)
	}

	loaded, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Len() != 2 {
		t.Fatalf("stream holds %d facts, want both", loaded.Len())
	}
}

// The precondition that keeps a decision made on a read from being applied to a
// stream that has moved (ADR-0034).
//
// This is the shape MintIdentity depends on: it resolves, finds nothing, and
// must not append if a mint has landed since.
func TestAnAppendOnAStaleReadIsRefused(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	// Two callers each read an empty stream.
	const readAt = 0

	if _, err := store.Append(
		ctx, "acct-1", app.AtLength(readAt), envelopeAt(t, 1), domain.KindObservation, claimFor(t, "PETR4"),
	); err != nil {
		t.Fatalf("the first append at the length it read was refused: %v", err)
	}

	_, err := store.Append(
		ctx, "acct-1", app.AtLength(readAt), envelopeAt(t, 2), domain.KindObservation, claimFor(t, "VALE3"))
	if !errors.Is(err, app.ErrStaleRead) {
		t.Fatalf("want ErrStaleRead, got %v", err)
	}
	if !strings.Contains(err.Error(), "acct-1") {
		t.Errorf("the rejection does not name the stream: %v", err)
	}

	loaded, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Len() != 1 {
		t.Errorf("the refused append was applied anyway: %d facts", loaded.Len())
	}
}

// Any() is not a weaker version of the same check — it is the answer for a
// caller whose decision did not depend on a read. Admission uses it, so a
// producer's submission is never rejected because somebody else wrote first.
func TestAnyAppendsRegardlessOfWhatLanded(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	for i, ticker := range []string{"PETR4", "VALE3", "ITUB4"} {
		if _, err := store.Append(
			ctx, "acct-1", app.Any(), envelopeAt(t, i+1), domain.KindObservation, claimFor(t, ticker),
		); err != nil {
			t.Fatalf("appending %s was refused: %v", ticker, err)
		}
	}

	loaded, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Len() != 3 {
		t.Errorf("stream holds %d facts, want 3", loaded.Len())
	}
}

// Knowledge time is monotonic per stream (ADR-0009). One writer satisfies that
// by construction; two satisfy it only because of this check.
//
// The refusal is honest rather than defensive: if two facts cannot be ordered on
// the axis that decides what was knowable when, FDOS cannot record both and
// pretend to know which came first.
func TestAnAppendThatGoesBackwardsInKnowledgeIsRefused(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	if _, err := store.Append(
		ctx, "acct-1", app.Any(), envelopeAt(t, 5), domain.KindObservation, claimFor(t, "PETR4"),
	); err != nil {
		t.Fatal(err)
	}

	for _, tc := range []struct {
		name string
		hour int
	}{
		{"earlier", 4},
		{"identical", 5},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := store.Append(
				ctx, "acct-1", app.Any(), envelopeAt(t, tc.hour), domain.KindObservation, claimFor(t, "VALE3"))
			if !errors.Is(err, app.ErrNonMonotonicKnowledge) {
				t.Fatalf("want ErrNonMonotonicKnowledge, got %v", err)
			}
		})
	}
}

// A stream is the facts in it, so there is nothing to declare in advance.
func TestTheFirstAppendCreatesTheStream(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	if _, err := store.Load(ctx, "acct-1"); !errors.Is(err, app.ErrStreamNotFound) {
		t.Fatalf("want ErrStreamNotFound before any append, got %v", err)
	}
	if _, err := store.Append(
		ctx, "acct-1", app.AtLength(0), envelopeAt(t, 1), domain.KindObservation, claimFor(t, "PETR4"),
	); err != nil {
		t.Fatalf("the first append was refused: %v", err)
	}
	if _, err := store.Load(ctx, "acct-1"); err != nil {
		t.Fatalf("the stream was not created: %v", err)
	}
}

// --- fixture -----------------------------------------------------------------

func claimFor(t *testing.T, ticker string) domain.HoldingClaimed {
	t.Helper()
	return domain.HoldingClaimed{
		Account:    identity.MustClaim("account_number", "0001234-5"),
		Instrument: identity.MustClaim("ticker", ticker),
		Quantity:   money.MustParseQuantity("100", "share"),
	}
}

// envelopeAt builds an envelope whose knowledge time is `hour` hours past the
// fixture epoch, so a test can order or deliberately misorder appends.
func envelopeAt(t *testing.T, hour int) domain.Envelope {
	t.Helper()
	epoch := time.Date(2026, time.March, 1, 0, 0, 0, 0, time.UTC)
	effective := temporal.MustAt(epoch)
	knowledge := temporal.MustAt(epoch.Add(time.Duration(hour) * time.Hour))

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
