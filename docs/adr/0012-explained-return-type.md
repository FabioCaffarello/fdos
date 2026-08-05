---
id: ADR-0012
title: Domain calculations producing financial values return Explained[T]
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0012 — Domain calculations producing financial values return Explained[T]

Records the acceptance of [RFC-0006](../rfc/0006-explainability-as-a-return-type.md).

## Context

Explainability is the weakest principle in the system — the only one
Constitution §15 records with no mechanism above documentation. The default
outcome is well understood: explanation gets reconstructed at the presentation
layer, drifts from the actual computation, and is eventually wrong in a way
nobody detects. An explanation that does not come from the computation is a
plausible story about one.

This matters more here because Constitution §2 permits LLMs to communicate
insight, and an LLM given a number will explain it fluently and unfalsifiably.
The only defence is that the explanation is generated data, not generated
prose.

Serves Constitution §8 (explainability) and §2 (LLMs never source truth).

## Decision

Every domain calculation that produces a financial value returns
`Explained[T] = {value, trace}`, never bare `T`. The trace is a reference to
the content-addressed `DerivationRecord` ADR-0010 already requires — inputs,
versioned method, parameters including every `RoundingContext` (ADR-0008),
reference-dataset versions, propagated ordinal confidence. That is exactly the
list Constitution §8 demands, by design.

Composition goes through combinators (`Map`, `Combine2`, `Fold`) that construct
the derivation record as a side effect of composing, so a trace cannot be
dropped by manual threading. Content addressing keeps a thousand-input fold a
DAG, not a tree.

A trace is data. Rendering it is a presentation concern in `app` or `adapters`.
An LLM may render a trace into prose; it may not produce one — no constructor
accepts model output as a `DerivationRecord`.

Scope: domain calculations producing financial values. Not predicates, lookups,
parsing, or anything in `app`/`adapters` — `Explained[bool]` for a validity
check is ceremony, not safety.

## Consequences

### Positive

- "Why is this number what it is" is answered by construction, and the answer
  is independently verifiable data an LLM can only re-render.
- When an input is retracted, everything derived from it is mechanically
  invalidatable.
- Constitution §8 climbs from rung 6 to rung 1 — the largest single climb
  available in the enforcement table.

### Negative

- The most invasive decision in the M1.5 set: every domain calculation
  signature changes, and composition requires combinators rather than plain
  expressions. Explicitly the decision most likely to be revised after contact
  with real code in M6; if combinator ergonomics prove intolerable, the
  re-execution alternative below is the recorded fallback.
- The rung-1 claim is conditional: "calculations producing financial values"
  needs a precise, checkable definition, or the rule degrades into judgement
  and the claim is false. That definition is owed no later than the M2
  analyser.
- Trace storage volume under a real workload is unknown.

### Enforcement

Today: rung 5 — there is no code. From M2: rung 1 (return type; combinators
construct the record; no constructor accepts model output; rounding contexts
recorded because ADR-0008 requires them as parameters). From M6: rung 3,
property test — shared with ADR-0010 — that every trace reaches primary
sources.

## Alternatives considered

- **Trace as a collected effect (recorder down the stack)** — a hidden mutable
  parameter, breaking domain purity, and optional in practice: a calculation
  that forgets to record still compiles.
- **Reconstruct the trace by re-execution under an instrumented interpreter** —
  zero cost on the normal path, but requires expressing calculations as data in
  a real interpreter with its own semantics and bugs, forfeiting Go's type
  checking for the most critical code. Recorded as the fallback if combinator
  ergonomics fail.
- **Explanation at the presentation layer** — a story about a computation, not
  the computation.
- **Keep §8 at rung 6** — "we intend to explain" is not an answer to an
  auditor.

Full exploration in RFC-0006.

## Notes

Open, deliberately: the precise, checkable scope boundary (owed with the M2
analyser); whether combinators suffice or real use demands code generation —
deciding before writing a domain calculation would be guessing; trace volume
and whether depth-pruning is compatible with Constitution §6.
