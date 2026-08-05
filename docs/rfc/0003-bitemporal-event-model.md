---
id: RFC-0003
title: Bitemporal event model — effective time, knowledge time and corrections
status: Accepted
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0003 — Bitemporal event model

## Summary

Proposes that bitemporality is **universal** in the canonical model, defines the
two time axes precisely, specifies deterministic ordering, and makes look-ahead
bias structurally impossible rather than merely discouraged.

## Motivation

Constitution §7 currently says FDOS adopts bitemporal modelling "whenever
appropriate". That is not a specification. Two systems can both satisfy it and be
incompatible.

The failure this prevents is **look-ahead bias**: computing a 2024 report using
information that arrived in 2025. A backtest contaminated this way reports
returns that were never achievable. The contamination is invisible — the numbers
look plausible — and it is the single most common way financial analysis
software produces confidently wrong answers.

**Not retrofittable.** If events do not carry knowledge time from the first
write, there is no way to reconstruct what was known when. The information was
never captured.

## Design

### Two axes, defined precisely

| Axis | Meaning | Assigned by | Mutable |
|------|---------|-------------|---------|
| **Effective time** | When the fact was true in the world | The source, or the domain | No |
| **Knowledge time** | When FDOS could first have acted on it | FDOS, at append | No |

Knowledge time is machine-assigned and monotonic per stream. It is not "when the
source published" — that is provenance (RFC-0004) — and it is not backdatable.
Allowing knowledge time to be set by a caller would reintroduce exactly the
contamination this design prevents.

### Effective time is an interval

```
EffectiveInterval := { from: Instant, to: Instant | Open }
```

An instantaneous fact is the degenerate case where `to = from`. Using an interval
uniformly avoids two representations for one concept.

Intervals are half-open `[from, to)` so that adjacent intervals compose without
gaps or overlaps.

### Universal, not scoped

Every canonical fact carries both axes. No exceptions.

The argument for scoping — that many facts have `effective == knowledge` and the
second axis is noise — is real but loses. Scoping creates two classes of fact,
and then every query, every projection and every join must know which class it
is handling. That knowledge inevitably becomes implicit, and the first place it
is assumed wrongly is a silent correctness bug.

Where the axes coincide, the cost is one redundant timestamp. That is a cheap
price for a uniform query model.

### Queries have no default as-of

A projection is a function of the ledger **and both temporal coordinates**:

```
Project(stream, asOfEffective, asOfKnowledge) -> Explained[View]
```

There is **no default** for either. A query that omits them does not compile.

This is the mechanism that makes look-ahead bias structural. The common failure
is a default of "now" for knowledge time: it is invisible, it is almost always
what a developer wants interactively, and it is almost always wrong in an
analysis. Removing the default makes the decision explicit at every call site.

### Deterministic ordering

Two facts can share both an effective time and a knowledge time. Reproducibility
(Constitution §9) requires a total order regardless.

```
Order := (effective_from, knowledge_time, stream_sequence)
```

`stream_sequence` is a monotonic integer assigned at append within a
`LedgerStream` (RFC-0001). It is a tiebreaker only, never semantic, and no
business rule may read it.

Cross-stream ordering is **not** globally defined. Facts in different streams
have no inherent order, and a projection requiring one must state its own
deterministic rule. Inventing a global order would imply a coordination
guarantee FDOS does not have.

### Corrections

A correction is a new fact, never a mutation:

```
correction := {
    effective:  <same interval as the corrected fact>
    knowledge:  <now — necessarily later>
    corrects:   FactRef
    reason:     CorrectionReason
}
```

The corrected fact remains readable. Asking the ledger as of a knowledge time
before the correction returns the original — which is precisely what "what did
we believe on date D" means, and precisely what an audit requires.

**Retraction** is distinct from correction: a retraction asserts the original
fact should never have been recorded, rather than that it was recorded with a
wrong value. Both are facts; neither deletes anything.

### What this forbids

| Forbidden | Why |
|-----------|-----|
| Caller-supplied knowledge time | Reintroduces backdating, defeats the whole model |
| `time.Now()` in domain code | Non-deterministic; knowledge time is assigned at the boundary |
| Projection without explicit as-of | The default is where look-ahead bias enters |
| Mutating or deleting a fact | Destroys the knowledge-time axis |
| Business rules reading `stream_sequence` | Couples semantics to an implementation tiebreaker |

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| Both axes always present | 1 | Envelope type has no constructor omitting them |
| Knowledge time not caller-supplied | 1 | Not a parameter of any public append operation |
| No projection without as-of | 1 | Required parameters, no variadic default |
| No `time.Now()` in domain | 2 | `go/analysis` pass (M2) |
| Total order is deterministic | 3 | Property test: shuffled append order with fixed timestamps yields identical projections |
| No look-ahead | 3 | Property test: a projection at knowledge time K is unchanged by appending any fact with knowledge time > K |

The last property is the one that matters most, and it is mechanically testable.

## Alternatives

**Uni-temporal (effective time only).** Much simpler and sufficient for
current-state questions. Rejected: it makes "what did we believe on date D"
unanswerable, which fails Constitution §9 and makes honest backtesting
impossible.

**Scoped bitemporality — bitemporal only where it matters.** Rejected above: two
classes of fact, with the distinction going implicit and then being assumed
wrongly.

**Tri-temporal (effective, knowledge, system/write time).** Genuinely useful for
distinguishing "when we learned" from "when we durably recorded", and standard
in some regulatory contexts. Rejected for now on complexity grounds: FDOS
controls its own writes, so the two coincide except during recovery. Revisit if
replication makes the distinction observable — the RFC that does so should note
this is a hard extension, not an additive one.

**Snapshot-per-day.** Simple, and how many systems approximate this. Rejected:
it loses intra-day knowledge ordering and grows storage without bound.

## Prior art

Snodgrass's work on temporal databases established valid/transaction time as the
standard decomposition; SQL:2011 standardised it. Datomic makes transaction time
non-backdatable for the same reason proposed here. Quantitative finance
independently arrived at "point-in-time" databases after backtest results proved
irreproducible — the same lesson, learned expensively.

## Open questions

- Timezone and precision of `Instant`. Proposal: UTC, nanosecond, but the
  precision choice interacts with equality and therefore with the ordering
  tiebreaker.
- Should effective intervals support open-start (`from = -∞`) for facts believed
  always to have been true?
- How are corrections to a *correction* presented? The chain is well-defined but
  the projection semantics need stating.
- Reference data is itself bitemporal (RFC-0004). Does a projection carry one
  as-of pair, or one per dataset?

## Consequences

**Easier:** honest backtesting; audit reconstruction; supporting late-arriving
and out-of-order data without special cases.

**Harder:** every query grows two required parameters. Storage grows with
correction history rather than staying flat.

**Impossible:** look-ahead bias entering through an unexamined default; rewriting
history.
