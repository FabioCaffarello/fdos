package domain_test

import (
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// The property ADR-0022 rests on: **replay does not re-resolve, it reads**.
//
// A 2031 replay of a 2026 artifact must find the identity minted in 2026. The
// discriminating case is a stream that contains a *second* mint for the same
// claim — the defect ADR-0022 says the ledger records rather than hides. A
// replay must still read the first one, or every artifact regenerated after the
// duplicate arrived would silently change what it referred to.
func TestALaterMintDoesNotShadowTheEarlierOne(t *testing.T) {
	claim := identity.MustClaim("ticker", "PETR4")

	first, err := domain.MintFor(identity.KindInstrument, claim)
	if err != nil {
		t.Fatal(err)
	}
	// A resolver bug, a replayed acquisition, a merge that should have been an
	// EntitiesIdentified: however it happened, the ledger keeps it.
	duplicate := domain.EntityMinted{
		Entity:   identity.MustDerive(identity.KindInstrument, "a different seed entirely"),
		BornFrom: claim,
	}
	if duplicate.Entity.Equal(first.Entity) {
		t.Fatal("fixture is not discriminating: both mints have the same identity")
	}

	stream := streamWith(t,
		entry{envelope: envelopeAt(t, day(1), knowledge(1)), payload: first},
		entry{envelope: envelopeAt(t, day(1), knowledge(2)), payload: duplicate},
	)

	early, err := domain.Resolve(stream, claim, asOf(t, day(2), knowledge(2)))
	if err != nil {
		t.Fatalf("resolve early: %v", err)
	}
	// Five years later, by knowledge time.
	late, err := domain.Resolve(stream, claim, asOf(t, day(2), knowledge(50_000)))
	if err != nil {
		t.Fatalf("resolve late: %v", err)
	}

	if !early.Equal(late) {
		t.Fatalf("replay resolved to a different identity: %s then %s", early, late)
	}
	if !early.Equal(first.Entity) {
		t.Fatalf("resolution took the later mint: got %s, want %s", early, first.Entity)
	}
}

// A mint is only visible after FDOS learned of it. Resolving before that must
// fail rather than reach forward — the same look-ahead guarantee the ledger
// already has, applied to identity.
func TestAMintIsInvisibleBeforeItWasKnown(t *testing.T) {
	claim := identity.MustClaim("ticker", "PETR4")
	stream := streamWith(t, mintFact(t, day(1), knowledge(100), claim))

	if _, err := domain.Resolve(stream, claim, asOf(t, day(2), knowledge(50))); !errors.Is(err, domain.ErrUnresolved) {
		t.Fatalf("a mint resolved before it was known: %v", err)
	}
}

// An unresolved claim is an error, not a silently minted identity. Whether to
// mint is a decision needing a clock and a source, which the pure domain does
// not have.
func TestAnUnknownClaimDoesNotResolve(t *testing.T) {
	stream := streamWith(t, mintFact(t, day(1), knowledge(1), identity.MustClaim("ticker", "PETR4")))

	_, err := domain.Resolve(stream, identity.MustClaim("ticker", "VALE3"), asOf(t, day(2), knowledge(2)))
	if !errors.Is(err, domain.ErrUnresolved) {
		t.Fatalf("expected ErrUnresolved, got %v", err)
	}
}

// Claims differing only by whitespace are different claims. Deciding they are
// the same thing is resolution, and resolution is a recorded fact.
func TestClaimMatchingIsExact(t *testing.T) {
	stream := streamWith(t, mintFact(t, day(1), knowledge(1), identity.MustClaim("ticker", "PETR4")))

	if _, err := domain.Resolve(stream, identity.MustClaim("ticker", "PETR4 "), asOf(t, day(2), knowledge(2))); err == nil {
		t.Fatal("a whitespace-differing claim resolved; that is a resolution decision, not an equality one")
	}
}

// Minting is deterministic: the same claim under the same kind yields the same
// identifier, so replaying an acquisition produces the same mint (§9).
func TestMintingIsDeterministicAndSeparatesKinds(t *testing.T) {
	claim := identity.MustClaim("ticker", "PETR4")

	first, err := domain.MintFor(identity.KindInstrument, claim)
	if err != nil {
		t.Fatal(err)
	}
	second, err := domain.MintFor(identity.KindInstrument, claim)
	if err != nil {
		t.Fatal(err)
	}
	if !first.Entity.Equal(second.Entity) {
		t.Fatalf("minting is not deterministic: %s vs %s", first.Entity, second.Entity)
	}

	asAccount, err := domain.MintFor(identity.KindAccount, claim)
	if err != nil {
		t.Fatal(err)
	}
	if asAccount.Entity.Equal(first.Entity) {
		t.Error("the same claim under different kinds produced the same identity")
	}

	if !first.BornFrom.Equal(claim) {
		t.Error("the mint does not record the claim it was born from")
	}
}

// The whole point of ADR-0022: a claimed holding becomes an observed one only
// through a recorded derivation naming what it consumed.
func TestDerivingAnObservationNamesTheClaimAndBothMints(t *testing.T) {
	account := identity.MustClaim("account_number", "0001234-5")
	instrument := identity.MustClaim("ticker", "PETR4")

	stream := streamWith(t,
		mintFact(t, day(1), knowledge(1), account),
		mintFact(t, day(1), knowledge(2), instrument),
		claimedFact(t, day(1), knowledge(3), account, instrument, "150"),
	)

	claimed, err := stream.Get(domain.Ref{Stream: "acct-1", Sequence: 3})
	if err != nil {
		t.Fatal(err)
	}

	result, err := domain.DeriveHoldingObserved(stream, claimed, asOf(t, day(2), knowledge(10)))
	if err != nil {
		t.Fatalf("derive: %v", err)
	}

	if result.Value().Quantity.String() != "150" {
		t.Errorf("quantity lost: %s", result.Value().Quantity)
	}
	if result.Trace().IsZero() {
		t.Fatal("derivation produced no trace")
	}
	// Named, not counted: three inputs that were all the same ref would satisfy
	// a count and explain nothing.
	wantInputs := []string{"acct-1#3", "acct-1#1", "acct-1#2"}
	if got := result.Record().Inputs(); !slices.Equal(got, wantInputs) {
		t.Errorf("inputs are %v, want the claim and both mints %v", got, wantInputs)
	}
	if !result.Value().Account.Equal(identity.MustDerive(identity.KindAccount, account.String())) {
		t.Error("account resolved to the wrong identity")
	}

	// The parameters are load-bearing, not decoration: without as_of, a
	// derivation at a coordinate that selected a different mint would share this
	// one's content address.
	params := map[string]string{}
	for _, p := range result.Record().Parameters() {
		params[p.Name] = p.Value
	}
	for _, want := range []string{"account_claim", "instrument_claim", "as_of"} {
		if params[want] == "" {
			t.Errorf("derivation does not record %q", want)
		}
	}
	if params["account_claim"] != account.String() {
		t.Errorf("account claim recorded as %q, want %q", params["account_claim"], account)
	}
}

// The consequence of recording as_of: the same claim resolved at two
// coordinates is two derivations, and they must not collide even when the
// answer happens to match.
func TestTheDerivationAddressDependsOnTheCoordinate(t *testing.T) {
	account := identity.MustClaim("account_number", "0001234-5")
	instrument := identity.MustClaim("ticker", "PETR4")

	stream := streamWith(t,
		mintFact(t, day(1), knowledge(1), account),
		mintFact(t, day(1), knowledge(2), instrument),
		claimedFact(t, day(1), knowledge(3), account, instrument, "150"),
	)
	claimed, err := stream.Get(domain.Ref{Stream: "acct-1", Sequence: 3})
	if err != nil {
		t.Fatal(err)
	}

	early, err := domain.DeriveHoldingObserved(stream, claimed, asOf(t, day(2), knowledge(10)))
	if err != nil {
		t.Fatal(err)
	}
	late, err := domain.DeriveHoldingObserved(stream, claimed, asOf(t, day(2), knowledge(50_000)))
	if err != nil {
		t.Fatal(err)
	}

	if early.Value().Quantity.String() != late.Value().Quantity.String() {
		t.Fatal("fixture is not discriminating: the two derivations disagree on the answer")
	}
	if early.Trace().Hash() == late.Trace().Hash() {
		t.Fatal("two derivations at different coordinates share a content address")
	}
}

// A claim that never resolves derives nothing. The observation is absent, not
// approximated — "we know nothing", distinct from "we hold nothing" (ADR-0022).
func TestAClaimWithNoMintDerivesNothing(t *testing.T) {
	account := identity.MustClaim("account_number", "0001234-5")
	instrument := identity.MustClaim("ticker", "PETR4")

	// Only the account was minted.
	stream := streamWith(t,
		mintFact(t, day(1), knowledge(1), account),
		claimedFact(t, day(1), knowledge(2), account, instrument, "150"),
	)
	claimed, err := stream.Get(domain.Ref{Stream: "acct-1", Sequence: 2})
	if err != nil {
		t.Fatal(err)
	}

	if _, err := domain.DeriveHoldingObserved(stream, claimed, asOf(t, day(2), knowledge(10))); !errors.Is(err, domain.ErrUnresolved) {
		t.Fatalf("expected ErrUnresolved, got %v", err)
	}
}

// --- fixture -----------------------------------------------------------------

func streamWith(t *testing.T, payloads ...struct {
	envelope domain.Envelope
	payload  domain.Payload
},
) domain.Stream {
	t.Helper()
	stream, err := domain.NewStream("acct-1")
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range payloads {
		stream, _, err = stream.Append(p.envelope, domain.KindObservation, p.payload)
		if err != nil {
			t.Fatal(err)
		}
	}
	return stream
}

type entry = struct {
	envelope domain.Envelope
	payload  domain.Payload
}

func mintFact(t *testing.T, effective, known temporal.Instant, claim identity.Claim) entry {
	t.Helper()
	kind := identity.KindInstrument
	if claim.Scheme() == "account_number" {
		kind = identity.KindAccount
	}
	minted, err := domain.MintFor(kind, claim)
	if err != nil {
		t.Fatal(err)
	}
	return entry{envelope: envelopeAt(t, effective, known), payload: minted}
}

func claimedFact(t *testing.T, effective, known temporal.Instant, account, instrument identity.Claim, qty string) entry {
	t.Helper()
	return entry{
		envelope: envelopeAt(t, effective, known),
		payload: domain.HoldingClaimed{
			Account:    account,
			Instrument: instrument,
			Quantity:   money.MustParseQuantity(qty, "share"),
		},
	}
}

func envelopeAt(t *testing.T, effective, known temporal.Instant) domain.Envelope {
	t.Helper()
	interval, err := temporal.OpenFrom(effective)
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := temporal.Assign(interval, known)
	if err != nil {
		t.Fatal(err)
	}
	source, err := provenance.NewSource("broker.statement")
	if err != nil {
		t.Fatal(err)
	}
	interpreter, err := provenance.NewInterpreter("statement-parser", "1.0.0")
	if err != nil {
		t.Fatal(err)
	}
	prov, err := provenance.Observed(source, effective, interpreter, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.NewEnvelope(coordinates, prov, nil)
	if err != nil {
		t.Fatal(err)
	}
	return envelope
}

func asOf(t *testing.T, effective, known temporal.Instant) temporal.AsOf {
	t.Helper()
	a, err := temporal.NewAsOf(effective, known)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func day(n int) temporal.Instant {
	return temporal.MustAt(time.Date(2026, time.January, n, 0, 0, 0, 0, time.UTC))
}

func knowledge(hours int) temporal.Instant {
	return temporal.MustAt(time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC).
		Add(time.Duration(hours) * time.Hour))
}
