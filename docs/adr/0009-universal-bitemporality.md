---
id: ADR-0009
title: Every canonical fact is bitemporal and no query has a default as-of
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0009 — Every canonical fact is bitemporal and no query has a default as-of

Records the acceptance of [RFC-0003](../rfc/0003-bitemporal-event-model.md).

## Context

Constitution §7 adopts bitemporal modelling but the founding text did not
specify scope, and "whenever appropriate" is not a specification. The failure
at stake is look-ahead bias: a 2024 report computed with information that
arrived in 2025 reports returns that were never achievable, and the
contamination is invisible.

Not retrofittable: if events do not carry knowledge time from the first write,
"what was known when" was never captured and cannot be reconstructed.

Serves Constitution §7 (temporal modelling) and §9 (reproducibility).

## Decision

Bitemporality is universal. Every canonical fact carries both axes, no
exceptions:

- **Effective time** — when the fact was true in the world; a half-open
  interval `[from, to)`, an instantaneous fact being the degenerate case.
- **Knowledge time** — when FDOS could first have acted on it;
  machine-assigned at append, monotonic per stream, and **never
  caller-supplied**. A source's `published_at` is provenance (ADR-0010), not
  knowledge time.

Projections take both temporal coordinates as required parameters. **There is
no default as-of** — a default of "now" for knowledge time is where look-ahead
bias enters, so a query that omits either coordinate does not compile.

Ordering is totally defined within a stream as
`(effective_from, knowledge_time, stream_sequence)`; the sequence is a
tiebreaker only and no business rule may read it. Cross-stream ordering is
deliberately undefined; a projection needing one states its own deterministic
rule.

Corrections and retractions are new facts carrying references to what they
correct; nothing is mutated or deleted, and querying as of a knowledge time
before the correction returns the original — which is what an audit requires.

## Consequences

### Positive

- Look-ahead bias becomes structurally impossible rather than discouraged, and
  mechanically testable: a projection at knowledge time K is unchanged by
  appending any fact with knowledge time greater than K.
- Late-arriving and out-of-order data need no special cases.
- "What did we believe on date D" is always answerable.

### Negative

- Every query grows two required parameters, including interactive use where
  "now" is almost always what the caller wants. The friction is the mechanism;
  it is also permanent.
- Storage grows with correction history rather than staying flat.
- Where the axes coincide the second timestamp is redundant — accepted as the
  price of a uniform query model over two silently divergent classes of fact.

### Enforcement

Today: rung 5 — there is no code. From M2: rung 1 (envelope constructors cannot
omit either axis; knowledge time is not a parameter of any public append;
as-of parameters are required) and rung 2 (`go/analysis` ban on `time.Now()`
in domain packages). From M6: rung 3 property tests — shuffled append order
with fixed timestamps yields identical projections, and the no-look-ahead
property above, which is the one that matters most.

## Alternatives considered

- **Uni-temporal** — makes "what did we believe on date D" unanswerable;
  honest backtesting impossible.
- **Scoped bitemporality** — two classes of fact whose distinction goes
  implicit and is eventually assumed wrongly; a silent correctness bug.
- **Tri-temporal (adding system time)** — real in replicated regulatory
  contexts; rejected for now because FDOS controls its own writes and the axes
  coincide except during recovery. Revisiting is a hard extension, not an
  additive one.
- **Snapshot-per-day** — loses intra-day knowledge ordering, unbounded storage.

Full exploration in RFC-0003.

## Notes

Open, deliberately: `Instant` timezone and precision (proposal: UTC,
nanosecond — the choice interacts with the ordering tiebreaker); open-start
effective intervals; projection semantics of corrections to corrections;
whether a projection carries one as-of pair or one per reference dataset
(interacts with ADR-0010). To be settled by the M6 ledger slice.
