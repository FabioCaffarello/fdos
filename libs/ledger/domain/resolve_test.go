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

	first := mintPayload(t, identity.KindInstrument, claim)
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

	early, err := domain.Resolve(stream, claim, rules(), asOf(t, day(2), knowledge(2)))
	if err != nil {
		t.Fatalf("resolve early: %v", err)
	}
	// Five years later, by knowledge time.
	late, err := domain.Resolve(stream, claim, rules(), asOf(t, day(2), knowledge(50_000)))
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

	if _, err := domain.Resolve(stream, claim, rules(), asOf(t, day(2), knowledge(50))); !errors.Is(err, domain.ErrUnresolved) {
		t.Fatalf("a mint resolved before it was known: %v", err)
	}
}

// An unresolved claim is an error, not a silently minted identity. Whether to
// mint is a decision needing a clock and a source, which the pure domain does
// not have.
func TestAnUnknownClaimDoesNotResolve(t *testing.T) {
	stream := streamWith(t, mintFact(t, day(1), knowledge(1), identity.MustClaim("ticker", "PETR4")))

	_, err := domain.Resolve(stream, identity.MustClaim("ticker", "VALE3"), rules(), asOf(t, day(2), knowledge(2)))
	if !errors.Is(err, domain.ErrUnresolved) {
		t.Fatalf("expected ErrUnresolved, got %v", err)
	}
}

// Claims differing only by whitespace are different claims. Deciding they are
// the same thing is resolution, and resolution is a recorded fact.
//
// ADR-0033 replaced this test. It is not a reversal of what it protected:
// `Claim.Equal` is still byte equality, and deciding two claims are the same
// thing is still resolution's job rather than an equality operator's. What
// changed is that resolution now has a stated, versioned rule to decide it
// with, instead of a byte comparison standing in for one.
//
// The trailing-space case is the one that matters for the record: it already
// derived the *same* identity before this change, so refusing to resolve it
// meant minting a second fact for an identity that already existed.
func TestResolutionDecidesSamenessByRuleRatherThanByBytes(t *testing.T) {
	stream := streamWith(t, mintFact(t, day(1), knowledge(1), identity.MustClaim("ticker", "PETR4")))
	at := asOf(t, day(2), knowledge(2))

	// Folded by the generic floor, which Derive has always applied.
	for _, variant := range []string{"PETR4 ", "petr4"} {
		if _, err := domain.Resolve(stream, identity.MustClaim("ticker", variant), rules(), at); err != nil {
			t.Errorf("%q did not resolve, though minting it derives the same identity: %v", variant, err)
		}
	}

	// Folded by nothing: `ticker` names no issuing standard, so it carries no
	// rule. These are the cases where a wrong rule would merge two real
	// instruments, and they must stay apart.
	//
	// `" PETR4"` sits here rather than above, and the asymmetry is real rather
	// than an oversight: the seed is `scheme:value`, so a *leading* space in the
	// value becomes an internal separator that the floor collapses but cannot
	// remove, while a *trailing* one is stripped. PR #51 measured exactly this.
	// Whether the two should behave alike is a `ticker` rule question, and
	// `ticker` cannot have one.
	for _, variant := range []string{" PETR4", "PETR4.SA", "PETR 4", "PETR4."} {
		_, err := domain.Resolve(stream, identity.MustClaim("ticker", variant), rules(), at)
		if !errors.Is(err, domain.ErrUnresolved) {
			t.Errorf("%q resolved to the PETR4 mint; that is a merge, and merges are recorded", variant)
		}
	}
}

