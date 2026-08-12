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

> **Decided by [ADR-0010](adr/0010-provenance-envelope-reference-versioning.md)
> (RFC-0004, Accepted).** The founding text qualified this with "whenever
> applicable". Optional provenance becomes absent provenance.
>
> M4 delivered the wire half: every ledger fact carries an `Envelope`, checked
> by `make proto-check`. It could not deliver the rest — **proto3 has no
> `required`**, so a schema cannot make an envelope-less fact unrepresentable.
> That guarantee belongs to the Go kernel constructors at M6, and the table in
> §15 says rung 3 rather than claiming otherwise.

## 7. Temporal Modeling

FDOS adopts bitemporal modeling. Always distinguish when a fact became true from
when FDOS learned about it. Historical analysis must never introduce look-ahead
bias.

> **Decided by [ADR-0009](adr/0009-universal-bitemporality.md) (RFC-0003,
> Accepted).** The founding text qualified this with "whenever appropriate".
> Bitemporality is universal, with no default as-of on any query — the
> mechanism that makes look-ahead bias structurally impossible rather than
> merely discouraged.

## 8. Explainability

Every financial insight is explainable. Every recommendation exposes its inputs,
deterministic calculations, assumptions, provenance and confidence.

AI improves communication. Never financial truth.

> **Still the weakest principle, and the gap is now precise.**
> [ADR-0012](adr/0012-explained-return-type.md) (RFC-0006, Accepted) makes the
> computation trace part of every calculation's return type, so a calculation
> that does not explain does not compile.
>
> M4 put the derivation record on the wire (`ExplainedMoney`,
> `DerivationRecord`), which is the substrate. `Explained[T]` itself is a Go
> generic with combinators and arrives with the kernel at **M6** — until then
> nothing enforces that a calculation returns one. Rung 6 → 1 remains the
> largest single climb available, and it is still unmade.

## 9. Reproducibility

Every report is reproducible years later using the same ledger and reference
datasets. Reproducibility takes precedence over convenience.

> **Decided by [ADR-0010](adr/0010-provenance-envelope-reference-versioning.md)
> (RFC-0004, Accepted).** Reference datasets are versioned data. Reproducing a
> 2026 report in 2031 requires the 2026 reference data, not today's. This is
> not retrofittable: if the canonical model does not carry a reference-dataset
> version from the first event, historical reproducibility is permanently lost.
> The decision adds a third leg most systems forget — the *code* must be pinned
> too, alongside the ledger and the reference data.

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

Recorded as of M4. This table is the repository's honest self-assessment and is
expected to improve every milestone — including downward, when a milestone shows
an earlier target was not achievable.

| # | Principle | Mechanism today | Rung | Target |
|---|-----------|-----------------|------|--------|
| 1 | Financial Truth | `Position` is a projection return value with no persist path (`domain.ProjectPosition`) | 1 | 1 ✅ |
| 2 | Deterministic Engineering | `nondet`, `nofloat` analysers, now covering the kernel (`make analyze`) | 2 | 2 ✅ |
| 3 | Canonical Model First | `impurity`, `layering` analysers (`make analyze`) | 2 | 2 ✅ |
| 4 | Immutable Ledger | `Stream` has Append and no Update or Delete; the store has no whole-stream write to refuse one with (ADR-0034) | 1 | 1 ✅ |
| 5 | Event Sourcing | every read is a fold over facts; there is no state to read from | 1 | 1 ✅ |
| 6 | Provenance | `NewEnvelope` requires it and `Provenance` has no incomplete constructor | 1 | 1 ✅ |
| 7 | Temporal Modeling | both axes required; knowledge time never a caller's parameter; no query without an `AsOf` | 1 | 1 ✅ |
| 8 | Explainability | `ProjectPosition` returns `Explained[Position]`; combinators build the trace | 1 | 1 ✅ |
| 9 | Reproducibility | double-build diff, tidy and toolchain pins, SHA-pinned actions and digest-pinned downloaded artifacts (`make repro-check`, `tidy-check`, `toolchain-check`, `action-pinning-check`, `toolchain-checksum-check`) | 3 | 3 ✅ |
| 10 | Domain Before Infrastructure | `impurity`, `layering` analysers + directory contracts | 2 | 2 ✅ |
| 11 | Contracts Over Implementations | `layering`, `buf breaking` against main, and wire/domain round-trip conformance (`make proto-check`, `make test`) | 3 | 3 ✅ |
| 12 | Knowledge Graph Strategy | — | 6 | 5 |
| 13 | Open Core | `GOWORK=off` everywhere, published-module consumer proof, branch and tag rulesets (`make consumer-check`) | 3 | 3 ✅ |
| 14 | Engineering Culture | ADR and RFC logs, ADR immutability against git history, enforcement-table coverage, documentation references, agent prompt contracts, secret and vulnerability scans (`make adr-check`, `adr-immutability-check`, `rfc-check`, `constitution-check`, `context-check`, `agent-contract-check`, `secrets-check`, `vuln-check`) | 3 | 3 ✅ |

Six principles reached their target at M2. M3 raised no rung numbers — it made
the existing mechanisms real. Every check now runs automatically on every pull
request, ADR immutability moved from review to a git-history check, and every
build input is pinned by digest.

The seven principles still at rung 6 need types that do not exist yet: the
canonical model and the ledger. They climb at M4 and M6, and until then the
honest reading is that they are unenforced.

§8 (Explainability) remains the weakest. ADR-0012 decided the mechanism; nothing
implements it yet.

M2.5 added the same kind of mechanism to the knowledge base: documentation that
names a `make` target, a script, an ADR or a link now has to be describing
something that exists (`make context-check`), and agent playbooks declare
obligations that can be checked as data (`make agent-contract-check`).

M6 moved six principles to rung 1 — the type system — which is where the
Constitution has been pointing since M0. §1 has no position type to store, §4
has a Stream with no Update, §6 has no constructor that omits provenance, §7 has
no query without an `AsOf`, and §8 has a projection that returns
`Explained[Position]` or nothing at all.

Seven principles were at rung 6 after M4. Two remain: §12 (Knowledge Graph,
target rung 5 — it is a projection strategy, not an invariant) and nothing else.

M6 also found that the analysers did not cover `libs/kernel` at all — the one
package holding `Money`, where the float ban matters most, was unchecked for
four milestones (ADR-0021). Fixed, and the gap is recorded rather than quietly
closed: a rule is only worth its coverage.

M4 moved §11 to its target: `buf breaking` against `main` means an incompatible
contract change fails the build. It also **corrected two targets downward**. §6
and §7 were recorded as reaching rung 1 at M4; proto3 has no `required`, so a
schema cannot make an omitted envelope unrepresentable. They reach rung 3 here
and rung 1 with the Go kernel constructors at M6. Recording that honestly is
the point of the table — a self-assessment that only ever improves is not an
assessment.

Two rules remain enforced by nothing:

**Branch protection, required checks and the merge queue are GitHub settings,
not files.** They cannot be enforced from this repository.
`docs/branch-protection.md` records the intended configuration and states
plainly that it is a checklist, not a mechanism.

**Whether an agent honours its prompt contract** cannot be verified from
outside. The `must_read` / `must_not` / `evidence` fields make the obligation
explicit and reviewable; they do not enforce it, and a green
`agent-contract-check` is not evidence of compliance (ADR-0015).

`make constitution-check` asserts that this table lists every principle above,
by number and by name, in both directions. The table cannot silently fall behind
the Constitution it measures — which would turn the one artifact tracking
architectural erosion into a source of false confidence.
