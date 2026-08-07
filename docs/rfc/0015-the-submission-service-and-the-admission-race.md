---
id: RFC-0015
title: The submission service, and the admission race a transport makes reachable
status: Accepted
date: 2026-08-07
authors:
  - "@FabioCaffarello"
---

# RFC-0015 — The submission service, and the admission race a transport makes reachable

## Summary

M11 puts the first composition root in `apps/`: a process that receives a
`fdos.ingest.v1.HoldingClaimSubmission` from outside and admits it. Three things
have to be settled before that can be written, and only one of them is about
transport.

1. **The admission race.** Measured at the gate: 32 concurrent submissions to
   one stream are refused 23–29 times with `ErrNonMonotonicKnowledge`, on
   `linux/amd64`, with a clock that hands every caller a distinct instant. This
   is reachable today from one process and it has no owner. Fixing it is a
   change to `libs/ledger`, not to a transport.
2. **Whether a service is allowed at all.**
   [ADR-0029](../adr/0029-the-public-surface-receives-a-claim.md) decided
   delivery is *"a library, never a service FDOS operates"* and its alternatives
   section says *"a service"* without the qualifier. The two do not say the same
   thing, and M11 sits in the gap.
3. **The transport itself**, which is the smallest of the three.

It cannot be settled with an ADR alone because (1) has five candidate answers
with materially different blast radii — one of them reopens
[ADR-0034](../adr/0034-the-ledger-event-store.md) — and because (2) is a
question about the meaning of an accepted decision rather than a new one.

**D2 is a dependency, not a subject.** This RFC names what the service does
while D2 is open and proposes nothing that would answer it.

## Motivation

### What breaks

**A transport makes an existing defect reachable, and nothing reports it.**
`make verify` is green on a tree where nine of ten concurrent admissions fail.
The property is not tested because until now the only caller of
`app.Ledger.AcceptHoldingClaim` was a single-threaded test in the same process.

