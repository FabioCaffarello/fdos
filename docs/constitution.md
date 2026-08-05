---
title: FDOS Engineering Constitution
version: 1.0.0
status: Ratified
ratified: 2026-08-04
---

# FDOS Engineering Constitution

This document states the principles that govern the Financial Data Operating
System. It is the highest authority in this repository: an ADR may implement a
principle, refine it, or record how it is enforced, but no ADR may contradict
one. Contradicting a principle requires amending the Constitution first.

## Amendment procedure

1. Amendments are proposed as an RFC (`docs/rfc/`), never as a direct edit.
2. An amendment must state which principle changes, why, and what breaks.
3. On acceptance, the RFC is merged, this document's `version` is incremented
   under semantic versioning, and an ADR records the amendment.
4. Prior versions are recoverable from git history. This document is versioned,
   not immutable — but it changes only through the procedure above.

Semantics: **major** removes or reverses a principle; **minor** adds one;
**patch** clarifies wording without changing meaning.

---

## 1. Financial Truth

FDOS never stores financial state. FDOS stores immutable financial facts.

Every position, allocation, balance, performance metric and recommendation must
always be reproducible from immutable ledger events. State is always derived.
Facts are never overwritten.

## 2. Deterministic Engineering

All business rules are deterministic. Every financial calculation is
reproducible. Every financial report is reproducible. Every AI-generated insight
originates from deterministic computations.

LLMs may explain, summarise, prioritise and communicate. LLMs must never become
the source of financial truth.

## 3. Canonical Model First

No connector, importer, parser or provider may introduce domain concepts
directly into FDOS. Every external representation is first normalised into the
Canonical Financial Model. Business rules operate exclusively on canonical
models. Provider-specific concepts must never leak into the domain.

## 4. Immutable Ledger

The ledger is append-only. History is never rewritten. Corrections are
represented by new events. The ledger is the single source of financial truth.

## 5. Event Sourcing

Every domain state is derivable from immutable events. Materialised views exist
only for performance. Business truth always comes from the event stream.

## 6. Provenance

Every financial datum preserves provenance: source, collection timestamp,
effective timestamp, knowledge timestamp, parser version, transformation
history, calculation method, confidence.

No computation may lose provenance.

> **Open question, to be closed by RFC in M1.5.** The founding text qualified
> this with "whenever applicable". Optional provenance becomes absent
> provenance. The RFC must decide whether the provenance envelope is universal
> or explicitly scoped, and the answer must be structural — a required field,
> not a convention.

## 7. Temporal Modeling

FDOS adopts bitemporal modeling. Always distinguish when a fact became true from
when FDOS learned about it. Historical analysis must never introduce look-ahead
bias.

> **Open question, to be closed by RFC in M1.5.** The founding text qualified
> this with "whenever appropriate". Bitemporality must be either universal in the
> canonical model or explicitly scoped per aggregate type — decided once, before
> the first event schema exists.

## 8. Explainability

Every financial insight is explainable. Every recommendation exposes its inputs,
deterministic calculations, assumptions, provenance and confidence.

AI improves communication. Never financial truth.

> **Weakest principle in the system.** As of version 1.0.0 this is the only
> principle with no enforcement mechanism above "documentation". The candidate
> remedy — making the computation trace part of every calculation's return type,
> so that a calculation which does not explain does not compile — is an M1.5 RFC.

## 9. Reproducibility

Every report is reproducible years later using the same ledger and reference
datasets. Reproducibility takes precedence over convenience.

> Reference datasets are versioned data. Reproducing a 2026 report in 2031
> requires the 2026 reference data, not today's. This is not retrofittable: if
> the canonical model does not carry a reference-dataset version from the first
> event, historical reproducibility is permanently lost. M1.5 RFC.

## 10. Domain Before Infrastructure

Infrastructure exists to serve the domain. The domain must never depend on
browser automation, storage, message brokers, databases, external APIs or
frameworks. Infrastructure is always replaceable.

## 11. Contracts Over Implementations

Subsystems communicate through explicit contracts. Contracts are versioned,
documented and tested. Implementations remain replaceable.

## 12. Knowledge Graph Strategy

The Knowledge Graph is a strategic capability, not an initial requirement. Every
canonical model, identifier strategy and event schema must naturally support
graph projection. The graph is never the source of truth; it is a semantic
projection over immutable financial facts.

## 13. Open Core

FDOS follows an Open Core architecture.

Public: engineering platform, canonical models, ledger, SDKs, APIs,
documentation, knowledge graph, testing infrastructure.

Private: authenticated providers, browser connectors, institution-specific
plugins, provider automation, private parsers.

Public contracts are published as versioned modules. Private implementations may
be coupled to the workspace for development convenience, but must depend only on
published contract versions — proven continuously, not assumed (ADR-0004).

## 14. Engineering Culture

Every implementation improves the repository. Prefer deleting complexity to
adding abstraction. Documentation is production code. Architecture is reviewed
before implementation. Benchmarks are mandatory for performance-sensitive
components. Security, observability and documentation are part of development,
not phases after it.

---

## 15. The Enforcement Ladder

A principle stated in prose is a wish. This section is the mechanism that turns
the preceding fourteen into engineering.

Every principle is enforced at the highest rung that can carry it:

| Rung | Mechanism | Fails at |
|------|-----------|----------|
| 1 | **Type system** — the violation cannot be expressed | authoring time |
| 2 | **Static analysis** — compiler pass, custom analyser, import boundary | build |
| 3 | **CI** — test, fitness function, reproducibility gate | pull request |
| 4 | **Automated review** — agent-assisted review with defined contracts | review |
| 5 | **Documentation** — convention, playbook, checklist | never, automatically |
| 6 | **Human discipline** | — |

Human discipline is the last line of defence, never the first. Every principle
carries an obligation to climb: when a mechanism one rung higher becomes
feasible, adopting it is not optional.

### Current position

Recorded as of Constitution 1.0.0 (M0). This table is the repository's honest
self-assessment and is expected to improve every milestone.

| # | Principle | Mechanism today | Rung | Target |
|---|-----------|-----------------|------|--------|
| 1 | Financial Truth | — | 6 | 1 (M6) |
| 2 | Deterministic Engineering | — | 6 | 2 (M2) |
| 3 | Canonical Model First | — | 6 | 2 (M2) |
| 4 | Immutable Ledger | — | 6 | 1 (M6) |
| 5 | Event Sourcing | — | 6 | 1 (M6) |
| 6 | Provenance | — | 6 | 1 (M4) |
| 7 | Temporal Modeling | — | 6 | 1 (M4) |
| 8 | Explainability | — | 6 | 1 (M4, unproven) |
| 9 | Reproducibility | — | 6 | 3 (M2) |
| 10 | Domain Before Infrastructure | directory contracts (`make contracts-check`) | 3 | 2 (M2) |
| 11 | Contracts Over Implementations | — | 6 | 3 (M4) |
| 12 | Knowledge Graph Strategy | — | 6 | 5 |
| 13 | Open Core | — | 6 | 3 (M5) |
| 14 | Engineering Culture | ADR log (`make adr-check`) | 3 | 3 |

At M0 almost everything sits at rung 6. That is the accurate reading of a
repository whose first commit this is, and stating it plainly is the point: the
table exists to make the gap visible and to make closing it measurable.
