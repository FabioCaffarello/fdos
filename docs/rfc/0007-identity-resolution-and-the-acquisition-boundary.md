---
id: RFC-0007
title: Identity resolution and the acquisition boundary
status: Accepted
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0007 — Identity resolution and the acquisition boundary

> **Accepted**, recorded by
> [ADR-0022](../adr/0022-minting-an-identity-is-a-fact.md), which also settles
> the four open questions below.

## Summary

Answers [fdos#10](https://github.com/FabioCaffarello/fdos/issues/10): no message
published in `contracts@v0.2.0` can be fully populated by a connector, because
every message that carries identity requires an `EntityId` and a connector
cannot know one.

Proposes that **a connector emits a claim, and resolution is a derivation
recorded in the ledger** — not a precondition of appending, and not something a
connector does.

## Motivation

The finding was verified independently by enumerating every published message.
Exactly three carry identity, and all three require it:

| Message | Requires |
|---------|----------|
| `ledger.payload.v1.HoldingObserved` | 2 × `EntityId` |
| `kernel.v1.IdentifierAssertion` | 1 × `EntityId` |
| `kernel.v1.EntitiesIdentified` | 2 × `EntityId` |

Everything else in `kernel/v1` is a value type or a part. `ledger.v1.{Envelope,
Fact, Correction}` wrap a payload and add nothing.

A connector reads `"PETR4"` off a page. It knows `{scheme: "ticker", value:
"PETR4"}` exactly as the provider stated it. It cannot know the `EntityId`, and
**it must not mint one**: ADR-0007 is explicit that deriving from a ticker would
make the ticker the primary key, and "the day the external world reused one, two
different things would silently merge with no recovery."

The gap is not the vocabulary. `IdentifierAssertion.scheme` was deliberately left
an open string so that a new institution would not force a public contract
release — that field was shaped for private connectors. What is missing is the
**hand-off**: what a connector emits when it knows `{scheme, value}` and cannot
know the entity.

There is a second, sharper problem hiding underneath. `IdentifierAssertion` is
the shape that would carry the claim, and it requires the identity it exists to
assert. That circularity is not an oversight in the message; it is a missing
event. **An `EntityId` comes into existence at some moment, and FDOS has never
said what that moment is.**

## Design

### Minting an identity is an event

This is the decision everything else follows from.

ADR-0007 already established the shape for the analogous case:

> Merging is recorded, never performed. A wrong merge is therefore reversible by
> a new fact that retracts it — where a computed merge would be unrecoverable,
> because the pre-merge state would no longer exist.

The same argument applies to *minting*. An `EntityId` that appears without a
recorded origin is an identity nobody can audit, and a wrong one cannot be
retracted because there is nothing to retract.

So: `EntityMinted` is a fact. It carries the claim the identity was born from,
its own provenance, and its own temporal coordinates. It is the birth
certificate the kernel documentation already describes, written down.

That breaks the circularity. `IdentifierAssertion` requires an `EntityId`
because it asserts something *about* an entity — and after `EntityMinted` there
is one.

### A connector emits a claim, and the claim is a fact

```
connector ──► HoldingClaimed ──► ledger
                    │
                    │  (derivation, recorded)
                    ▼
              EntityMinted / IdentifierAssertion  ──► ledger
                    │
                    ▼
              HoldingObserved ──► ledger
```

`HoldingClaimed` is the same shape as `HoldingObserved` with `IdentifierClaim`
in place of each `EntityId`:

```protobuf
message IdentifierClaim {
  string scheme = 1;  // "ticker", "isin", "account_number" — open vocabulary
  string value  = 2;  // verbatim as the provider stated it
}

message HoldingClaimed {
  IdentifierClaim account    = 1;
  IdentifierClaim instrument = 2;
  fdos.kernel.v1.Quantity quantity = 3;
}
```

It is a full fact with a full envelope: kind `OBSERVATION`, provenance naming
the connector and its parser version, both temporal axes. Nothing about it is
provisional.

### The claim reaches the ledger. Resolution is not a precondition of appending

This answers the question the issue asks fourth, and it is the answer with the
most consequence.

An unresolved observation **is** a fact — the most primitive one FDOS has. It is
what the provider actually said, before FDOS decided what any of it referred to.
Requiring resolution before appending would mean:

- the raw claim is lost, so a resolution later found wrong cannot be re-done
  from evidence;
- an unresolvable observation is dropped at the door, and Constitution §4 has no
  provision for facts FDOS declines to remember;
- ingestion becomes a place where truth is decided silently, which is exactly
  what §6 exists to prevent.

Appending the claim first also means a connector's output is never blocked on a
resolver being available, correct, or fast.

### `HoldingObserved` becomes derived, and says so

Today `HoldingObserved` is written as though observed directly. Under this
proposal a connector never emits one — FDOS derives it, and its provenance is
`Derived` with a `DerivationRef` naming:

- the `HoldingClaimed` fact it came from,
- each `IdentifierAssertion` consumed to resolve a claim.

That is ADR-0011's rule applied where it was always meant to apply: *"Deriving
occurrences from observations is a domain computation with its own provenance,
never an ingestion shortcut."*

Confidence propagates as the weakest input (ADR-0010). An instrument resolved by
an exact ISIN match and one resolved by a reused ticker do not deserve the same
confidence, and the ordinal scale already carries the difference.

### Replay is deterministic because resolution is in the ledger

The issue's fifth question is the one that decides whether this design is sound,
and the answer is structural rather than procedural.

**Replay does not re-resolve. It reads.**

Because minting and assertion are facts, a 2031 replay of a 2026 artifact reads
the ledger as-of and finds the `EntityId` that was minted in 2026. There is no
resolver to re-run and no opportunity to mint a second identity for the same
thing.

This is precisely Constitution §9's formulation — *"the same ledger and the same
versioned reference datasets"* — and the reason the guarantee holds is that the
resolution **result** is ledger content, not resolver behaviour.

A corollary worth stating, because it answers the collision the downstream RFC
recorded: **resolution reads the ledger, not an external map.** If a resolver
ever consults something outside the ledger — a vendor's ticker-to-issuer table —
that is versioned reference data consumed by the derivation, and the binding
belongs in `Envelope.references` (ADR-0010). It is not currently needed, and
adding a resolver that needs it is a change that must say so.

### Account and instrument have the same shape and different resolvers

The issue's sixth question is right to ask, and the answer is: one mechanism,
two resolvers, different confidence.

| | Account | Instrument |
|---|---|---|
| Scope | one holder at one institution | global, shared |
| Resolver | host configuration; the account is known before the first fetch | a real resolver over prior assertions |
| Typical confidence | `ASSERTED` | `ASSERTED` on an ISIN match, `INFERRED` on a ticker |
| Ambiguity | rare | expected |

An account can often be minted before any observation arrives, because the
operator configured the connector to fetch it. That is still an `EntityMinted`
fact — it just happens earlier and from a different source.

### What a connector may never do

Stated so the boundary is not eroded by convenience:

- mint an `EntityId`,
- resolve a claim,
- emit `HoldingObserved`, `IdentifierAssertion` or `EntitiesIdentified`,
- normalise a claim value. `"PETR4 "` with a trailing space is what the provider
  said, and canonicalisation is a resolver's decision to record, not a parser's
  to make silently.

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| A claim carries no identity | 1 | `IdentifierClaim` has no `EntityId` field |
| Minting is recorded | 1 | `EntityId` only reaches a fact through a payload; `EntityMinted` is the only fact that introduces one |
| Derived observations name their inputs | 1 | `provenance.Derived` requires a `DerivationRef` (ADR-0010) |
| A connector cannot emit a resolved payload | 3 | conformance suite in the connector SDK; a plugin returning `HoldingObserved` fails |
| Replay does not re-resolve | 3 | property test: replaying a claim against a ledger containing its resolution yields the same `EntityId` |

The fourth is the one that needs care. It is a downstream mechanism, and FDOS
cannot enforce it — which is an argument for the SDK type being deliberately
unable to express a resolved payload, rather than merely discouraged from it.

## Alternatives

**The connector resolves, calling an FDOS resolver port.** Rejected. It couples
every private connector to a running FDOS service, turns acquisition into a
synchronous distributed operation, and puts a decision about financial truth
behind a network call that can fail. It also inverts the open-core dependency
direction: a connector would need FDOS at runtime, not just its contracts.

**The connector mints, deterministically, from `{scheme, value}`.** Superficially
attractive: no new message, no resolver, and replay is trivially deterministic.
Rejected because it makes the external identifier the primary key, which ADR-0007
rejected with the failure named — a reused ticker silently merges two
instruments inside an append-only ledger. The determinism it buys is the
determinism of being consistently wrong.

**Resolution as a precondition of appending.** Rejected above: it loses the raw
claim, drops unresolvable observations, and moves a truth decision into a place
with no provenance.

**Leave `EntityId` optional in `HoldingObserved`.** The minimal change: one
message, no new facts. Rejected because "sometimes populated" is the shape that
makes every downstream consumer write a nil check and guess what absence means.
It also cannot express *why* an identity is missing, which is the thing a
resolver most needs to know.

**A separate "staging" store outside the ledger for unresolved claims.**
Rejected: it is a second place financial data lives, with its own durability and
provenance story, and Constitution §1 exists to prevent exactly that.

## Prior art

Master data management systems converge on the same split — a landing record of
what a source said, then a match-and-merge step whose output is auditable — and
the ones that fail are the ones where the match is performed rather than
recorded. Event-sourced systems that resolve identity at ingestion without
recording the resolution reliably develop a reconciliation job between the
identity map and the event stream, which is the failure ADR-0007 anticipated.

## Open questions

- **`EntityMinted` and `IdentifierAssertion` may be the same fact.** Minting
  always accompanies a first assertion, and two facts where one would do is the
  kind of thing that looks tidy in an RFC and tedious for a decade. Deciding
  this is part of accepting the proposal.
- **What happens to a claim that never resolves?** It stays in the ledger,
  unresolved, and no `HoldingObserved` is derived from it. Whether that is
  reported, and to whom, is an operational question this RFC does not answer.
- **Does `EntitiesIdentified` change?** It merges two existing identities and is
  unaffected — but a resolver that discovers two mints were the same thing is
  its most likely producer, and that path is not designed here.
- **Confidence for a resolver that had to choose.** An `INFERRED` instrument is
  honest, but a downstream consumer has no way to ask *how* inferred. Whether the
  derivation record is sufficient is unresolved.

## Consequences

**Easier:** a connector can be written at all. A wrong resolution is
correctable, because the claim it came from is still in the ledger. Replay is
deterministic without a resolver being reproducible.

**Harder:** two facts per observation instead of one, and a resolution step that
must exist before any position can be projected. Ledger volume grows.

**Impossible:** minting an identity without recording where it came from;
losing what a provider actually said.

## Release consequence

Additive, and therefore a minor bump — `contracts@v0.3.0`:

- `kernel.v1.IdentifierClaim`
- `ledger.payload.v1.HoldingClaimed`
- `ledger.payload.v1.EntityMinted`

No existing message changes shape, so `buf breaking` passes and no consumer is
forced to migrate. `HoldingObserved` gains a documented obligation — its
provenance must be `Derived` — which is a rule the wire cannot express and the
Go kernel can.

Downstream this moves a platform pin and a module pin, and unblocks
`fdos-connectors` B-009, C2 and C4. Stated here because the issue asked for it
explicitly rather than leaving it to be discovered.