The measurement, from a throwaway pull request whose red `verify` run was the
evidence ([#69](https://github.com/FabioCaffarello/fdos/pull/69), closed):

```
platform: linux/amd64  NumCPU=4  GOMAXPROCS=4
clock: 10000 readings -> 10000 distinct instants

round 1  concurrent/wall:     admitted=7 nonMono=25 stale=0
         concurrent/distinct: admitted=5 nonMono=27 stale=0
         sequential/wall:     admitted=32 nonMono=0 stale=0
round 3  concurrent/wall:     admitted=4 nonMono=28
         concurrent/distinct: admitted=3 nonMono=29
         sequential/wall:     admitted=32 nonMono=0
```

Two readings matter and the second is the one that makes this an RFC.

**The clock is not the problem.** The `distinct` case injects a clock that hands
every caller an instant no other caller received. It is indistinguishable from
the wall-clock case. Nanosecond resolution — 10 000 distinct readings from
10 000 — does not help either. **No clock fixes this.**

**A sequential caller is fine.** 32 of 32, every round. The failure appears
exactly when more than one caller exists, which is exactly what a transport
creates.

### Why it happens

`AcceptHoldingClaim` reads knowledge time, builds the envelope, and only then
reaches the store:

```go
coordinates, err := temporal.Assign(cmd.Effective, l.clock.Now())   // (1) read
…
ref, err := l.store.Append(ctx, cmd.Stream, Any(), envelope, …)     // (2) append
```

Between (1) and (2) the caller queues for the store's write lock —
`SetMaxOpenConns(1)` and `_txlock=immediate`, both deliberate. A caller that
read at *t₁* and arrives after one that read at *t₂ > t₁* is refused, correctly:
ADR-0009 requires knowledge time monotonic per stream, and the store is the only
place that can check it.

**ADR-0034 predicted this and mis-scoped it.** It recorded, under negative
consequences:

> **Clock skew becomes write unavailability.** With two processes and two
> clocks, a writer whose clock lags has a *legitimate* append rejected.

The prediction is right and the precondition is wrong. It needs neither two
processes nor two clocks nor any skew. **One process, one clock, two
goroutines** is sufficient, because the unserialised gap is between the clock
read and the write lock, not between two clocks.

### Which principle is at stake

Constitution §1 (Financial Truth) and §4 (Immutable Ledger) are what the store's
check protects, and it protects them correctly — this RFC does not propose
weakening it. What is at stake is whether the ledger can **accept** the traffic
it was built to accept. A truth engine that refuses nine in ten honest facts is
not safe, it is unavailable, and E9's *"any third party can ingest"* is not met
by a path that rejects most of what it is given.

### Is this retrofittable?

**The fix is. The data is not.** Whatever is chosen, facts already written keep
the knowledge times they were given, and no migration is implied. But the
*shape* of knowledge time — whether it remains exactly "the wall-clock instant
the application read" — is a property that every later reader depends on, and
alternative **E** below changes it. That one is not retrofittable, which is why
it is named here rather than discovered as a convenient patch later.

## Design

### §1 — Admission serialises the clock read with the append

**Proposed: `app.Ledger` holds the clock read and the append together, per
stream.**

```go
type Ledger struct {
    store  Store
    clock  Clock
    rules  identity.Ruleset
    writes *streamLocks   // new
}
```

Every write use case — not only admission — takes the stream's lock, reads the
clock, appends, releases. The store's monotonicity check stays exactly as it is
and keeps its job: it is the cross-process guard, and it is honest about being
one.

**Why in `app` and not in the service.** Three reasons, in order of weight:

- **It is a correctness property of the ledger, not of a transport.** A second
  composition root — the conformance kit, a CLI, a test harness — would have to
  reimplement it, and the one that forgot would lose facts silently.
- **`apps/README.md` forbids exactly this.** *"Logic that cannot be tested
  without starting a process"* is a listed prohibition. A serialisation
  mechanism in the service is testable only by running the service; the same
  mechanism in `libs/ledger/app` is testable by starting goroutines.
- **`app` already owns the clock.** ADR-0034 put knowledge-time assignment in
  the application layer deliberately, and the gap this closes is inside the
  window that decision created.

**Why per stream and not global.** Knowledge time is monotonic *per stream*
(ADR-0009), so a global lock would serialise unrelated accounts to buy nothing.

**The cost, and it is a real one.** Stream names are producer-supplied and
unbounded (ADR-0030), so a map of per-stream locks is a map a hostile producer
can grow without limit — one lock per fabricated stream name, from a caller that
never has to succeed. That is a memory-exhaustion surface created by this
design, and it is **D2-adjacent**: the reason nothing bounds it is that nothing
knows who may name a stream.

Mitigations that do not require D2, in preference order:

1. **Evict.** A bounded map of locks, keyed by stream, with the lock released
   and the entry dropped when no writer holds it. Correctness does not depend on
   the lock persisting — only on it existing for the duration of a write — so
   eviction is safe and the bound is a resource decision rather than a
   correctness one.
2. **Shard.** A fixed array of *N* locks indexed by a hash of the stream name.
   Constant memory, and two unrelated streams occasionally serialise. Simplest
   thing that is obviously correct.

**Sharding is proposed.** It is bounded by construction, it has no eviction race
to get wrong, and the cost — occasional false serialisation between unrelated
streams — is throughput, not correctness. The unbounded map is what a reviewer
would reach for first and it is the one that hands a stranger an allocation
primitive.

### §2 — What this does *not* fix, stated rather than discovered

**Two processes.** The gap between clock read and append closes inside one
process. Two service instances against one SQLite file reintroduce it exactly as
ADR-0034 described, and the store's check remains the only thing standing
between them and a broken ordering.

This is not a gap this RFC closes, and it should not be papered over in a
deployment story. SQLite is single-writer (ADR-0035), nothing deploys two
writers today, and ADR-0034 recorded the multi-writer question as open. **What
changes is that the store's check becomes a genuine cross-process guard instead
of a mechanism that fires constantly in the single-process case for reasons
having nothing to do with skew.**

### §3 — The transport: HTTP over `net/http`, one endpoint, protobuf body

**Proposed:** HTTP/1.1 on the standard library, a single `POST` endpoint taking
a length-delimited `fdos.ingest.v1.HoldingClaimSubmission` as
`application/x-protobuf`, returning the assigned `domain.Ref` or a typed error.

**Zero new dependencies.** This is the discriminating criterion and it is the
same one ADR-0035 used: build-input integrity first. `net/http` is in the
toolchain already pinned by `mise.toml`; gRPC is `google.golang.org/grpc` plus
its transitive graph, which would be the **second** heavy dependency in the
repository and would arrive with the same audit obligation ADR-0035 paid for the
first — for a milestone whose deliverable is one endpoint.

**The contract stays protobuf** (ADR-0018). The body is the published message,
not a JSON re-expression of it, so a producer in any language sends the same
bytes the conformance kit produces. No wire type crosses into the domain:
`ledgerwire.DecodeHoldingClaimSubmission` already returns an
`app.AcceptHoldingClaimCommand` and that is the only thing the handler passes on.

**gRPC stays available and additive.** The port is the app-layer use case, not
the handler, so a gRPC front end is a second composition root when a consumer
asks for one. Choosing the smaller thing first is reversible; choosing the
larger first is not.

### §4 — What the service does while D2 is open

**It refuses to pretend.** Three proposals, none of which is an authentication
model:

- **Bind to loopback by default**, and require an explicit flag to bind
  anywhere else — one whose name states what it means rather than one called
  `--host`. A default that listens on `0.0.0.0` answers D2 by accident, in the
  direction of "anyone", and nobody would have written that decision down.
- **Refuse to start if configured to listen off-loopback without an
  acknowledgement flag.** The operator asserts they have put authentication in
  front of it. This is the `unmediated` pattern from ADR-0028: an explicit
  assertion beats an absent one, and there is deliberately no value meaning
  "I did not think about it".
- **Say so in `apps/<name>/README.md`**, which is a binding directory contract
  here, not documentation.

**This is not an answer to D2** ([#64](https://github.com/FabioCaffarello/fdos/issues/64)).
It is a refusal to answer it by default. The distinction matters: option A in
that issue — *the deployment boundary is the answer* — may well be the decision,
and if it is, it deserves an ADR rather than a flag default that grew into a
policy.

### §5 — Scope

**Not covered, and not by omission:**

- **Batch submission.** One fact per request. A batch is a different admission
  semantics question — partial success — and it wants its own decision.
- **Rate limiting, quotas, abuse.** ADR-0029 assigned these to D2.
- **TLS.** The operator's, under §4.
- **Read surfaces.** No query endpoint, no MCP. Both are D2's other half.
- **Multi-instance deployment.** §2.
- **Crash-safety under real power loss.** ADR-0035's named open gap; this RFC
  does not close it and the service's documentation must not imply it is closed.

## Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| Concurrent admission to one stream does not lose facts | **3** | test: N goroutines, one stream, assert N admitted — the test the gate's throwaway spike should have been |
| Knowledge time is monotonic per stream | 3 | unchanged: the store's existing check, now a cross-process guard rather than a constant firing |
| The transport adds no dependency | **2** | `make tidy-check` and `make vuln-check` over an `apps/` module whose `go.mod` requires only first-party modules and `google.golang.org/protobuf` |
| No wire type reaches the domain | 2 | the analysers; the handler's only outbound call takes an `app.AcceptHoldingClaimCommand` |
| `apps/` holds no logic | 2 | `make contracts-check` against the directory contract |
| The service does not listen off-loopback by accident | **3** | test: default configuration binds loopback; off-loopback without the acknowledgement flag refuses to start |
| Only an authorised party may write to a stream | **none** | **D2 is open.** Unchanged by this RFC and it must not be read as changed |

The first row is the one that matters and it is the mechanism this whole RFC
exists to make possible. **It is currently at rung none**, because the property
is false and nothing reports it.

## Alternatives

### A — The producer retries

Return the error; the producer re-sends. **Rejected.**

On the measured numbers this is not a retry contract for a rare conflict, it is
the common path: a producer sending 32 concurrent submissions succeeds with 4
and retries 28, and those retries collide with each other. It also asks the
producer to compensate for a property it cannot observe — FDOS's internal clock
ordering — which puts part of the ledger's correctness on the far side of the
truth boundary. ADR-0029 is explicit that a producer is assumed hostile and
never load-bearing.

It is not absurd: RFC-0014 left the retry contract open and this is one of the
two answers it contemplated. It is rejected on the measurement, which did not
exist when that was written.

### B — The application layer retries internally

`AcceptHoldingClaim` catches `ErrNonMonotonicKnowledge`, re-reads the clock,
retries with a bounded attempt count. **Rejected, but it is the closest
alternative and worth the reasoning.**

It is *safe* in a way it would not be for other use cases: admission passes
`Any()` and reads nothing, so there is no stale decision to re-derive — the
retry re-does only the clock reading. For `MintIdentity`, which passes
`AtLength(n)`, a blind retry would be a bug, and a mechanism that is correct for
one caller and wrong for its neighbour is a mechanism someone will generalise.

It also converges badly for the same reason A does: under contention, the window
between the retried clock read and the append is still open, so the retry count
grows with the number of waiters. Serialising removes the window; retrying
samples it repeatedly and hopes.

### C — The service serialises per stream

Same mechanism, in `apps/` instead of `libs/ledger/app`. **Rejected** on
`apps/README.md`'s prohibition on logic that cannot be tested without starting a
process, and because a second composition root would have to reimplement it. The
directory contract is not a formality here — it is the reason the property can
be tested at all.

### D — The store assigns knowledge time

**Rejected, and it was already rejected.** ADR-0034 recorded the structural
reason: `domain.NewEnvelope` requires `temporal.Coordinates`, which include
knowledge time, so an envelope cannot exist before it is known. A store that
assigned it would construct the envelope, moving the one constructor that makes
a fact well-formed outside the domain that defines well-formed.

Nothing in the measurement changes that argument. It is recorded here because it
is the first thing a reader will propose.

### E — Knowledge time becomes a hybrid clock at the serialisation point

The store assigns `max(now, last + ε)` per stream: monotonic by construction,
no rejections, no retries, no locks. This is what mature event stores do, and it
is the only alternative that removes the failure entirely rather than closing
the window it occurs in.

**Not proposed, and this is the RFC's least comfortable position.**

It inherits D's structural problem, and it adds one of its own: knowledge time
would stop being exactly *"the instant the application read the clock"* and
become *"an instant not before that, nudged forward under write pressure"*.
ADR-0009 requires knowledge time to be machine-assigned and monotonic, both of
which a hybrid clock satisfies — so this is not forbidden, it is a change to
what a recorded number **means**, in a ledger built to answer a regulator's
question about what was known when. Under sustained write pressure the nudge
compounds, and a fact's knowledge time can drift ahead of when FDOS actually
learned it.

That is a decision about the meaning of recorded truth and it belongs to a human
in its own RFC, not to this one as a convenience. §1 is proposed because it
fixes the measured failure **without changing what any recorded number means**.
If §1's throughput ceiling later proves to be the binding constraint, E is where
to go, and it is written down here so that the reasoning does not have to be
rediscovered.

### F — No service; M11 ships a CLI

A composition root that reads submission bytes from a file or stdin and admits
them. **Not rejected — see §Open questions.** It satisfies ADR-0029 as written
with no interpretation required, delivers `apps/`, and defers D2 honestly. It
also does not exercise the concurrency the finding is about, which is either a
feature or the reason it is the wrong milestone.

## Prior art

**Event stores with optimistic concurrency.** EventStoreDB and Marten both
expose expected-version preconditions and both assign the ordering value at the
serialisation point rather than in the caller. The lesson taken here is narrow
and specific: the value that establishes total order is assigned where writes
serialise, and every system that let a caller compute it has an incident report
about it. ADR-0034 already applied this to `sequence`. It did not apply it to
knowledge time, and the gap between the two is exactly the failure measured
above.

**Hybrid logical clocks** (Kulkarni et al., as deployed in CockroachDB) are
alternative E, and their published trade — a timestamp that is monotonic and
approximately wall-clock rather than exactly wall-clock — is the trade this RFC
declines to make without a separate decision.

**Bitemporal financial systems** generally separate *transaction time* assigned
by the system from *valid time* asserted by the source, which is exactly
ADR-0009's split. What they mostly do not do is expose the system-assigned axis
to write-path failure, because they assign it at commit. FDOS assigns it before
commit for a stated structural reason, and that is where the cost lands.

## Open questions

1. ~~**Is a service permitted at all?**~~ **Closed by
   [ADR-0037](../adr/0037-delivery-includes-a-service-the-adopter-operates.md),
   which supersedes ADR-0029.** A service an adopter operates is permitted; a
   service FDOS operates stays rejected.

   This RFC recommended a **narrowing** — the decision body already carries the
   qualifier, and RFC-0008 with ADR-0026 is the repository's precedent for
   narrowing rather than superseding. **The recommendation was not taken**, and
   the argument that won is recorded in ADR-0037 §Context: a decision whose
   alternatives section flatly rejects *"a service"* cannot be read into
   permitting one without the reading itself being a decision, and a narrowing
   would have left `status: Accepted` on a document that refuses what the next
   milestone builds.

   Left here rather than deleted. The recommendation lost on its merits and a
   reader should be able to see that it was made.
2. **D2** ([#64](https://github.com/FabioCaffarello/fdos/issues/64)) — who may
   write to a named stream. §4 refuses to answer it by default; it does not
   answer it. `risk/truth-path`-adjacent.
3. **Does the retry contract RFC-0014 left open still need answering?** §1
   removes the single-process cause of `ErrNonMonotonicKnowledge`. It does not
   remove `ErrStaleRead`, which `MintIdentity` and `CorrectFact` can still see.
   The contract is now narrower and still open.
4. **Sharded locks: how many shards?** A resource decision, not a correctness
   one. Proposed to be a constant with the reasoning recorded rather than a
   configuration knob nobody tunes.

## Consequences

### Easier

- Concurrent admission works, which is the milestone.
- The store's monotonicity check becomes meaningful: it fires on genuine
  cross-process disorder instead of on ordinary single-process traffic, so an
  operator who sees it learns something.
- A second composition root — the conformance kit, a CLI, a gRPC front end —
  inherits the fix, because it lives in the use case rather than in the handler.

### Harder

- **`app.Ledger` acquires concurrency state**, which every write use case now
  passes through. It is the first mutable, shared, non-injected state in that
  type, and a test that constructs a `Ledger` per goroutine will not exercise
  it — which is a trap worth a comment in the code.
- **Per-stream write throughput acquires a ceiling** equal to one append
  latency. Nothing needs more today; a batch endpoint would.
- **A hostile producer gains an allocation surface** unless the locks are
  bounded. §1 proposes sharding for exactly this reason, and the reason is D2.

### Impossible

- **Silently losing a concurrent admission.** It is already impossible — the
  store refuses rather than losing — and what changes is that the honest refusal
  stops being the common case.
- **Deploying two writers and claiming ordering.** Unchanged by this RFC and
  restated because a running service invites exactly that, and §2 is the only
  place it is written down.