// The invariant ADR-0033 rests on, asserted rather than argued.
//
// Without it the design is not merely inelegant: a claim that mints to an
// existing identity but does not resolve to it is a claim MintIdentity refuses
// and Resolve cannot answer.
func TestResolutionAgreesWithMinting(t *testing.T) {
	base := identity.MustClaim("isin", "BRPETRACNPR0")
	stream := streamWith(t, mintFact(t, day(1), knowledge(1), base))
	at := asOf(t, day(2), knowledge(2))
	baseID := mintPayload(t, identity.KindInstrument, base).Entity

	// Spread across both folds and neither: vendor spacing on a standard
	// scheme, generic whitespace and case, and values that are simply other
	// instruments.
	for _, value := range []string{
		"BRPETRACNPR0", "BR PETR ACNPR0", "brpetracnpr0", "BRPETRACNPR0 ",
		"BRPETRACNPR1", "BRPETRACNPR", "US0378331005",
	} {
		claim := identity.MustClaim("isin", value)

		resolved, err := domain.Resolve(stream, claim, rules(), at)
		resolvesToBase := err == nil && resolved.Equal(baseID)
		mintsToBase := mintPayload(t, identity.KindInstrument, claim).Entity.Equal(baseID)

		if resolvesToBase != mintsToBase {
			t.Errorf("%q resolves-to-base=%v but mints-to-base=%v", value, resolvesToBase, mintsToBase)
		}
	}
}

// A mint records what the provider said, not what FDOS made of it. The folded
// value is an input to the derivation; the raw value is the evidence, and a
// resolution later found wrong has to be re-doable from it.
func TestAMintRecordsTheClaimVerbatim(t *testing.T) {
	spaced := identity.MustClaim("isin", "BR PETR ACNPR0")
	minted := mintPayload(t, identity.KindInstrument, spaced)

	if minted.BornFrom.Value() != "BR PETR ACNPR0" {
		t.Errorf("BornFrom was canonicalised to %q; it is the birth certificate", minted.BornFrom.Value())
	}
	clean := mintPayload(t, identity.KindInstrument, identity.MustClaim("isin", "BRPETRACNPR0"))
	if !minted.Entity.Equal(clean.Entity) {
		t.Error("two renderings of one ISIN minted two identities")
	}
}

