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

// Resolve finds the identity a claim refers to, reading the ledger.
//
// **It reads. It does not re-resolve.** That is what makes replay deterministic
// (ADR-0009, ADR-0022): because minting is a fact, a 2031 replay of a 2026
// artifact reads the ledger as-of and finds the identity minted in 2026. There
// is no resolver to re-run and no opportunity to mint a second identity for the
// same thing.
//
// The guarantee holds because the resolution *result* is ledger content, not
// resolver behaviour.
//
// Resolution reads the ledger and nothing else. A resolver that ever consulted
// something outside it — a vendor's ticker-to-issuer table — would be consuming
// versioned reference data, and the binding would belong in the envelope
// (ADR-0010). That is not what this does.
//
// Later mints do not shadow earlier ones: the *first* mint visible at the
// coordinate wins, in the stream's deterministic order. Two mints for the same
// claim is a defect the ledger records rather than hides, and merging them is
// an EntitiesIdentified decision, not this function's.
func Resolve(stream Stream, claim identity.Claim, asOf temporal.AsOf) (identity.ID, error) {
	for _, f := range stream.VisibleAt(asOf) {
		minted, ok := f.Payload().(EntityMinted)
		if !ok {
			continue
		}
		if minted.BornFrom.Equal(claim) {
			return minted.Entity, nil
		}
	}
	return identity.ID{}, fmt.Errorf("%w: %s", ErrUnresolved, claim)
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
	asOf temporal.AsOf,
) (explained.Value[HoldingObserved], error) {
	claimed, ok := claimedFact.Payload().(HoldingClaimed)
	if !ok {
		return explained.Value[HoldingObserved]{},
			fmt.Errorf("%w: fact %s is not a HoldingClaimed", ErrEmptyType, claimedFact.Ref())
	}

	account, accountRef, err := resolveWithTrace(stream, claimed.Account, asOf)
	if err != nil {
		return explained.Value[HoldingObserved]{}, err
	}
	instrument, instrumentRef, err := resolveWithTrace(stream, claimed.Instrument, asOf)
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
	return explained.FromDerivation(
		observed,
		observationMethod,
		[]string{claimedFact.Ref().String(), accountRef.String(), instrumentRef.String()},
		[]provenance.Parameter{
			{Name: "account_claim", Value: claimed.Account.String()},
			{Name: "instrument_claim", Value: claimed.Instrument.String()},
			{Name: "as_of", Value: asOf.String()},
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
	asOf temporal.AsOf,
) (identity.ID, Ref, error) {
	for _, f := range stream.VisibleAt(asOf) {
		minted, ok := f.Payload().(EntityMinted)
		if !ok {
			continue
		}
		if minted.BornFrom.Equal(claim) {
			return minted.Entity, f.Ref(), nil
		}
	}
	return identity.ID{}, Ref{}, fmt.Errorf("%w: %s", ErrUnresolved, claim)
}

// MintFor builds the payload that brings an identity into existence for a
// claim.
//
// The identity is derived from the claim, once. `identity.Derive` is
// deterministic, so replaying the same acquisition produces the same identifier
// — and because the mint is then recorded as a fact, it is never re-derived
// (ADR-0007).
//
// Deciding *whether* to mint is not this function's: the application layer
// resolves first, and mints only when nothing answers.
func MintFor(kind identity.Kind, claim identity.Claim) (EntityMinted, error) {
	if claim.IsZero() {
		return EntityMinted{}, fmt.Errorf("%w: claim is unset", ErrEmptyType)
	}
	id, err := identity.Derive(kind, claim.String())
	if err != nil {
		return EntityMinted{}, err
	}
	return EntityMinted{Entity: id, BornFrom: claim}, nil
}
