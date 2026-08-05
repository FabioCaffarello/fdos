---
id: ADR-0011
title: Facts are Occurrences or Observations and schemas evolve by upcast-on-read
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0011 — Facts are Occurrences or Observations and schemas evolve by upcast-on-read

Records the acceptance of [RFC-0005](../rfc/0005-event-taxonomy-and-schema-evolution.md).

## Context

Event-sourced systems fail two ways. Taxonomy collapse: events named after
storage operations (`PositionUpdated`) carry no domain meaning, and the intent
is never recoverable because it was never written down. Schema drift: a schema
written in 2026 must be readable in 2036, and the naive fix — migrating stored
events — rewrites history, which Constitution §4 forbids outright.

The deeper trap is specific to FDOS's ingestion model: a broker statement line
is not a transaction, and recording it as one fabricates history that did not
occur.

Serves Constitution §3 (canonical model first), §4 (immutable ledger) and §5
(event sourcing).

## Decision

FDOS stores facts of two kinds, as distinct envelope types with no implicit
conversion:

- **Occurrence** — something happened in the world (`TradeSettled`,
  `DividendPaid`). Effective time is when it happened.
- **Observation** — FDOS was told something is so (`HoldingObserved`,
  `PriceObserved`). Effective time is when the observed state held.

Most connector output is Observations; most domain reasoning wants Occurrences;
deriving one from the other is a domain computation with its own provenance
(ADR-0010), never an ingestion shortcut. This distinction is the most
consequential part of the decision.

Naming: Occurrences are past-tense domain verbs; Observations name what was
observed; `Created`/`Updated`/`Deleted`/`Changed`/`Synced`/`Imported` are
forbidden. Each domain owns its vocabulary, qualified by domain.

Schemas version per fact type, independently — no global schema version.
Within a major version changes are additive-only, field numbers never reused.
Anything else is a new major version, and every version remains readable
forever. Stored events are **never migrated**: reading applies upcasters —
pure, deterministic, versioned `vN → vN+1` functions, total and lossless,
pinned in a report's provenance exactly as reference datasets are.

Three correction operations remain distinct: `FactCorrected` (wrong value),
`FactRetracted` (should never have been recorded), `FactSuperseded` (a
better-sourced fact replaces a legitimate one) — three different answers to an
auditor.

Stream assignment is structural, derived from the aggregate the fact concerns
(ADR-0007), never a routing decision.

## Consequences

### Positive

- A decade-old event is readable, and replaying it under a pinned upcaster
  chain gives the original answer.
- The ledger states what it means without reading consumer code.
- What happened and what was reported are never conflated — the ledger cannot
  record trades that never occurred.

### Negative

- A genuine modelling decision at every ingestion point: is this an Occurrence
  or an Observation? Getting it wrong is now visible, but it must be made.
- Upcaster chains are maintained and pinned indefinitely; type count grows.

### Enforcement

Today: rung 5 — there is no code. From M2: rung 1 (distinct envelope types,
three distinct correction types) and rung 2 (static check against the
forbidden-name list; the purity analyser covers upcasters). From M4: rung 3
(`buf breaking` enforces additive-only evolution). From M6: rung 3 property
test that a round-trip through an upcaster chain preserves all information.

## Alternatives considered

- **A single fact kind** — connectors fabricate occurrences from statements;
  the ledger records trades that never happened.
- **Migrating stored events** — rewrites history; unreviewable after the fact.
- **A global schema version** — couples unrelated types and in practice
  freezes them.
- **Weak schemas (JSON, no contract)** — defers every compatibility question
  to read time, when the writer is gone.
- **Upcast on write** — migration wearing a different name.

Full exploration in RFC-0005.

## Notes

Open, deliberately: whether a third kind — `Assertion`, for facts FDOS itself
concludes, such as `EntitiesIdentified` (ADR-0007) — is needed; modelling those
as Observations by FDOS-as-source may be a category error, and this must be
resolved before the M6 vocabulary is cut. Also open: whether upcaster chains
compose across major versions or a direct upcaster is required; how a fact type
is retired (never appended again, forever readable).
