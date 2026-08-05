package domain

import (
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
)

// HoldingClaimed is what a connector emits (ADR-0022).
//
// The same shape as HoldingObserved with an identity.Claim in place of each
// identity.ID. A connector reads "PETR4" off a page and knows
// {scheme: "ticker", value: "PETR4"} — it cannot know an ID, and must not mint
// one.
//
// A full fact with a full envelope, and it reaches the ledger. Resolution is a
// derivation recorded afterwards, never a precondition of appending: requiring
// resolution first would lose the raw claim, so a resolution later found wrong
// could not be re-done from evidence.
type HoldingClaimed struct {
	Account    identity.Claim
	Instrument identity.Claim
	Quantity   money.Quantity
}

// FactType returns the domain-qualified type name.
func (HoldingClaimed) FactType() string { return "ledger.HoldingClaimed" }

// TypeVersion returns the per-type major version.
func (HoldingClaimed) TypeVersion() uint32 { return 1 }

// EntityMinted records that an identity came into existence, and the claim it
// was born from (ADR-0022).
//
// The missing event. Before this, an identity.ID appeared in a fact with no
// recorded origin — unauditable, and a wrong one unretractable because there
// was nothing to retract.
//
// Exactly one mint per ID. That cardinality is why this is a separate fact from
// an identifier assertion, which is many-per-entity from many sources: merging
// the two would make "the assertion that happens to be first" special, and
// orderings change as facts arrive with earlier effective times.
type EntityMinted struct {
	Entity   identity.ID
	BornFrom identity.Claim
}

// FactType returns the domain-qualified type name.
func (EntityMinted) FactType() string { return "ledger.EntityMinted" }

// TypeVersion returns the per-type major version.
func (EntityMinted) TypeVersion() uint32 { return 1 }
