---
type: doc
name: glossary
description: FDOS terminology — governance terms are binding, domain terms are provisional
category: glossary
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Glossary

Each FDOS domain owns its own ubiquitous language, and no implementation detail
leaks across domains. This file holds only the cross-cutting vocabulary.

> **Domain terms below are provisional.** The M1.5 RFC set (RFC-0001 …
> RFC-0006) now proposes definitions for most of them, but *Proposed is not
> decided*: nothing binds until an RFC is accepted and an ADR records it. Terms
> marked *(provisional)* describe intent. Do not implement against them.

## Governance — binding

**Constitution** — `docs/constitution.md`. The fourteen principles governing
FDOS. Highest authority in the repository. Amended only through an RFC, with a
version bump and an ADR.

**ADR** — Architecture Decision Record. Append-only, immutable. Corrections are
new ADRs that supersede old ones.

**RFC** — Request for Comments. For decisions requiring design exploration
before they can be made. Produces ADRs.

**Enforcement ladder** — the six-rung hierarchy from ADR-0005: type system >
static analysis > CI > automated review > documentation > human discipline.
Every principle is enforced at the highest feasible rung.

**Rung** — a principle's current position on that ladder. Recorded in
Constitution §15. Rung 6 means "unenforced".

**Fitness function** — an automated check asserting an architectural property
holds. Lives in `scripts/`, invoked through `make`.

**Directory contract** — the `purpose`/`owner`/`allowed`/`forbidden` front
matter in each directory's `README.md`. Binding, and from M2 the source of the
import-boundary configuration.

**Milestone** — M0 … M6. Note the ordering: M1 → M1.5 → M2 → M3 → M2.5 → M3.5 →
M4 → M5 → M6. M2.5 follows M3 deliberately, so AI-assisted work is only enabled
once CI enforces the gates automatically.

## Domain — provisional

**Financial fact** *(provisional)* — an immutable statement that something was
true. Never overwritten; corrected only by a new fact.

**Financial state** *(provisional)* — a derived view. FDOS never stores it.

**Ledger** *(provisional)* — the append-only event log. Single source of
financial truth.

**Canonical Financial Model** *(provisional)* — the internal representation all
external data is normalised into before any business rule sees it.

**Provenance** *(provisional)* — the origin and history travelling with every
datum: source, collection/effective/knowledge timestamps, parser version,
transformation history, calculation method, confidence.

**Bitemporality** *(provisional)* — modelling both when a fact became true and
when FDOS learned of it. The distinction that prevents look-ahead bias.

**Effective time** *(provisional)* — when the fact became true in the world.

**Knowledge time** *(provisional)* — when FDOS learned it. Also called decision
time.

**Look-ahead bias** *(provisional)* — using information in a historical analysis
that was not known at the time being analysed. Bitemporality exists to make this
structurally impossible.

**Reference data** *(provisional)* — versioned external data a calculation
depends on: FX rates, holiday calendars, issuer classifications. Reproducing a
2026 report in 2031 needs the 2026 versions. **Not retrofittable** — if the
model does not carry a reference-data version from the first event, historical
reproducibility is permanently lost.

**Materialised view** *(provisional)* — a derived projection existing only for
performance. Never a source of truth; must be byte-reproducible from the ledger.

**Explainability** *(provisional)* — the obligation that every insight exposes
inputs, calculations, assumptions, provenance and confidence. Currently the
weakest principle in FDOS, with no mechanism above documentation. The candidate
remedy — making the computation trace part of every calculation's return type —
is an M1.5 RFC.

## Vocabulary proposed by the M1.5 RFCs

All *(provisional)* until the relevant RFC is accepted.

**EntityID** *(RFC-0001)* — opaque internal identifier, derived deterministically
at first observation. Never parsed, never a natural key.

**Identifier assertion** *(RFC-0001)* — a sourced, timestamped claim that an
entity is identified by an external scheme value. An instrument does not *have*
an ISIN; a source *asserted* one.

**Aggregate** *(RFC-0001)* — a consistency boundary. FDOS proposes four:
`Instrument`, `Party`, `Account`, `LedgerStream`. `Position`, `Balance` and
`Performance` are explicitly **not** aggregates — they are projections.

**Money** *(RFC-0002)* — `{decimal amount, currency}`. Cross-currency arithmetic
is a type error, not a runtime check.

**RoundingContext** *(RFC-0002)* — precision plus rounding mode, required by
every division. There is deliberately **no default**: unexamined default
rounding is where most financial calculation bugs live.

**Occurrence** *(RFC-0005)* — a fact asserting something happened in the world.
`TradeSettled`, `DividendPaid`.

**Observation** *(RFC-0005)* — a fact asserting FDOS was *told* something is so.
`HoldingObserved`. A broker statement is an Observation; recording it as an
Occurrence fabricates history that never happened.

**Upcaster** *(RFC-0005)* — a pure, versioned `vN → vN+1` function applied on
read. Stored events are never migrated; the upcaster chain is pinned per report.

**DerivationRecord** *(RFC-0004)* — content-addressed record of a computation:
inputs, method, parameters, reference bindings. Referenced by hash rather than
inlined, so lineage is a DAG and not a quadratic blowup.

**ReferenceBinding** *(RFC-0004)* — `{dataset, version}`. What a calculation
consumed, pinned by version rather than by value.

**Confidence** *(RFC-0004)* — ordinal, not numeric: `Asserted | Derived |
Estimated | Inferred | Disputed`. A number would invite arithmetic that has no
defensible meaning.

**`Explained[T]`** *(RFC-0006)* — a value paired with its computation trace. The
proposed mechanism that moves explainability from rung 6 to rung 1.

## Open Core

**Public core** — this repository. Apache-2.0.

**Private repositories** — authenticated providers, browser connectors,
institution-specific plugins. Separately licensed; depend on the public core
only through published, versioned contract modules.

**`GOWORK=off`** — the CI setting that makes the open-core boundary real. Without
it, module coupling resolves through local paths and the boundary silently stops
being verified.
