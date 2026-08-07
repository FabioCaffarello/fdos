---
id: ADR-0036
title: Knowledge time is assigned under the stream's write lock
status: Accepted
date: 2026-08-07
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0036 — Knowledge time is assigned under the stream's write lock

## Context

Records what [RFC-0015](../rfc/0015-the-submission-service-and-the-admission-race.md)
settled about the write path, in answer to the M11 gate
([PR #63](https://github.com/FabioCaffarello/fdos/pull/63)).

**The ledger refuses most of the traffic it was built to accept.** Measured on
`linux/amd64` in CI, 32 concurrent submissions of an identical valid claim to one
stream through `app.Ledger.AcceptHoldingClaim` and the SQLite store:

```
clock: 10000 readings -> 10000 distinct instants

round 1  concurrent/wall:     admitted=7 nonMono=25 stale=0
         concurrent/distinct: admitted=5 nonMono=27 stale=0
         sequential/wall:     admitted=32 nonMono=0 stale=0
round 3  concurrent/wall:     admitted=4 nonMono=28
         concurrent/distinct: admitted=3 nonMono=29
         sequential/wall:     admitted=32 nonMono=0
```

Three readings make this a decision rather than a bug report.

**It is not the clock.** The `distinct` case injects a clock handing every
caller an instant no other caller received. It is indistinguishable from the
wall-clock case. Linux resolves 10 000 readings to 10 000 distinct instants and
it does not help.

**It is not the platform.** The gate first measured this on `darwin/arm64`,
where a *sequential* loop also failed — one run admitted 4 of 32. That half was
real and was entirely darwin's microsecond wall clock: on Linux the sequential
case admits 32 of 32, every round. The concurrent failure survives both
platforms.

**It is the window between reading the clock and taking the write lock.**

```go
coordinates, err := temporal.Assign(cmd.Effective, l.clock.Now())   // (1)
…
ref, err := l.store.Append(ctx, cmd.Stream, Any(), envelope, …)     // (2)
```

Between (1) and (2) the caller queues for the store's single writer
(`SetMaxOpenConns(1)`, `_txlock=immediate`, both deliberate). A caller that read
at *t₁* and arrives after one that read at *t₂ > t₁* is refused — correctly.
[ADR-0009](0009-universal-bitemporality.md) requires knowledge time monotonic
per stream and [ADR-0034](0034-the-ledger-event-store.md) put the check in the
only place that can perform it.

### ADR-0034 predicted this and scoped it wrongly

It recorded, under negative consequences:

> **Clock skew becomes write unavailability.** With two processes and two
> clocks, a writer whose clock lags has a *legitimate* append rejected.

The prediction is right; the precondition is not. It needs **no second process,
no second clock and no skew.** One process, one clock and two goroutines
suffice, because the unserialised gap is between the clock read and the write
lock — not between two clocks.

That ADR is not edited (ADR-0000). The correction lives here, which is where a
reader following the link will find it.

Constitution §1 (Financial Truth), §4 (Immutable Ledger) and §7 (Temporal
Modeling) are what the store's check protects, and this decision does not weaken
it. What is at stake is availability of the truth path: E9's *"any third party
can ingest"* is not met by a path that refuses nine in ten honest facts.

## Decision

**FDOS assigns knowledge time under the stream's write lock.**

`app.Ledger` acquires a lock for the target stream, reads the clock, appends,
and releases. Every write use case passes through it — not only admission —
because the window is in the shape of the write path rather than in any one
caller.

```go
type Ledger struct {
    store  Store
    clock  Clock
    rules  identity.Ruleset
    writes *streamLocks
}
```

### The locks are sharded, not per-name

A fixed array of locks indexed by a hash of the stream name. Two unrelated
streams occasionally serialise; memory is constant.

**The alternative — a map with one lock per stream — is refused for a reason
that is not performance.** Stream names are producer-supplied and unbounded
([ADR-0030](0030-the-submission-shape.md)), so a per-name map is an allocation
primitive handed to a stranger: one entry per fabricated name, from a caller
that never has to succeed. Nothing bounds it, because nothing knows who may name
a stream — **that is D2**
([#64](https://github.com/FabioCaffarello/fdos/issues/64)), and it is open.

A bound that depends on an open question is not a bound. Sharding is bounded by
construction and its cost is throughput rather than correctness.

### It lives in `libs/ledger/app`, not in a composition root

- **`apps/README.md` forbids it.** *"Logic that cannot be tested without
  starting a process"* is a listed prohibition. The same mechanism in `app` is
  tested by starting goroutines.
- **A second composition root would have to reimplement it.** The conformance
  kit, a CLI, a future gRPC front end — and the one that forgot would lose the
  property silently, which is the failure mode this decision exists to remove.
- **`app` already owns the clock.** ADR-0034 put knowledge-time assignment in
  the application layer deliberately; the window is inside the space that
  decision created, so the close belongs there too.

### The store's check is unchanged, and acquires its real job

`ErrNonMonotonicKnowledge` stays exactly as it is. What changes is what it
*means*: it stops firing on ordinary single-process traffic and becomes the
cross-process guard ADR-0034 designed it to be. An operator who sees it now
learns something.

### What this does not fix

**Two processes.** The window closes inside one process. Two service instances
against one store reintroduce it exactly as ADR-0034 described. SQLite is
single-writer ([ADR-0035](0035-the-sqlite-driver-and-its-provenance-risk.md)),
nothing deploys two writers today, and ADR-0034 left the multi-writer question
open. It stays open. **No deployment story may imply otherwise.**

## Consequences

### Positive

- **Concurrent admission works.** The measured 3-to-9-of-32 becomes all of them,
  which is the difference between a ledger and a demonstration.
- **The fix is inherited by every caller**, present and future, because it is in
  the use case rather than in a handler.
- **`ErrNonMonotonicKnowledge` becomes informative.** A refusal now indicates
  genuine cross-process disorder rather than ordinary traffic, so it can be
  alerted on instead of filtered out.
- **Nothing recorded changes meaning.** Knowledge time remains exactly the
  instant the application read the clock. That is the property alternative E
  would have traded away and this decision does not.

### Negative

- **`app.Ledger` acquires shared mutable state.** It is the first in that type,
  and it is not injected — so a test that constructs one `Ledger` per goroutine
  will pass while exercising nothing. That trap is the reason the concurrency
  test must construct **one** `Ledger` and share it, and the reason this is
  called out here rather than left to a reviewer.
- **Per-stream write throughput acquires a ceiling**, and it is the *larger* of
  one append latency and **one tick of the platform's wall clock**. The second
  half is easy to miss and is not merely slowness: two serialised appends that
  read the same instant are refused, because monotonic means *strictly* greater.
  Measured after the change on `darwin/arm64`, whose clock resolves to
  microseconds — 1 280 concurrent admissions across 20 runs, zero refusals, so
  the critical section is longer than a tick there; on `linux/amd64` the clock
  resolves to nanoseconds and the question does not arise. **It would arise on a
  coarser clock or a faster store**, and the symptom would be the same
  `ErrNonMonotonicKnowledge` this decision exists to remove. Nothing needs the
  throughput today; a batch endpoint would, and that is the signal to revisit.
- **Unrelated streams occasionally serialise**, by construction, as the price of
  a bounded lock table. Raising the shard count trades memory for collisions and
  is a tuning decision, not a correctness one.
- **The multi-process case is untouched and now less visible.** With the
  single-process noise gone, a two-writer deployment fails rarely rather than
  constantly — which is worse to diagnose. This is the cost that would make this
  decision wrong later: if FDOS deploys two writers before answering ADR-0034's
  open clock-skew question, this change will have made the symptom quieter
  without making the problem smaller.
- **A correctness property now depends on a lock nobody can see from the type.**
  `Store` implementations cannot tell whether their caller serialised. The store
  check is what keeps that honest, which is the argument for never removing it.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| Concurrent admission to one stream loses no facts | **3** | test: N goroutines, **one shared** `Ledger`, one stream, assert N admitted and N distinct `Ref`s |
| Knowledge time is monotonic per stream | 3 | unchanged — the store's existing check, now a cross-process guard |
| The lock table is bounded | **1** | a fixed-size array; there is no code path that grows it |
| Two writers in two processes are ordered | **none** | nothing. See §"What this does not fix" |

**Execution-context question.** The first row is the one this decision exists
for, and it is currently at rung none because the property is false and nothing
reports it — `make verify` is green on a tree where nine in ten concurrent
admissions fail. The test is the deliverable, and it must be **mutation-checked**:
remove the lock, confirm the test goes red naming `ErrNonMonotonicKnowledge`,
restore.

The last row is the honest weak point, and it is inherited rather than
introduced.

**Constitution §15 moves no principle up a rung.** This adds a mechanism where
there was none for a property no row claims; §7 (Temporal Modeling) already
sits at rung 1 for the *shape* of knowledge time, which is a different
statement from its availability under concurrency.

**One §15 row is corrected downward-in-accuracy by this change**, and the
correction is M10's rather than this decision's: row 4 (Immutable Ledger) cited
*"the store refuses a shorter stream"*, a mechanism ADR-0034 deleted along with
`Save`. Verified against the tree — no `Save` remains in `libs/ledger` or
`libs/ledger-sqlite`. The rung does not change; the description now names a
mechanism that exists.

## Alternatives considered

**A — the producer retries.** Return the error and let the caller re-send.
Rejected on the measurement: at 4 admitted of 32 this is the common path rather
than a rare-conflict contract, retries collide with each other, and it asks a
producer to compensate for FDOS's internal clock ordering — a property it cannot
observe. ADR-0029 assumes a hostile producer and never makes one load-bearing.
It was one of the two answers [RFC-0014](../rfc/0014-the-ledger-event-store.md)
contemplated for the open retry contract, and it loses to evidence that did not
exist then.

**B — the application layer retries internally.** Catch
`ErrNonMonotonicKnowledge`, re-read the clock, retry with a bounded count.
Rejected, and it was the closest: it is genuinely *safe* for admission, which
passes `Any()` and reads nothing, so a retry re-does only the clock reading. It
is a bug for `MintIdentity`, which passes `AtLength(n)` and whose retry would
paper over a real conflict — and a mechanism correct for one caller and wrong
for its neighbour is one somebody will generalise. It also converges badly: the
window stays open, so retries grow with the number of waiters. Serialising
removes the window; retrying samples it and hopes.

**C — the service serialises.** Same mechanism in `apps/`. Rejected on the
directory contract and on reimplementation, above. The contract is not a
formality here: it is the reason the property is testable at all.

**D — the store assigns knowledge time.** Rejected, and already rejected by
ADR-0034 for a structural reason nothing in the measurement changes:
`domain.NewEnvelope` requires `temporal.Coordinates`, which include knowledge
time, so an envelope cannot exist before it is known, and a store assigning it
would move the one constructor that makes a fact well-formed outside the domain
that defines well-formed. Recorded because it is the first thing a reader
proposes.

**E — a hybrid clock at the serialisation point.** The store assigns
`max(now, last + ε)` per stream: monotonic by construction, no locks, no
retries, no refusals. It is what mature event stores do and it is the only
option that removes the failure rather than closing the window it occurs in.

**Declined rather than dismissed, and this is the uncomfortable one.** It
inherits D's structural problem and adds its own: knowledge time would stop
being *the instant the application read the clock* and become *an instant not
before that, nudged forward under write pressure*, with the nudge compounding
under sustained load. ADR-0009 permits it — machine-assigned and monotonic are
both satisfied — which is not the same as it being free. It changes what a
recorded number **means** in a ledger built to answer a regulator's question
about what was known when.

That is a decision about the meaning of recorded truth and it wants its own RFC.
This decision fixes the measured failure without changing what any recorded
number means. **If the throughput ceiling in §"Negative" ever binds, E is where
to go**, and it is written down here so the reasoning need not be rediscovered.

## Notes

Accepted by @FabioCaffarello, who accepted RFC-0015.

The measurement is reproducible: the gate's spike ran as a deliberate `t.Error`
on a throwaway pull request
([#69](https://github.com/FabioCaffarello/fdos/pull/69), closed, branch deleted)
because `make test` passes no `-v` and a passing test's log never reaches CI. The
permanent version is the enforcement table's first row.

Open and deliberately not decided here:

- the retry contract RFC-0014 left open, now **narrower**: this removes the
  single-process cause of `ErrNonMonotonicKnowledge` and removes no cause of
  `ErrStaleRead`, which `MintIdentity` and `CorrectFact` can still see;
- the shard count, which is a resource decision and is to be a constant with its
  reasoning recorded rather than a knob nobody tunes;
- whether clock skew needs an operational answer before more than one writer is
  deployed — ADR-0034's open item, unchanged;
- D2 ([#64](https://github.com/FabioCaffarello/fdos/issues/64)), which is why the
  lock table is bounded by construction rather than by policy.
