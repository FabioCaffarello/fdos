package app_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/clock"
	"github.com/FabioCaffarello/fdos/libs/ledger/adapters/memory"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// The whole point of M9 Track A: a claim can finally become an observation.
//
// Before this, `Resolve`, `MintFor` and `DeriveHoldingObserved` all existed and
// nothing called them, so FDOS could receive financial data and do nothing with
// it. This walks the path end to end — admit, mint, derive — and it is the test
// that would go red if any link in it were removed.
func TestAClaimBecomesAnObservation(t *testing.T) {
	ledger, cmd := acceptFixture(t, validDigest)
	ctx := context.Background()

	claimRef, err := ledger.AcceptHoldingClaim(ctx, cmd)
	if err != nil {
		t.Fatalf("accept: %v", err)
	}

	// Nothing resolves yet: admission mints nothing, by design.
	waiting, err := ledger.UnresolvedClaims(ctx, app.UnresolvedClaimsQuery{Stream: "acct-1", AsOf: lateAsOf(t)})
	if err != nil {
		t.Fatalf("unresolved: %v", err)
	}
	if len(waiting) != 2 {
		t.Fatalf("want two claims waiting, got %d", len(waiting))
	}

	// An owner acts on each, naming the claim fact it answers.
	for _, m := range []struct {
		kind  identity.Kind
		claim identity.Claim
	}{
		{identity.KindAccount, cmd.Account},
		{identity.KindInstrument, cmd.Instrument},
	} {
		if _, mintErr := ledger.MintIdentity(ctx, mintCommand(t, m.kind, m.claim, claimRef)); mintErr != nil {
			t.Fatalf("mint %s: %v", m.claim, mintErr)
		}
	}

	remaining, err := ledger.UnresolvedClaims(ctx, app.UnresolvedClaimsQuery{
		Stream: "acct-1", AsOf: lateAsOf(t),
	})
	if err != nil || len(remaining) != 0 {
		t.Fatalf("claims still waiting after minting: %v (%v)", remaining, err)
	}

	observedRef, err := ledger.ObserveClaimedHolding(ctx, app.ObserveClaimedHoldingCommand{
		Stream:      "acct-1",
		Claim:       claimRef,
		AsOf:        lateAsOf(t),
		Source:      mustSource(validDigest),
		CollectedAt: fixtureAt(),
		Interpreter: provenance.Unmediated(),
	})
	if err != nil {
		t.Fatalf("observe: %v", err)
	}
	if observedRef == claimRef {
		t.Fatal("the observation did not become its own fact")
	}
}

// The load-bearing refusal (ADR-0033).
//
// Returning the existing identity as though a mint had happened is how a caller
// learns to stop checking, so this must be an error a caller has to handle.
func TestMintingTwiceIsRefused(t *testing.T) {
	ledger, ctx := mintOnlyFixture(t)
	claim := identity.MustClaim("isin", "BRPETRACNPR0")

	if _, err := ledger.MintIdentity(ctx, mintCommand(t, identity.KindInstrument, claim, domain.Ref{})); err != nil {
		t.Fatalf("first mint: %v", err)
	}

	_, err := ledger.MintIdentity(ctx, mintCommand(t, identity.KindInstrument, claim, domain.Ref{}))
	if !errors.Is(err, app.ErrAlreadyMinted) {
		t.Fatalf("want ErrAlreadyMinted, got %v", err)
	}
}

// The consequence of resolution folding: a vendor's spaced rendering is refused
// as a duplicate rather than recorded as a second mint for an identity that
// already exists.
//
// This is the behaviour the earlier design could not have. Byte-exact matching
// would let this through, appending a second EntityMinted fact carrying the
// same identity.
func TestAVendorRenderingOfAMintedIdentifierIsRefused(t *testing.T) {
	ledger, ctx := mintOnlyFixture(t)

	clean := identity.MustClaim("isin", "BRPETRACNPR0")
	if _, err := ledger.MintIdentity(ctx, mintCommand(t, identity.KindInstrument, clean, domain.Ref{})); err != nil {
		t.Fatalf("first mint: %v", err)
	}

	spaced := identity.MustClaim("isin", "BR PETR ACNPR0")
	if _, err := ledger.MintIdentity(ctx, mintCommand(t, identity.KindInstrument, spaced, domain.Ref{})); !errors.Is(err, app.ErrAlreadyMinted) {
		t.Fatalf("a spaced ISIN minted a second fact for one identity: %v", err)
	}

	// And the reverse of the same coin: a ticker and its venue-suffixed form are
	// two instruments until somebody records otherwise, so this must succeed.
	if _, err := ledger.MintIdentity(ctx, mintCommand(
		t, identity.KindInstrument, identity.MustClaim("ticker", "PETR4"), domain.Ref{})); err != nil {
		t.Fatalf("mint PETR4: %v", err)
	}
	if _, err := ledger.MintIdentity(ctx, mintCommand(
		t, identity.KindInstrument, identity.MustClaim("ticker", "PETR4.SA"), domain.Ref{})); err != nil {
		t.Fatalf("PETR4.SA was refused; a venue suffix is not a merge FDOS may perform: %v", err)
	}
}

