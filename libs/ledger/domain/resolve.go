package domain

import (
	"errors"
	"fmt"

	"github.com/FabioCaffarello/fdos/libs/kernel/explained"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// ErrUnresolved is returned when no mint visible at a coordinate matches a
// claim.
//
// A value, not a panic, and not a silently-minted identity: whether to mint is
// a decision the application layer makes with a clock and a source, and the
// pure domain cannot make it (ADR-0022).
var ErrUnresolved = errors.New("ledger: claim does not resolve to a minted identity")

// observationMethod names the derivation that turns a claim into an
// observation. Versioned: changing how a claim becomes an observation is a new
// method version, pinned in the report so a regeneration years later uses the
// method of the time (ADR-0010).
//
// There is deliberately no `ledger.ResolveClaim` method beside it. Resolution
// reads a recorded mint; a read derives nothing and has nothing to explain.
// Giving it a method would put a derivation record on an act that produced no
// new knowledge, which is how a trace stops meaning anything.
var observationMethod = mustMethod("ledger.DeriveHoldingObserved", "1")

// mintMethod names the derivation that brings an identity into existence.
//
// Minting is a derivation and not an observation: an identity is *computed*
// from a claim by a versioned method, and nobody told FDOS it exists. That is
// what makes `provenance.Derived` the right envelope for a mint (ADR-0033).
var mintMethod = mustMethod("ledger.MintFor", "1")

// resolves reports whether a mint answers a claim under a ruleset.
//
// Matching is on the **canonical seed**, not on bytes. `Claim.Equal` stays byte
// equality — it is a value-type operator, and two claims differing by a byte
// are two claims — but *resolution* is where sameness is decided, and ADR-0033
// gives that decision a stated, versioned rule.
//
// Matching on the seed rather than on the folded claim is what makes resolution
// agree with minting by construction: the seed is exactly what `identity.Derive`
// hashes, so two claims resolve together precisely when they would mint
// together.
func resolves(minted EntityMinted, claim identity.Claim, rules identity.Ruleset) bool {
	return rules.CanonicalSeed(minted.BornFrom) == rules.CanonicalSeed(claim)
}

// Resolve finds the identity a claim refers to, reading the ledger.
//
// **It reads. It does not re-resolve.** That is what makes replay deterministic
// (ADR-0009, ADR-0022): because minting is a fact, a 2031 replay of a 2026
// artifact reads the ledger as-of and finds the identity minted in 2026. There
// is no resolver to re-run and no opportunity to mint a second identity for the
// same thing.
//
// The guarantee holds because the resolution *result* is ledger content, not
// resolver behaviour. What the ruleset governs is which *recorded* mint answers
// a claim — so a ruleset change alters reads, which is why changing a shipped
// rule is an ADR-class act (ADR-0033).
//
// Resolution reads the ledger and nothing else. A resolver that ever consulted
// something outside it — a vendor's ticker-to-issuer table — would be consuming
// versioned reference data, and the binding would belong in the envelope
// (ADR-0010). That is not what this does; a ruleset is code, not data.
//
// Later mints do not shadow earlier ones: the *first* mint visible at the
// coordinate wins, in the stream's deterministic order. Two mints for the same
// claim is a defect the ledger records rather than hides, and merging them is
// an EntitiesIdentified decision, not this function's.
func Resolve(
	stream Stream,
	claim identity.Claim,
	rules identity.Ruleset,
	asOf temporal.AsOf,
) (identity.ID, error) {
	id, _, err := resolveWithTrace(stream, claim, rules, asOf)
	return id, err
}

// DeriveHoldingObserved turns a claimed holding into an observed one by
// resolving both claims.
//
// Returns Explained[HoldingObserved]: the result carries the derivation naming
// the claim fact it came from and the mints consumed, so "why is this account
// entity X" is data rather than a story told afterwards (ADR-0012).
//
// This is ADR-0011's rule applied where it was always meant to: deriving a
// canonical form from an observation is a domain computation with its own
// provenance, never an ingestion shortcut.
//
// Confidence propagates as the weakest input. A holding resolved through a mint
// FDOS is unsure of is not more trustworthy than that mint.
func DeriveHoldingObserved(
	stream Stream,
	claimedFact Fact,
	rules identity.Ruleset,
	asOf temporal.AsOf,
) (explained.Value[HoldingObserved], error) {
	claimed, ok := claimedFact.Payload().(HoldingClaimed)
	if !ok {
		return explained.Value[HoldingObserved]{},
			fmt.Errorf("%w: fact %s is not a HoldingClaimed", ErrEmptyType, claimedFact.Ref())
	}

	account, accountRef, err := resolveWithTrace(stream, claimed.Account, rules, asOf)
	if err != nil {
		return explained.Value[HoldingObserved]{}, err
	}
	instrument, instrumentRef, err := resolveWithTrace(stream, claimed.Instrument, rules, asOf)
	if err != nil {
		return explained.Value[HoldingObserved]{}, err
	}

	observed := HoldingObserved{
		Account:    account,
		Instrument: instrument,
		Quantity:   claimed.Quantity,
	}

	// Inputs are the claim fact and both mints. A trace that named only the
	// claim would be unable to explain which identity was chosen or why.
	//
	// `as_of` is a parameter and not an ornament: resolving the same claim at a
	// different coordinate can select a different mint, so two derivations that
	// omitted it would share a content address while disagreeing about the
	// answer.
	//
	// `canonicalisation` is here for the same reason. Which mint answers a claim
	// depends on the ruleset, so a derivation that omitted it would be unable to
	// explain a match that a later ruleset would not make (ADR-0033).
	return explained.FromDerivation(
		observed,
		observationMethod,
		[]string{claimedFact.Ref().String(), accountRef.String(), instrumentRef.String()},
		[]provenance.Parameter{
			{Name: "account_claim", Value: claimed.Account.String()},
			{Name: "instrument_claim", Value: claimed.Instrument.String()},
			{Name: "as_of", Value: asOf.String()},
			{Name: "canonicalisation", Value: rules.Version()},
		},
		nil,
		claimedFact.Envelope().Provenance().Confidence(),
	)
}

// resolveWithTrace resolves a claim and returns the mint fact that answered it,
// so the derivation can name it.
func resolveWithTrace(
	stream Stream,
	claim identity.Claim,
	rules identity.Ruleset,
	asOf temporal.AsOf,
) (identity.ID, Ref, error) {
	for _, f := range stream.VisibleAt(asOf) {
		minted, ok := f.Payload().(EntityMinted)
		if !ok {
			continue
		}
		if resolves(minted, claim, rules) {
			return minted.Entity, f.Ref(), nil
		}
	}
	return identity.ID{}, Ref{}, fmt.Errorf("%w: %s", ErrUnresolved, claim)
}

// MintFor builds the payload that brings an identity into existence for a
// claim, explained.
//
// The identity is derived from the claim's **canonicalised** value, once.
// `identity.Derive` is deterministic, so replaying the same acquisition produces
// the same identifier — and because the mint is then recorded as a fact, it is
// never re-derived (ADR-0007).
//
// Canonicalisation happens **here**, before Derive, never inside it:
// `canonicaliseSeed` stays the generic, scheme-blind floor (ADR-0033). What is
// recorded in `BornFrom` is the claim **verbatim**: it is the birth certificate,
// what the provider actually said, and the folded form is an input to the
// derivation rather than a replacement for the evidence.
//
// The returned trace is what makes the mint's provenance `Derived`. It names
// the claim fact being answered — when there is one, since RFC-0007 established
// that an account is often minted from operator configuration before any
// observation arrives — and records the ruleset version, so a mint made under
// one set of rules stays distinguishable from a mint made under another.
//
// Deciding *whether* to mint is not this function's: the application layer
// resolves first, and mints only when nothing answers (ADR-0033).
func MintFor(
	kind identity.Kind,
	claim identity.Claim,
	rules identity.Ruleset,
	answers []string,
	confidence provenance.Confidence,
) (explained.Value[EntityMinted], error) {
	if claim.IsZero() {
		return explained.Value[EntityMinted]{}, fmt.Errorf("%w: claim is unset", ErrEmptyType)
	}
	id, err := identity.Derive(kind, rules.Fold(claim).String())
	if err != nil {
		return explained.Value[EntityMinted]{}, err
	}

	return explained.FromDerivation(
		EntityMinted{Entity: id, BornFrom: claim},
		mintMethod,
		answers,
		[]provenance.Parameter{
			{Name: "claim", Value: claim.String()},
			{Name: "kind", Value: kind.String()},
			{Name: "seed", Value: rules.CanonicalSeed(claim)},
			{Name: "canonicalisation", Value: rules.Version()},
		},
		nil,
		confidence,
	)
}
