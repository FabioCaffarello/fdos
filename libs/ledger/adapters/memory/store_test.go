package memory_test

import (
	"context"
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

// The defect this slice exists to close, and the one the M10 gate measured.
//
// `Save` rejected a stream *shorter* than the one held — history is never
// rewritten — and accepted an **equal-length** one. Two callers loading the
// same stream, each appending a different fact, each saving at the same length:
// both saves returned nil, and the first fact was gone with nothing recording
// that it had ever existed.
//
// That is Constitution §4 violated by the store that exists to uphold it. A
// ledger may refuse a write; it may not accept one and drop another.
//
// The second assertion is the sharper one. Both appends were assigned the
// **same Ref**, so the losing caller holds a reference that now addresses a
// different fact — including a Ref already recorded as a derivation input.
func TestSaveRefusesAWriteThatWouldLoseAFact(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	base, err := domain.NewStream("acct-1")
	if err != nil {
		t.Fatal(err)
	}
	env := envelopeAt(t)

	withA, refA, err := base.Append(env, domain.KindObservation, claimFor(t, "PETR4"))
	if err != nil {
		t.Fatal(err)
	}
	withB, refB, err := base.Append(env, domain.KindObservation, claimFor(t, "VALE3"))
	if err != nil {
		t.Fatal(err)
	}
	if refA != refB {
		t.Fatalf("fixture is not discriminating: the two appends got different refs, %s and %s", refA, refB)
	}

	if saveErr := store.Save(ctx, withA); saveErr != nil {
		t.Fatalf("save A: %v", saveErr)
	}

	err = store.Save(ctx, withB)
	if err == nil {
		t.Fatal("the second concurrent append was accepted; one fact was silently lost")
	}
	if !strings.Contains(err.Error(), "acct-1") {
		t.Errorf("the rejection does not name the stream: %v", err)
	}

	loaded, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Len() != 1 {
		t.Fatalf("stream length is %d, want 1", loaded.Len())
	}
	survivor, ok := loaded.Facts()[0].Payload().(domain.HoldingClaimed)
	if !ok {
		t.Fatal("the surviving fact is not a HoldingClaimed")
	}
	if survivor.Instrument.Value() != "PETR4" {
		t.Errorf("the rejected write overwrote the accepted one: survivor is %s", survivor.Instrument)
	}
}

// The rule that was already there, and must not regress: a shorter stream is a
// rewrite of history wearing the costume of a write.
func TestSaveRefusesToShortenAStream(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	base, err := domain.NewStream("acct-1")
	if err != nil {
		t.Fatal(err)
	}
	env := envelopeAt(t)
	grown, _, err := base.Append(env, domain.KindObservation, claimFor(t, "PETR4"))
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(ctx, grown); err != nil {
		t.Fatal(err)
	}

	if err := store.Save(ctx, base); err == nil {
		t.Fatal("a shorter stream was accepted; history was rewritten")
	}
}

// The ordinary path must keep working: an append extends the stream by one and
// is accepted, repeatedly.
func TestSaveAcceptsAnAppend(t *testing.T) {
	ctx := context.Background()
	store := memory.NewStore()

	stream, err := domain.NewStream("acct-1")
	if err != nil {
		t.Fatal(err)
	}
	env := envelopeAt(t)

	for _, ticker := range []string{"PETR4", "VALE3", "ITUB4"} {
		stream, _, err = stream.Append(env, domain.KindObservation, claimFor(t, ticker))
		if err != nil {
			t.Fatal(err)
		}
		if saveErr := store.Save(ctx, stream); saveErr != nil {
			t.Fatalf("appending %s was refused: %v", ticker, saveErr)
		}
	}

	loaded, err := store.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Len() != 3 {
		t.Errorf("stream length is %d, want 3", loaded.Len())
	}
}

// A stream nobody has written is absent, not empty. "We know nothing" and "we
// hold nothing" are different answers (ADR-0022).
func TestLoadingAnUnknownStreamSaysSo(t *testing.T) {
	if _, err := memory.NewStore().Load(context.Background(), "nope"); err == nil {
		t.Fatal("an unknown stream loaded without error")
	} else if !strings.Contains(err.Error(), app.ErrStreamNotFound.Error()) {
		t.Errorf("want ErrStreamNotFound, got %v", err)
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

func envelopeAt(t *testing.T) domain.Envelope {
	t.Helper()
	at := temporal.MustAt(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC))
	interval, err := temporal.OpenFrom(at)
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := temporal.Assign(interval, at)
	if err != nil {
		t.Fatal(err)
	}
	source, err := provenance.NewSource(
		"sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef")
	if err != nil {
		t.Fatal(err)
	}
	prov, err := provenance.Observed(source, at, provenance.Unmediated(), provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.NewEnvelope(coordinates, prov, nil)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}