// Minting names an authority, and a mint is where a malformed one must be
// caught — the same discipline admission applies, for the same reason.
//
// What is *not* checked, and cannot be, is whether that authority was entitled
// to mint. ADR-0033 records that at rung 6; this test only asserts the part
// that is mechanised.
func TestMintingRefusesAMalformedAuthority(t *testing.T) {
	ledger, ctx := mintOnlyFixture(t)

	cmd := mintCommand(t, identity.KindInstrument, identity.MustClaim("ticker", "PETR4"), domain.Ref{})
	bad, err := provenance.NewSource("operator-console")
	if err != nil {
		t.Fatal(err)
	}
	cmd.Source = bad

	if _, err := ledger.MintIdentity(ctx, cmd); !errors.Is(err, provenance.ErrMalformedSource) {
		t.Fatalf("want ErrMalformedSource, got %v", err)
	}
}

// A mint with no kind would derive nothing, and a mint with no claim would have
// no birth certificate.
func TestMintingRefusesAnIncompleteCommand(t *testing.T) {
	ledger, ctx := mintOnlyFixture(t)

	noKind := mintCommand(t, identity.KindUnspecified, identity.MustClaim("ticker", "PETR4"), domain.Ref{})
	if _, err := ledger.MintIdentity(ctx, noKind); !errors.Is(err, identity.ErrUnknownKind) {
		t.Errorf("want ErrUnknownKind, got %v", err)
	}

	noClaim := mintCommand(t, identity.KindInstrument, identity.MustClaim("ticker", "PETR4"), domain.Ref{})
	noClaim.Claim = identity.Claim{}
	if _, err := ledger.MintIdentity(ctx, noClaim); err == nil {
		t.Error("a mint with no claim was accepted")
	}
}

// The ruleset determines which real-world things FDOS treats as one thing, so
// it is a decision to be made rather than defaulted into (ADR-0033).
func TestALedgerWithoutARulesetIsRefused(t *testing.T) {
	at := temporal.MustAt(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC))

	if _, err := app.NewLedger(memory.NewStore(), clock.NewSequence(at, time.Hour), identity.Ruleset{}); err == nil {
		t.Fatal("a ledger was built with no canonicalisation ruleset")
	}
}

// A mint is an occurrence: nobody told FDOS the identity exists, FDOS decided
// it and the decision happened. Recording it as an observation would assert a
// source that does not exist (ADR-0011).
//
// Its provenance is Derived for the same reason, and that is not decoration —
// `provenance.Derived` cannot be built without a derivation reference, so the
// mint has to be able to explain itself.
func TestAMintIsAnOccurrenceThatExplainsItself(t *testing.T) {
	ledger, store := mintOnlyFixtureWithStore(t)
	ctx := context.Background()
	claim := identity.MustClaim("isin", "BRPETRACNPR0")

	ref, err := ledger.MintIdentity(ctx, mintCommand(t, identity.KindInstrument, claim, domain.Ref{}))
	if err != nil {
		t.Fatalf("mint: %v", err)
	}

	fact := loadFact(t, ctx, store, ref)
	if fact.Kind() != domain.KindOccurrence {
		t.Errorf("a mint was recorded as %s", fact.Kind())
	}
	if _, ok := fact.Envelope().Provenance().Derivation(); !ok {
		t.Error("a mint carries Observed provenance; nobody observed an identity coming into existence")
	}
}

// --- fixture -----------------------------------------------------------------

func fixtureAt() temporal.Instant {
	return temporal.MustAt(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC))
}

func mintOnlyFixture(t *testing.T) (*app.Ledger, context.Context) {
	t.Helper()
	ledger, _ := mintOnlyFixtureWithStore(t)
	return ledger, context.Background()
}

func mintOnlyFixtureWithStore(t *testing.T) (*app.Ledger, app.Store) {
	t.Helper()
	store := memory.NewStore()
	ledger, err := app.NewLedger(store, clock.NewSequence(fixtureAt(), time.Hour), identity.Canonicalisation())
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	return ledger, store
}

func mintCommand(
	t *testing.T,
	kind identity.Kind,
	claim identity.Claim,
	answers domain.Ref,
) app.MintIdentityCommand {
	t.Helper()
	effective, err := temporal.OpenFrom(fixtureAt())
	if err != nil {
		t.Fatal(err)
	}
	return app.MintIdentityCommand{
		Stream:      "acct-1",
		Kind:        kind,
		Claim:       claim,
		Answers:     answers,
		Effective:   effective,
		Source:      mustSource(validDigest),
		CollectedAt: fixtureAt(),
		Interpreter: provenance.Unmediated(),
		Confidence:  provenance.ConfidenceAsserted,
	}
}

