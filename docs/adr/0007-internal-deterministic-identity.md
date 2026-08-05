---
id: ADR-0007
title: Entity identity is internal, deterministic and assertion-based
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0007 — Entity identity is internal, deterministic and assertion-based

Records the acceptance of [RFC-0001](../rfc/0001-identity-and-aggregate-boundaries.md).

## Context

External financial identifiers — ISINs, tickers, account numbers — are claims
about entities, not identities. They are reused, reassigned, changed by
corporate actions and contested between providers. Keying facts by one means
that the day the external world reassigns it, history silently merges two
different things, and the facts have already been filed under the wrong entity.

Random identifiers avoid that failure but destroy replay determinism
(Constitution §2): the same input ingested twice yields different identifiers,
and no report is byte-reproducible.

Identity is not retrofittable. The information needed to distinguish two
entities exists only at first write; identity assigned wrongly then cannot be
re-derived later. This decision therefore precedes every event schema.

Serves Constitution §2 (determinism), §3 (canonical model first) and §12 (graph
projection needs stable node identity).

## Decision

FDOS assigns every entity an opaque internal `EntityID`, derived
deterministically as `UUIDv5(namespace, canonical_seed)` — the namespace per
entity kind, the seed the canonicalised first-observed natural key. The seed is
a birth certificate, not a key: it fixes the identifier once and is never used
for lookup or re-derived.

External identifiers are stored as timestamped, sourced `IdentifierAssertion`
facts about an entity — never as its key. A source asserted, over an interval,
that this entity carries that ISIN; a later contradiction is a new fact.

Identity resolution is recorded, never performed: an `EntitiesIdentified` fact
links a canonical entity and an alias with basis, confidence and provenance. A
wrong merge is corrected by a retracting fact; the pre-merge state always
remains derivable.

The aggregates are `Instrument`, `Party`, `Account` and `LedgerStream`.
`Position`, `Balance`, `Performance`, `Exposure` and `Allocation` are
projections and are never persisted as state (Constitution §1).

## Consequences

### Positive

- Replaying identical input yields identical identifiers — no coordination
  service, no ordering leak.
- Contradictory sources coexist as assertions; nothing forces choosing a winner
  at ingestion.
- Merges are reversible because they were recorded rather than performed.
- Graph projection gets stable nodes and explicit, weighted `same_as` edges for
  free.

### Negative

- There is no `WHERE isin = ?`. Every lookup by external identifier becomes a
  query over assertions with a temporal qualifier. This is a permanent ergonomic
  tax, paid so that history cannot silently corrupt.
- Seed canonicalisation must itself be deterministic and versioned; changing the
  algorithm changes identifiers for *new* entities. That versioning question is
  still open and must be settled before the first ingestion path exists.

### Enforcement

Today: rung 5 — there is no code. The design is held at rung 1 from M2
(`EntityID` a distinct opaque type; assertion types unusable where an `EntityID`
is required; no write path accepts a projection type, checked by the M2 boundary
analyser) and rung 3 from M6 (property test: replaying a merge and its
retraction restores the pre-merge projection exactly).

## Alternatives considered

- **Natural keys as primary keys** — mutable, reassignable, contested; silent
  and unrecoverable failure.
- **Random UUIDv4** — destroys replay determinism.
- **Ledger-assigned monotonic sequence** — central allocator; ordering leaks
  into identity and invites dependence on it.
- **Content-addressed identity** — the identifier moves whenever an attribute
  changes, the opposite of identity.

Full exploration in RFC-0001.

## Notes

Open, deliberately: which entity kinds get namespaces and whether that list is
closed; whether the canonicalisation algorithm version belongs in the namespace;
whether `Account` belongs in the public core given its shape is driven by
private connectors. All must be resolved no later than the M6 ledger slice.
