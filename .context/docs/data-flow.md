---
type: doc
name: data-flow
description: Intended flow from external source to derived view — design intent, not implementation
category: data-flow
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Data Flow

> **Nothing described here is implemented.** There is no connector, parser,
> ledger, projection or API in this repository. This documents the flow the
> architecture is designed to permit. The detailed designs live in the M1.5
> RFC set — [RFC-0003](../../docs/rfc/0003-bitemporal-event-model.md),
> [RFC-0004](../../docs/rfc/0004-provenance-and-reference-data.md) and
> [RFC-0005](../../docs/rfc/0005-event-taxonomy-and-schema-evolution.md) are
> the relevant ones — all *Accepted*, recorded by ADR-0009, ADR-0010 and
> ADR-0011.
>
> One correction this document predates: what enters from a provider is usually
> an **Observation** ("the statement says the holding is 100 shares"), not an
> **Occurrence** ("a trade settled"). RFC-0005 makes that distinction
> load-bearing. Deriving occurrences from observations is a domain computation
> with its own provenance, never an ingestion shortcut.

## The intended pipeline

```
External source
      ↓                     ← private repository (connector)
Raw capture + provenance stamp
      ↓                     ← parser, versioned
Provider representation
      ↓                     ← normalisation at the boundary
Canonical Financial Model
      ↓                     ← deterministic business rules
Ledger events (immutable, append-only)
      ↓                     ← projection, rebuildable
Materialised views
      ↓
Reports · APIs · Knowledge Graph · LLM explanation
```

Two directions matter more than the boxes:

- **Everything above the ledger is a funnel.** Provider-specific concepts are
  eliminated before the canonical model, not after. A broker field name must
  never reach a business rule (Constitution §3).
- **Everything below the ledger is derived.** Views, reports, the graph and any
  LLM output are projections. None can be a source of truth, and all must be
  reconstructible from the ledger alone.

## Provenance is stamped at capture, never later

Provenance attaches where data enters and travels with it through every
transformation. No computation may lose it (Constitution §6).

Intended fields — universality is an open M1.5 question, and the honest position
is that optional provenance becomes absent provenance:

| Field | Meaning |
|-------|---------|
| source | where it came from |
| collection timestamp | when FDOS fetched it |
| effective timestamp | when the fact became true |
| knowledge timestamp | when FDOS learned it |
| parser version | which code interpreted it |
| transformation history | what was applied since capture |
| calculation method | how a derived value was produced |
| confidence | how much to trust it |

## The ledger boundary

The ledger is the only writable store of truth, and it is append-only. There is
no update and no delete. A correction is a new event referencing what it
corrects.

This is what makes a 2031 regeneration of a 2026 report possible: the inputs
still exist, unchanged, alongside everything learned since.

**Reference data is part of the input.** An FX rate or holiday calendar used by
a calculation must be captured by version, not by value-at-the-time. This cannot
be retrofitted.

## Where LLMs attach

At the end, and only there. Models read derived views and explain them. There is
no path from a model output back into the ledger, and M4 makes that structural
rather than conventional: model outputs carry a distinct type the write path
cannot accept.

## Reproducibility test

The property every part of this flow must satisfy:

> Given the same ledger and the same versioned reference datasets, regenerating
> any report at any future date produces a byte-identical result.

If a proposed design cannot satisfy that, the design is wrong — not the
requirement.
