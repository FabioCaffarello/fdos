---
id: RFC-0005
title: Event taxonomy and schema evolution
status: Accepted
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0005 — Event taxonomy and schema evolution

## Summary

Proposes the vocabulary of facts FDOS stores, the distinction between things
that happened and things FDOS learned, and how schemas evolve without breaking
the reproducibility of events written years earlier.

## Motivation

Event-sourced systems fail in two characteristic ways, both of which this RFC
addresses.

**Taxonomy collapse.** Events named after storage operations — `PositionUpdated`,
`AccountChanged` — carry no domain meaning. The meaning ends up in the payload,
then in the consumer, and the ledger becomes a change log rather than a record of
facts. Recovering the intent afterwards is impossible: it was never written down.

**Schema drift.** An event schema written in 2026 must still be readable in 2036,
by code that has changed many times. The naive fix — migrate stored events — is
forbidden outright: it rewrites history, violating Constitution §4.

## Design

### Two kinds of fact

FDOS does not store "events" in the usual sense. It stores **facts**, of two
kinds, and the distinction is load-bearing:

| Kind | Asserts | Effective time means |
|------|---------|----------------------|
| **Occurrence** | Something happened in the world | When it happened |
| **Observation** | FDOS was told something is so | When the observed state held |

A trade settling is an Occurrence. A broker statement saying the position is 100
shares is an Observation — the position is not an event, and treating a statement
line as though it were a transaction fabricates history that did not occur.

Most connector output is Observations. Most domain reasoning wants Occurrences.
Deriving one from the other is a domain computation with its own provenance
(RFC-0004), never an ingestion shortcut.

This is the distinction that keeps Constitution §3 honest: a provider's
statement is an observation *of* a provider's view, not a fact about the world.

### Naming

- **Occurrences** are past tense and domain-specific: `DividendPaid`,
  `TradeSettled`, `BondMatured`, `SharesSplit`.
- **Observations** name what was observed: `HoldingObserved`,
  `PriceObserved`, `BalanceObserved`.
- **Corrections** name the correction, not the payload: `FactCorrected`,
  `FactRetracted`, `EntitiesIdentified` (RFC-0001).

Forbidden: `Created`, `Updated`, `Deleted`, `Changed`, `Synced`, `Imported`.
These describe what a database did, not what is true.

Each domain owns its vocabulary. A name is qualified by its domain
(`ledger.TradeSettled`), and cross-domain reuse of a name is a signal that a
concept has leaked.

### Schemas are per-type and independently versioned

There is no global schema version. Each fact type versions independently:

```
ledger.TradeSettled.v1
ledger.TradeSettled.v2
```

A global version forces unrelated types to move together, which in practice
means they never move.

### Evolution rules

Within a major version, changes are **additive only**: new optional fields.
Field numbers are never reused; removed fields are reserved permanently. This is
enforced by `buf breaking` in CI from M4.

Anything else is a new major version. Both versions remain readable forever,
because both exist in the ledger forever.

### Upcasting happens on read, and is versioned

Stored events are never migrated. A `v1` event written in 2026 is still a `v1`
event in 2036.

Reading code applies an **upcaster**: a pure, deterministic function
`v1 -> v2`. Upcasters are themselves versioned artifacts and are pinned in a
report's provenance (RFC-0004).

This resolves the question that makes long-horizon event sourcing hard:

> How do you replay a 2026 event in 2031 and get the 2026 answer?

You pin the upcaster chain, exactly as you pin reference datasets and calculation
methods. An upcaster that changes behaviour is a new upcaster version, not an
edit — the same law as everything else here.

An upcaster must be **total and lossless**: it cannot fail, and it cannot discard
information. If a v1 → v2 transformation would lose data, v2 is the wrong design.

### Correction semantics

Three distinct operations — the first two defined by RFC-0003, the third added
here:

| Operation | Asserts |
|-----------|---------|
| `FactCorrected` | The fact occurred, but was recorded with a wrong value |
| `FactRetracted` | The fact should never have been recorded |
| `FactSuperseded` | A better-sourced fact replaces this one; both were legitimate |

Collapsing these into one loses the distinction between "we were wrong", "it
didn't happen" and "we now have a better source" — three different answers to an
auditor, with three different downstream consequences.

### Streams

Facts are appended to a `LedgerStream` (RFC-0001), which provides the ordering
tiebreaker. Stream assignment is structural — derived from the aggregate the
fact concerns — never a routing decision, because a routing decision would make
ordering depend on configuration.

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| No CRUD-named fact types | 2 | Static check against a forbidden-name list (M2) |
| Occurrence vs Observation distinguished | 1 | Distinct envelope types; no implicit conversion |
| Additive-only within a major version | 3 | `buf breaking` in CI (M4) |
| Upcasters total and lossless | 3 | Property test: round-trip through the chain preserves all information |
| Upcasters deterministic | 2 | Pure functions in domain packages; M2 analyser applies |
| Correction kinds not collapsed | 1 | Three distinct types |

## Alternatives

**A single fact kind.** Simpler; the Occurrence/Observation distinction is
subtle. Rejected: without it, connectors fabricate occurrences from statements,
and the ledger records trades that never happened. This is the most consequential
proposal in this RFC.

**Migrate stored events on schema change.** Standard practice in many systems,
and much simpler than upcaster chains. Rejected outright: it rewrites history and
violates Constitution §4. The migration is also unreviewable after the fact.

**Global schema version.** Rejected: couples unrelated types and, in practice,
freezes them.

**Weak schemas (JSON, no contract).** Maximum flexibility at write time.
Rejected: it defers every compatibility question to read time, when the writer is
gone and the data cannot be re-derived.

**Upcast on write instead of read.** Faster reads, and only one schema in play.
Rejected: it is migration wearing a different name.

## Prior art

Greg Young's work on event sourcing establishes upcasting-on-read and forbids
event mutation for the reasons given. Protobuf's field-reservation discipline is
the mechanism that makes additive evolution safe. The Occurrence/Observation
split follows from the same reasoning behind bitemporality — separating what
happened from what was reported — and is where systems ingesting broker
statements most often go wrong.

## Open questions

- Is there a third kind — `Assertion`, for facts FDOS itself concludes (identity
  merges, classifications)? These are neither observed nor occurred. RFC-0001
  currently models them as Observations by FDOS-as-source, which may be a
  category error.
- Do upcaster chains compose across more than one major version, or is a direct
  `v1 -> v3` upcaster required? Composition is elegant but multiplies the
  behaviours that must be pinned.
- How is a fact type retired? Never appended again, but forever readable.

## Consequences

**Easier:** reading a decade-old event; understanding what the ledger means
without reading consumer code; distinguishing what happened from what was
reported.

**Harder:** more types, and a genuine modelling decision at every ingestion
point. Upcaster chains must be maintained and pinned indefinitely.

**Impossible:** migrating stored events; recording a broker's statement as
though it were a transaction.
