package app_test

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
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/memory"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
)

const validDigest = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func acceptFixture(t *testing.T, source string) (*app.Ledger, app.AcceptHoldingClaimCommand) {
	t.Helper()

	at := temporal.MustAt(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC))
	ledger, err := app.NewLedger(memory.NewStore(), clock.NewSequence(at, time.Hour), identity.Canonicalisation())
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}

	src, err := provenance.NewSource(source)
	if err != nil {
		t.Fatalf("source: %v", err)
	}
	quantity, err := money.ParseQuantity("100", "share")
	if err != nil {
		t.Fatalf("quantity: %v", err)
	}
	effective, err := temporal.OpenFrom(at)
	if err != nil {
		t.Fatalf("interval: %v", err)
	}

	return ledger, app.AcceptHoldingClaimCommand{
		Stream:      "acct-1",
		Account:     identity.MustClaim("account", "A-1"),
		Instrument:  identity.MustClaim("ticker", "PETR4"),
		Quantity:    quantity,
		Effective:   effective,
		Source:      src,
		CollectedAt: at,
		Interpreter: provenance.Unmediated(),
		Confidence:  provenance.ConfidenceAsserted,
	}
}

// The point of the whole entry point: a producer that cannot mint an identity
// can still put a fact in the ledger (ADR-0029).
func TestAProducerWithoutAnIdentityCanSubmit(t *testing.T) {
	ledger, cmd := acceptFixture(t, validDigest)

	ref, err := ledger.AcceptHoldingClaim(context.Background(), cmd)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}
	if ref.Stream != "acct-1" {
		t.Fatalf("ref: got %q", ref.Stream)
	}
}

// Admission is where the grammar ADR-0028 specified is finally enforced. It
// shipped at rung 6 because there was no admission point; this is the point.
//
// Each of these is a value someone would plausibly put in a field named
// `value`, which is the mistake the grammar exists to refuse.
func TestAdmissionRefusesAMalformedSource(t *testing.T) {
	for _, tc := range []struct{ name, source string }{
		{"a url", "https://broker.example/statement.pdf"},
		{"an account id", "account-88213"},
		{"a bare digest with no algorithm", strings.Repeat("a", 64)},
		{"a truncated digest", "sha256:abc123"},
		{"uppercase hex", "sha256:" + strings.Repeat("A", 64)},
		{"a plausible non-hex", "sha256:" + strings.Repeat("z", 64)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ledger, cmd := acceptFixture(t, tc.source)

			_, err := ledger.AcceptHoldingClaim(context.Background(), cmd)
			if !errors.Is(err, provenance.ErrMalformedSource) {
				t.Fatalf("want ErrMalformedSource, got %v", err)
			}
		})
	}
}

// Admission assumes nothing about what the caller ran. A producer may link a
// modified build of any helper FDOS publishes, so every rule is checked here
// regardless (ADR-0029).
func TestAdmissionRefusesAClaimWithNoIdentifier(t *testing.T) {
	ledger, cmd := acceptFixture(t, validDigest)
	cmd.Instrument = identity.Claim{}

	if _, err := ledger.AcceptHoldingClaim(context.Background(), cmd); err == nil {
		t.Fatal("want a rejection for a claim naming no instrument")
	}
}

// The load-bearing negative: admission must not mint.
//
// An identity that came into existence because a stranger submitted a claim is
// an identity nobody chose, and once a producer depends on that behaviour
// removing it is a change to what the ledger does (ADR-0007, ADR-0022).
func TestAdmissionMintsNothing(t *testing.T) {
	ledger, cmd := acceptFixture(t, validDigest)
	ctx := context.Background()

	if _, err := ledger.AcceptHoldingClaim(ctx, cmd); err != nil {
		t.Fatalf("accept: %v", err)
	}

	unresolved, err := ledger.UnresolvedClaims(ctx, app.UnresolvedClaimsQuery{
		Stream: "acct-1",
		AsOf:   lateAsOf(t),
	})
	if err != nil {
		t.Fatalf("unresolved: %v", err)
	}

	// Both identifiers must still be waiting. If either resolved, admission
	// minted something.
	if len(unresolved) != 2 {
		t.Fatalf("want both claims unresolved after admission, got %d", len(unresolved))
	}
}

// A connector can publish faithfully into silence. This is what breaks the
// silence, and it must not break it by resolving anything.
func TestLookingDoesNotResolve(t *testing.T) {
	ledger, cmd := acceptFixture(t, validDigest)
	ctx := context.Background()
	asOf := lateAsOf(t)

	if _, err := ledger.AcceptHoldingClaim(ctx, cmd); err != nil {
		t.Fatalf("accept: %v", err)
	}

	first, err := ledger.UnresolvedClaims(ctx, app.UnresolvedClaimsQuery{Stream: "acct-1", AsOf: asOf})
	if err != nil {
		t.Fatalf("unresolved: %v", err)
	}
	second, err := ledger.UnresolvedClaims(ctx, app.UnresolvedClaimsQuery{Stream: "acct-1", AsOf: asOf})
	if err != nil {
		t.Fatalf("unresolved again: %v", err)
	}

	if len(first) != len(second) {
		t.Fatalf("asking changed the answer: %d then %d", len(first), len(second))
	}
	for i := range first {
		if !first[i].Claim.Equal(second[i].Claim) || first[i].Fact != second[i].Fact {
			t.Fatal("asking twice gave different answers")
		}
	}
}

// The claim must survive admission verbatim. A value silently canonicalised at
// the door is a resolution decision taken by a parser, which ADR-0007 puts on
// the other side of the line.
func TestTheClaimIsStoredAsSubmitted(t *testing.T) {
	ledger, cmd := acceptFixture(t, validDigest)
	cmd.Instrument = identity.MustClaim("ticker", "PETR4 ")
	ctx := context.Background()

	ref, err := ledger.AcceptHoldingClaim(ctx, cmd)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}

	unresolved, err := ledger.UnresolvedClaims(ctx, app.UnresolvedClaimsQuery{
		Stream: "acct-1",
		AsOf:   lateAsOf(t),
	})
	if err != nil {
		t.Fatalf("unresolved: %v", err)
	}

	var found bool
	for _, u := range unresolved {
		if u.Fact == ref && u.Claim.Value() == "PETR4 " {
			found = true
		}
	}
	if !found {
		t.Fatal("the trailing space was altered somewhere between submission and storage")
	}
}

// lateAsOf is a coordinate after everything these tests append, on both axes.
func lateAsOf(t *testing.T) temporal.AsOf {
	t.Helper()
	late := temporal.MustAt(time.Date(2027, time.January, 1, 0, 0, 0, 0, time.UTC))
	asOf, err := temporal.NewAsOf(late, late)
	if err != nil {
		t.Fatalf("as-of: %v", err)
	}
	return asOf
}