// loadFact reads one fact back through the store.
//
// Deliberately not through a query on Ledger: there is no use case for "give me
// the raw fact", and adding one so a test can look would put a method on the
// application surface that exists for the test's benefit.
func loadFact(t *testing.T, ctx context.Context, store app.Store, ref domain.Ref) domain.Fact {
	t.Helper()
	stream, err := store.Load(ctx, ref.Stream)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	fact, err := stream.Get(ref)
	if err != nil {
		t.Fatalf("get %s: %v", ref, err)
	}
	return fact
}

// The race ADR-0034 exists to close, and the reason `Expectation` is not
// decoration.
//
// `MintIdentity` resolves, finds nothing, and then appends. Between those two
// steps another writer can mint for the same claim. Appending anyway would
// produce two mints for one claim — the duplication ADR-0033 refuses, arriving
// as a race rather than as a caller error, and therefore invisible to the
// `ErrAlreadyMinted` check that runs before it.
//
// The store is wrapped so that exactly one competing append lands in that
// window. Without the precondition this test records a second mint; with it,
// the caller is told to resolve again.
func TestMintingRefusesWhenTheStreamMovedUnderIt(t *testing.T) {
	ctx := context.Background()
	inner := memory.NewStore()
	claim := identity.MustClaim("isin", "BRPETRACNPR0")

	racing := &racingStore{inner: inner, intrude: func(s *memory.Store) {
		// Another writer mints for the same claim, a minute before ours.
		minted, err := domain.MintFor(
			identity.KindInstrument, claim, identity.Canonicalisation(), nil, provenance.ConfidenceAsserted)
		if err != nil {
			t.Fatal(err)
		}
		env := mintEnvelope(t, fixtureAt().Time().Add(-time.Minute), minted.Trace())
		if _, err := s.Append(ctx, "acct-1", app.Any(), env, domain.KindOccurrence, minted.Value()); err != nil {
			t.Fatalf("the competing writer could not append: %v", err)
		}
	}}

	ledger, err := app.NewLedger(racing, clock.NewSequence(fixtureAt(), time.Hour), identity.Canonicalisation())
	if err != nil {
		t.Fatal(err)
	}

	_, err = ledger.MintIdentity(ctx, mintCommand(t, identity.KindInstrument, claim, domain.Ref{}))
	if !errors.Is(err, app.ErrStaleRead) {
		t.Fatalf("want ErrStaleRead, got %v", err)
	}

	stream, err := inner.Load(ctx, "acct-1")
	if err != nil {
		t.Fatal(err)
	}
	if stream.Len() != 1 {
		t.Errorf("stream holds %d facts; the refused mint was appended anyway", stream.Len())
	}
}

// racingStore lets exactly one competing write land between a caller's read and
// its append — the window a precondition exists to protect.
type racingStore struct {
	inner   *memory.Store
	intrude func(*memory.Store)
	done    bool
}

func (r *racingStore) Load(ctx context.Context, name string) (domain.Stream, error) {
	return r.inner.Load(ctx, name)
}

// Serialise deliberately serialises nothing.
//
// ADR-0041 moved the write region into the store, and a store that honoured it
// here would close the very window this test opens. That is the right outcome
// for a writer inside the region and the wrong one for this test, because the
// intruder stands in for a writer that is **not** in it — another process
// before ADR-0041, or a caller of a store that implements the port badly.
//
// `Expectation` is what catches that writer, and it has to keep working when
// the region does not. A test whose competing write were serialised away would
// prove the region works and say nothing about the precondition, which is what
// it exists to hold (ADR-0034).
func (r *racingStore) Serialise(
	ctx context.Context,
	_ string,
	fn func(context.Context, app.Store) error,
) error {
	return fn(ctx, r)
}

func (r *racingStore) Append(
	ctx context.Context,
	name string,
	expect app.Expectation,
	envelope domain.Envelope,
	kind domain.Kind,
	payload domain.Payload,
) (domain.Ref, error) {
	if !r.done {
		r.done = true
		r.intrude(r.inner)
	}
	return r.inner.Append(ctx, name, expect, envelope, kind, payload)
}

func mintEnvelope(t *testing.T, at time.Time, trace provenance.Ref) domain.Envelope {
	t.Helper()
	instant := temporal.MustAt(at)
	interval, err := temporal.OpenFrom(instant)
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := temporal.Assign(interval, instant)
	if err != nil {
		t.Fatal(err)
	}
	prov, err := provenance.Derived(
		mustSource(validDigest), instant, provenance.Unmediated(), trace, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.NewEnvelope(coordinates, prov, nil)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}
