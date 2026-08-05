---
id: RFC-0006
title: Explainability as a return type
status: Proposed
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0006 — Explainability as a return type

## Summary

Proposes that every deterministic calculation returns its computation trace
alongside its result, so that a calculation which cannot explain itself does not
compile.

## Motivation

Explainability is the weakest principle in FDOS. Constitution §15 records it at
rung 6 — the only principle with no mechanism above documentation.

Constitution §8 requires every recommendation to expose its inputs, calculations,
assumptions, provenance and confidence. Today nothing enforces that, and the
default outcome is well understood: explanation is added at the presentation
layer, reconstructed by hand, drifts from the actual computation, and is
eventually wrong in a way nobody detects. An explanation that does not come from
the computation is not an explanation — it is a plausible story about one.

This matters more than usual here because Constitution §2 permits LLMs to
*communicate* insight. An LLM given a number and asked to explain it will
produce something fluent and unfalsifiable. The only defence is that the
explanation is generated data, not generated prose.

## Design

### The type

```
Explained[T] := {
    value: T
    trace: DerivationRef        # RFC-0004
}
```

Every domain calculation returns `Explained[T]`, never bare `T`. The trace is a
reference to a content-addressed derivation record — the structure RFC-0004
already defines for provenance, reused rather than duplicated.

This means explainability costs almost nothing structurally: the derivation
record must exist anyway to satisfy Constitution §6.

### Composition

The verbosity problem is real. Go has no monadic `do` notation, so naive
composition is painful:

```
a := Sum(xs)                       // Explained[Money]
b := Divide(a.value, n, ctx)       // caller must thread a.trace manually
```

Threading traces by hand guarantees they will be dropped. The proposal is a
small set of combinators that make the trace impossible to lose:

```
Map(e Explained[A], method MethodRef, f func(A) B) Explained[B]
Combine2(a Explained[A], b Explained[B], method MethodRef, f func(A,B) C) Explained[C]
Fold(xs []Explained[A], method MethodRef, f func(B,A) B, seed B) Explained[B]
```

Each combinator constructs a derivation record naming the method and referencing
the input traces. The trace is built as a side effect of composing, not as an
additional obligation on the author.

`Fold` is the important one: it is where a thousand-input calculation would
otherwise produce a thousand-entry inline trace. Content addressing (RFC-0004)
keeps this a DAG rather than a tree.

### What a trace contains

By construction, from RFC-0004's `DerivationRecord`:

- **inputs** — references to the facts consumed
- **method** — which calculation, versioned
- **parameters** — including the `RoundingContext` of every division (RFC-0002)
- **references** — reference dataset versions consumed
- **confidence** — ordinal, propagated from inputs

Which is exactly the list Constitution §8 demands. That is not a coincidence —
the list was the design target.

### Rendering, and the LLM boundary

A trace is data. Rendering it for a human is a presentation concern living in
`app` or `adapters`, never in `domain`.

An LLM may render a trace into prose. It may not *produce* a trace. The boundary
is enforced by type: an LLM adapter returns a presentation string, and no
constructor accepts a model output as a `DerivationRecord`.

This is what makes Constitution §2 mechanical rather than aspirational: the model
explains an explanation that already exists and is independently verifiable.

### Scope

`Explained[T]` applies to **domain calculations that produce financial values**.

It does not apply to predicates, lookups, parsing, or anything in `app` or
`adapters`. Applying it universally would produce ceremony without safety —
`Explained[bool]` for a validity check explains nothing anyone will read.

Drawing this line precisely is the main open question below.

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| Calculations return `Explained[T]` | 1 | Return type; a bare `Money` from a calculation does not satisfy the interface |
| Traces cannot be dropped in composition | 1 | Combinators construct the record; no manual threading |
| LLM output cannot become a trace | 1 | No constructor accepts a model output type |
| Rounding decisions appear in the trace | 1 | `RoundingContext` is a required parameter (RFC-0002) and is recorded |
| Every trace reaches primary sources | 3 | Property test, shared with RFC-0004 |

This moves Constitution §8 from rung 6 to rung 1 — the largest single climb
available anywhere in the current enforcement table.

## Alternatives

**Trace as a collected effect.** Pass a recorder down the call stack; calculations
append to it. Far less verbose, and familiar. Rejected: the recorder is a hidden
mutable parameter, which breaks the purity the domain layer exists to guarantee
(RFC-0003 forbids exactly this shape). It also makes the trace optional — a
calculation that forgets to record still compiles.

**Reconstruct the trace by re-execution.** Since the domain is pure and
deterministic, a calculation can be re-run under an instrumented interpreter to
recover its trace. Genuinely attractive: zero cost on the normal path.
Rejected: it requires calculations to be expressed as *data* — a small
deterministic expression language — rather than as Go. That is a real
interpreter, with its own semantics, versioning and bugs, and it forfeits Go's
type checking for the domain's most critical code. Worth reconsidering only if
the combinator ergonomics prove intolerable in practice; noted here so the option
is not lost.

**Explanation at the presentation layer.** Cheapest, and what most systems do.
Rejected on the motivation above: it is a story about a computation, not the
computation.

**Nothing — keep §8 at rung 6.** The honest baseline. Rejected because §8 is a
regulatory-facing obligation, and "we intend to explain" is not an answer to an
auditor.

## Prior art

Differentiable programming and build systems both solve this shape by making the
computation graph a first-class value. Spreadsheet auditing tools reconstruct
precedent chains after the fact and are unreliable for precisely the reason given
here. The combinator approach is the applicative-functor pattern, narrowed to
one concrete effect.

## Open questions

- **The scope boundary.** "Calculations producing financial values" needs a
  precise, checkable definition, or the rule degrades into judgement and the
  rung-1 claim becomes false.
- Do combinators suffice, or does real use demand code generation? Deciding
  before writing a domain calculation would be guessing.
- Trace storage volume. A high-frequency calculation produces many derivation
  records. Content addressing deduplicates identical ones, but the ratio is
  unknown until there is a real workload.
- Should `Explained[T]` be pruned for traces below some depth, and does pruning
  violate Constitution §6?

## Consequences

**Easier:** answering "why is this number what it is"; auditing; giving an LLM
something true to render; invalidating derived values when an input is retracted.

**Harder:** every domain calculation signature changes. Composition requires
combinators rather than plain expressions. This is the most invasive proposal in
the M1.5 set, and the one most likely to be revised after contact with real code.

**Impossible:** producing a financial value that cannot say how it was computed;
an LLM inventing a derivation.