// The mint's own derivation, without which its provenance cannot be Derived.
func TestAMintExplainsItself(t *testing.T) {
	claim := identity.MustClaim("isin", "BR PETR ACNPR0")

	minted, err := domain.MintFor(
		identity.KindInstrument, claim, rules(), []string{"acct-1#7"}, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	if minted.Trace().IsZero() {
		t.Fatal("a mint produced no trace, so its provenance cannot be Derived")
	}
	if got := minted.Record().Inputs(); !slices.Equal(got, []string{"acct-1#7"}) {
		t.Errorf("the mint does not name the claim fact it answers: %v", got)
	}

	params := map[string]string{}
	for _, p := range minted.Record().Parameters() {
		params[p.Name] = p.Value
	}
	if params["claim"] != claim.String() {
		t.Errorf("claim recorded as %q", params["claim"])
	}
	if params["seed"] != "ISIN:BRPETRACNPR0" {
		t.Errorf("seed recorded as %q, want the canonicalised form", params["seed"])
	}
	// Which rules minted this is what makes a later rule change nameable.
	if params["canonicalisation"] != rules().Version() {
		t.Errorf("ruleset version recorded as %q, want %q", params["canonicalisation"], rules().Version())
	}

	// An account minted from operator configuration answers no fact, and that
	// must be representable — RFC-0007's case, and it has no claim to name.
	ahead, err := domain.MintFor(
		identity.KindAccount, identity.MustClaim("account_number", "0001234-5"),
		rules(), nil, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatalf("minting ahead of any observation failed: %v", err)
	}
	if len(ahead.Record().Inputs()) != 0 {
		t.Errorf("a mint with no claim fact named inputs: %v", ahead.Record().Inputs())
	}
}

// Minting is deterministic: the same claim under the same kind yields the same
// identifier, so replaying an acquisition produces the same mint (§9).
func TestMintingIsDeterministicAndSeparatesKinds(t *testing.T) {
	claim := identity.MustClaim("ticker", "PETR4")

	first := mintPayload(t, identity.KindInstrument, claim)
	second := mintPayload(t, identity.KindInstrument, claim)
	if !first.Entity.Equal(second.Entity) {
		t.Fatalf("minting is not deterministic: %s vs %s", first.Entity, second.Entity)
	}

	asAccount := mintPayload(t, identity.KindAccount, claim)
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

	result, err := domain.DeriveHoldingObserved(stream, claimed, rules(), asOf(t, day(2), knowledge(10)))
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
	// Derived the way minting derives — through the folded claim, not through a
	// flattened `scheme:value` (ADR-0040). Reproducing the route rather than
	// pinning a literal is what makes this assert that resolution found the mint,
	// instead of asserting which bytes the kernel happens to hash.
	wantAccount, err := identity.DeriveFromClaim(identity.KindAccount, rules().Fold(account))
	if err != nil {
		t.Fatal(err)
	}
	if !result.Value().Account.Equal(wantAccount) {
		t.Error("account resolved to the wrong identity")
	}

	// The parameters are load-bearing, not decoration: without as_of, a
	// derivation at a coordinate that selected a different mint would share this
	// one's content address.
	params := map[string]string{}
	for _, p := range result.Record().Parameters() {
		params[p.Name] = p.Value
	}
	for _, want := range []string{"account_claim", "instrument_claim", "as_of", "canonicalisation"} {
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

	early, err := domain.DeriveHoldingObserved(stream, claimed, rules(), asOf(t, day(2), knowledge(10)))
	if err != nil {
		t.Fatal(err)
	}
	late, err := domain.DeriveHoldingObserved(stream, claimed, rules(), asOf(t, day(2), knowledge(50_000)))
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

	if _, err := domain.DeriveHoldingObserved(stream, claimed, rules(), asOf(t, day(2), knowledge(10))); !errors.Is(err, domain.ErrUnresolved) {
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
	return entry{envelope: envelopeAt(t, effective, known), payload: mintPayload(t, kind, claim)}
}

// rules is the ruleset every test here resolves under: the one this build
// ships (ADR-0033).
func rules() identity.Ruleset { return identity.Canonicalisation() }

func mintPayload(t *testing.T, kind identity.Kind, claim identity.Claim) domain.EntityMinted {
	t.Helper()
	minted, err := domain.MintFor(kind, claim, rules(), nil, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	return minted.Value()
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

// The collision closed end to end, which is what neither half could do alone.
//
// Measured on 2026-08-10: `claim("ticker", "x:y")` and `claim("ticker:x", "y")`
// flattened to one `scheme:value` rendering, so minting derived one identifier
// for both and resolution answered either claim with the other's identity. A
// silent merge, in an append-only ledger, which ADR-0007 forbids and records as
// EntitiesIdentified instead.
//
// Both assertions matter and they are not redundant. Minting apart while
// resolving together would answer the wrong claim; resolving apart while minting
// together would mint a second identity for an entity that already has one. The
// invariant ADR-0033 states — two claims resolve to the same identity if and only
// if minting them derives the same identity — is what ties them, and ADR-0040 is
// why both sides had to move in one commit.
func TestAMovedClaimBoundaryNeitherMintsNorResolvesTogether(t *testing.T) {
	left := identity.MustClaim("ticker", "x:y")
	right := identity.MustClaim("ticker:x", "y")

	leftMint, err := domain.MintFor(
		identity.KindInstrument, left, rules(), []string{"acct-1#1"}, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatalf("mint %s: %v", left, err)
	}
	rightMint, err := domain.MintFor(
		identity.KindInstrument, right, rules(), []string{"acct-1#2"}, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatalf("mint %s: %v", right, err)
	}

	if leftMint.Value().Entity.Equal(rightMint.Value().Entity) {
		t.Fatalf("%s and %s minted one identity (%s) — two claims, one entity, silently",
			left, right, leftMint.Value().Entity)
	}

	// And resolution agrees: the stream holds only the left mint, so the right
	// claim must find nothing rather than find the left entity.
	stream := streamWith(t,
		entry{envelope: envelopeAt(t, day(1), knowledge(1)), payload: leftMint.Value()},
	)

	resolved, err := domain.Resolve(stream, left, rules(), asOf(t, day(2), knowledge(2)))
	if err != nil {
		t.Fatalf("the minted claim did not resolve to its own mint: %v", err)
	}
	if !resolved.Equal(leftMint.Value().Entity) {
		t.Errorf("%s resolved to %s, want its own mint %s", left, resolved, leftMint.Value().Entity)
	}

	if _, err := domain.Resolve(stream, right, rules(), asOf(t, day(2), knowledge(2))); !errors.Is(err, domain.ErrUnresolved) {
		t.Errorf("%s resolved against a stream that only minted %s (err=%v); that is a merge, "+
			"and merges are recorded as facts", right, left, err)
	}
}
