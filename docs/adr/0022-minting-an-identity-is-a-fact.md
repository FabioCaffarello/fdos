---
id: ADR-0022
title: Minting an identity is a fact, and a connector emits a claim
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0022 — Minting an identity is a fact, and a connector emits a claim

## Context

Records what [RFC-0007](../rfc/0007-identity-resolution-and-the-acquisition-boundary.md)
settled, in answer to [fdos#10](https://github.com/FabioCaffarello/fdos/issues/10).

No message published in `contracts@v0.2.0` could be fully populated by a
connector. Exactly three carry identity and all three require an `EntityId`,
which a connector cannot know and must not mint.

The cause was not a missing message. `IdentifierAssertion` requires the identity
it exists to assert, and that circularity exposed a **missing event**: an
`EntityId` comes into existence at some moment, and FDOS had never said what
that moment is.

## Decision

### Minting is a fact

`EntityMinted` records that an `EntityId` came into existence and the claim it
was born from. It carries a full envelope like any other fact.

ADR-0007 already settled the analogous case — *"merging is recorded, never
performed, so a wrong merge is reversible by a new fact that retracts it"* — and
the same argument governs minting. An identity that appears without a recorded
origin cannot be audited, and a wrong one cannot be retracted because there is
nothing to retract.

### A connector emits a claim, and the claim is a fact

`IdentifierClaim` is `{scheme, value}` verbatim as a provider stated it.
`HoldingClaimed` is `HoldingObserved` with a claim in place of each `EntityId`.

It is a full fact with a full envelope, and **it reaches the ledger**.
Resolution is a derivation recorded afterwards, not a precondition of appending.

### `HoldingObserved` is derived, never observed directly

A connector never emits one. FDOS derives it, and its provenance is `Derived`
with a `DerivationRef` naming the `HoldingClaimed` fact and each
`IdentifierAssertion` consumed.

This is ADR-0011's rule applied where it was always meant to: *"Deriving
occurrences from observations is a domain computation with its own provenance,
never an ingestion shortcut."*

### Resolution reads the ledger, not an external map

A resolver that consults something outside the ledger — a vendor's
ticker-to-issuer table — is consuming versioned reference data, and the binding
belongs in `Envelope.references` (ADR-0010).

Not currently needed. Adding a resolver that needs it is a change that must say
so.

### The open questions RFC-0007 left, now settled

**`EntityMinted` and `IdentifierAssertion` stay two facts.** They protect
different invariants: exactly one mint per `EntityId`, and many assertions per
`EntityId` from many sources over many intervals. Merging them would make *"the
assertion that happens to be first"* special — a rule about ordering rather than
about the fact, and orderings change as facts arrive with earlier effective
times.

**A claim that never resolves stays in the ledger, unresolved.** No
`HoldingObserved` is derived from it, and a projection over it returns
`ErrNoFacts`. M6 already distinguished *"we hold nothing"* from *"we know
nothing"*, and this is the second. Reporting unresolved claims is operational
and not decided here.

**`EntitiesIdentified` is unchanged.** It merges two existing identities. A
resolver discovering that two mints were the same thing is its most likely
producer, and that path is not designed here.

**No confidence field is added.** *"How inferred"* is answerable from the
derivation record, which already carries the method and its parameters. A second
place to record the same thing would drift from the first.

## Consequences

### Positive

- A connector can be written at all, which was the blocking problem.
- A wrong resolution is correctable: the claim it came from is still in the
  ledger, with provenance.
- Replay is deterministic without the resolver being reproducible. It does not
  re-resolve — it reads, and finds the identity minted at the time.
- Identity acquires an audit trail it did not have. Before this, an `EntityId`
  in a fact had no recorded origin at all.

### Negative

- **Two facts per observation instead of one**, plus a mint the first time an
  entity is seen. Ledger volume grows, and a position cannot be projected until
  resolution has run.
- **A resolution step now exists that can fail or lag.** Claims accumulate
  unresolved, and nothing here says who notices. That is a real operational gap
  and it is not closed by this decision.
- `HoldingObserved` gains an obligation the wire cannot express — its provenance
  must be `Derived`. proto3 has no way to require it, so this is a Go-side
  invariant and a review obligation until the domain type enforces it.
- The claim vocabulary is an open string. A connector emitting `"Ticker"` where
  another emits `"ticker"` produces two entities, and nothing in the contract
  prevents it. Canonicalisation is a resolver decision; whose decision it is at
  the boundary is unresolved.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| A claim carries no identity | 1 | `IdentifierClaim` has no `EntityId` field |
| Every ledger fact carries an envelope | 3 | `make proto-check` |
| No published message references model output | 3 | `make proto-check` |
| `HoldingObserved` provenance is `Derived` | 6 | review, until the Go domain type enforces it |
| A connector cannot emit a resolved payload | 6 | downstream; FDOS cannot enforce it |

The last two are rung 6 and are the honest weak points of this decision.

## Alternatives considered

Recorded in full in RFC-0007. The two that were closest:

**The connector mints deterministically from `{scheme, value}`.** No new message,
no resolver, trivially deterministic replay. Rejected because it makes the
external identifier the primary key — the failure ADR-0007 named, where a reused
ticker silently merges two instruments inside an append-only ledger. The
determinism it buys is the determinism of being consistently wrong.

**Leave `EntityId` optional in `HoldingObserved`.** One message, no new facts.
Rejected because *"sometimes populated"* makes every consumer write a nil check
and guess what absence means, and it cannot express **why** an identity is
missing — the thing a resolver most needs to know.

## Notes

Released as `contracts@v0.3.0`, additive: `kernel.v1.IdentifierClaim`,
`ledger.payload.v1.HoldingClaimed`, `ledger.payload.v1.EntityMinted`. No
existing message changes shape, so `buf breaking` passes and no consumer is
forced to migrate.

Not yet built, and tracked as B-007:

- Go domain types for claims and mints, and the resolution derivation itself.
- Codec and round-trip conformance for the new payloads. Until those exist,
  `encodePayload` rejects them — loudly, by design, rather than emitting an
  empty `Any`.

Unblocks `fdos-connectors` B-009, C2 and C4.
