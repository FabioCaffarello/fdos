---
id: RFC-0001
title: Identity and aggregate boundaries in the Canonical Financial Model
status: Accepted
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0001 — Identity and aggregate boundaries in the Canonical Financial Model

## Summary

Proposes how FDOS identifies the things it stores facts about, and where the
consistency boundaries lie. Nothing else in the canonical model can be designed
first: an event schema cannot be written without knowing what an event is *about*,
and a graph projection cannot exist without stable node identity.

This needs an RFC rather than an ADR because the obvious answer — use ISINs and
account numbers — is wrong in ways that only surface years later.

## Motivation

External financial identifiers are not identities. They are **claims about**
entities, made by parties, at points in time, and they are unstable in ways that
silently corrupt history:

- Tickers are reused. `FB` and `META` are the same issuer; a 1998 `AAPL` and a
  2026 `AAPL` are the same, but many recycled tickers are not.
- ISINs are reassigned after long dormancy, and change on some corporate actions.
- Account numbers change on institutional migration while the account continues.
- Two providers disagree about whether two instruments are the same instrument.

If FDOS uses an external identifier as its primary key, then the day the external
world reassigns it, history silently merges two different things. There is no
recovery: the facts have already been filed under the wrong entity.

**Not retrofittable.** Identity assigned at first write cannot be re-derived
later, because the information needed to distinguish the entities was discarded
at write time.

## Design

### Identity is internal, deterministic, and assigned once

Every entity carries an opaque internal identifier. External identifiers are
stored as timestamped, sourced *assertions about* that entity — never as its key.

```
Entity        := EntityID (opaque, internal)
IdentifierAssertion := {
    entity:     EntityID
    scheme:     ISIN | CUSIP | FIGI | Ticker | AccountNumber | ...
    value:      string
    asserted_by: source
    effective:  interval        # RFC-0003
    provenance: Provenance      # RFC-0004
}
```

An instrument does not *have* an ISIN. A source *asserted*, over some interval,
that this entity is identified by that ISIN. That assertion can later be
contradicted, and the contradiction is a new fact rather than an update.

### EntityIDs are derived, not random

Constitution §2 requires reproducibility. A random UUID assigned at ingestion
means replaying the same input twice produces different identifiers, and no
report is byte-reproducible.

Proposal: `EntityID = UUIDv5(namespace, canonical_seed)` where the seed is the
canonicalised first-observed natural key, and the namespace is per entity kind.

Replaying identical input yields identical identifiers. There is no coordination
service, no sequence, and no ordering leak.

### The seed is a birth certificate, not a key

The seed determines the identifier once, at first observation. It is never used
for lookup and never re-derived. If the seed's underlying facts later change —
the ISIN is reassigned, the company renames — the identifier does not move,
because the identifier was never a function of current state.

This is the crux: the seed makes creation *deterministic*, while assertions make
identity *stable*.

### Identity resolution is a fact, not a computation

When two observations are believed to describe the same entity, FDOS does not
silently merge them. It records an assertion:

```
EntitiesIdentified := {
    canonical: EntityID
    alias:     EntityID
    basis:     rule id + inputs
    confidence: Confidence        # RFC-0004
    provenance: Provenance
}
```

Merging is reversible because it was recorded rather than performed. A wrong
merge is corrected by a new fact that retracts it, exactly like every other
correction in FDOS. If merging were a computation, an incorrect merge would be
unrecoverable — the pre-merge state would no longer exist.

This also gives graph projection (Constitution §12) what it needs: nodes with
stable identity and explicit, weighted `same_as` edges.

### Aggregates

An aggregate is a consistency boundary — the unit within which invariants must
hold atomically. FDOS has few, because most financial concepts are projections.

**Proposed aggregates:**

| Aggregate | Invariant it protects |
|-----------|-----------------------|
| `Instrument` | The identity and terms of a tradable thing |
| `Party` | The identity of an issuer, counterparty or institution |
| `Account` | The custodial container facts are filed against |
| `LedgerStream` | Append-only ordering of entries for one account |

**Explicitly not aggregates — these are projections:**

`Position`, `Balance`, `Performance`, `Exposure`, `Allocation`.

This is the direct consequence of Constitution §1. A position is not a thing
that exists and changes; it is the answer to a question asked of the ledger at a
given effective and knowledge time. Modelling it as an aggregate would create a
second place where truth lives, and the two would diverge.

### Naming

Identifiers are opaque in code and never parsed. Any code that reads structure
out of an `EntityID` has created a coupling that the next identifier scheme
breaks.

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| External identifiers never used as primary keys | 1 | `EntityID` is a distinct type; assertion types cannot be used where an `EntityID` is required |
| `Position` etc. are not persisted as state | 2 | No write path accepts a projection type; enforced by the M2 boundary analyser |
| Identifiers are opaque | 1 | No exported accessors for internal structure |
| Merges are recorded, not performed | 3 | Property test: replaying a merge and its retraction restores the pre-merge projection exactly |

## Alternatives

**Natural keys (ISIN, account number) as primary keys.** Simplest, and every
provider hands them to us. Rejected: they are mutable, reassignable, contested
between sources, and provider-specific — a direct violation of Constitution §3.
The failure is silent and unrecoverable.

**Random UUIDv4 at ingestion.** Stable and collision-free. Rejected: it destroys
replay determinism, so the same input produces different output and no report is
byte-reproducible.

**Monotonic sequence assigned by the ledger.** Deterministic given a fixed order.
Rejected: it requires a central allocator, makes ingestion order part of
identity, and leaks ordering into the identifier — which then invites code to
depend on it.

**Content-addressed identity (hash of all current attributes).** Elegant, and
self-verifying. Rejected: identity would change whenever any attribute changed,
which is the opposite of what an identity is for.

## Prior art

FIGI exists precisely because ISIN-as-identity failed at scale; its design
lesson — a permanent identifier that is never reused and never carries meaning —
is the one adopted here. Event-sourced systems that model positions as
aggregates (rather than folds) reliably develop reconciliation jobs between the
aggregate and the event stream, which is the specific failure Constitution §1
exists to prevent.

## Open questions

- Which entity kinds get their own namespace, and is that list closed? Adding a
  namespace later is safe; changing one is not.
- Canonicalisation of the seed must itself be deterministic and versioned. If
  the canonicalisation algorithm changes, identifiers for *new* entities change.
  Does the algorithm version belong in the namespace?
- Does `Account` belong in the public core, given that its shape is largely
  driven by private connectors?

## Consequences

**Easier:** correcting a wrong merge; supporting contradictory sources without
choosing a winner at ingestion; projecting to a graph.

**Harder:** every lookup by ISIN becomes a query over assertions with a temporal
qualifier. There is no `WHERE isin = ?`. This is a real and permanent ergonomic
cost, and the honest justification is that the alternative is silent historical
corruption.

**Impossible:** filing a fact against an entity without having first decided
that the entity exists.
