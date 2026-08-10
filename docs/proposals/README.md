---
title: Architectural audit proposals — 2026-08-07
status: Provisional — proposal package from the 2026-08-07 architectural audit
date: 2026-08-07
---

# Audit proposal package

> **Status: provisional, all of it.** Every document in this directory is a
> proposal produced by the 2026-08-07 architectural audit. None is an accepted
> decision. Per [ADR-0000](../adr/0000-record-architecture-decisions.md) and
> `AGENTS.md`, nothing may be implemented against any content here until an RFC
> and ADR accept the relevant part. Where a document conflicts with an accepted
> ADR, **the ADR governs** until superseded through the normal procedure.

## Why this directory exists

The audit was asked for a set of architecture documents "if missing". Several
of the requested documents exist in authoritative form and are **not**
duplicated here — duplicating them would create the drifted second copy this
repository's own `CLAUDE.md` warns about:

| Requested | Authoritative location | This package adds |
|---|---|---|
| Constitution | [`docs/constitution.md`](../constitution.md) (ratified v1.0.0) | [`Constitution-Amendments.md`](Constitution-Amendments.md) — candidate amendments only |
| Roadmap | [`docs/ecosystem/roadmap.md`](../ecosystem/roadmap.md) (M0–M12) | [`Roadmap.md`](Roadmap.md) — proposed continuation M12a–M19 |
| ADR strategy | ADR-0000 | [`adr-index.md`](adr-index.md) — navigable index (descriptive, regenerable) |
| RFC strategy | `docs/rfc/template.md` | [`rfc-index.md`](rfc-index.md) — navigable index (descriptive, regenerable) |

Documents that were genuinely missing are proposed here in full:

| Document | Contents |
|---|---|
| [`Vision.md`](Vision.md) | The Domain Vision that ADR-0013 cites as binding and that has never existed |
| [`Domain-Model.md`](Domain-Model.md) | Bounded contexts, aggregates, entities, value objects, commands, events, queries, projections, ubiquitous language |
| [`Canonical-Financial-Model.md`](Canonical-Financial-Model.md) | The complete canonical type system and target fact taxonomy |
| [`Ledger.md`](Ledger.md) | Target ledger architecture: admission, store, evolution, corrections, streams |
| [`Corporate-Actions.md`](Corporate-Actions.md) | The corporate-actions engine design |
| [`Financial-Engines.md`](Financial-Engines.md) | Deterministic engine architecture: portfolio, fixed income, risk; Credit Intelligence verdict |
| [`MCP.md`](MCP.md) | The MCP surface: capabilities, never raw data |
| [`Knowledge-Layer.md`](Knowledge-Layer.md) | Derivation store, explanation rendering, knowledge-graph verdict |
| [`Repository-Architecture.md`](Repository-Architecture.md) | Current vs target module topology |
| [`Engineering-Principles.md`](Engineering-Principles.md) | Operating principles the audit recommends adding to practice |
| [`audit-review.md`](audit-review.md) | The 32-area review verdicts, with pointers into the corpus and the audit |
| [`ecosystem-architecture-reconciliation.md`](ecosystem-architecture-reconciliation.md) | Reconciliation of the fdos and fdos-connectors architecture reviews: responsibility matrix, contract/provenance ownership, contradictions, decision routing (2026-08-08) |

## How to adopt any of this

One part at a time, through the repository's own machinery:

1. Open an issue for the part (per ADR-0032's register convention).
2. Write the RFC; the relevant proposal document here is source material, not
   the RFC itself.
3. Record acceptance as an ADR; supersede any ADR the change reverses.
4. Only then implement.

The audit's defect list has a stricter rule: the Phase-0 integrity items
(storage encodings, hash and identity-seed injectivity, the exact-context
traps) should be sequenced **before** any new external consumer or binary
release, because each becomes permanently unfixable once real facts persist.

## Provenance

Produced by the Distinguished-Engineer audit of 2026-08-07 at commit
`bf371b7`, alongside the full findings report (execution-verified defects,
severity rankings, priority matrix) published separately as an artifact. All
audit probes were removed; the working tree contains only this directory's
additions.
