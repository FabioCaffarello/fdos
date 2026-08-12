---
id: ADR-0041
title: The write path serialises in the store, not in the application
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes:
  - ADR-0036
superseded_by: []
---

# ADR-0041 — The write path serialises in the store, not in the application

## Context

Records what [RFC-0017](../rfc/0017-the-write-path-serialises-in-the-store.md)
settled.

[ADR-0036](0036-knowledge-time-is-assigned-under-the-streams-write-lock.md)
closed the window between reading the clock and appending with a sharded lock
table held by `app.Ledger`. That decision was right for the topology it was made
in and it is not being reversed on its merits — the rule it states, *knowledge
time is assigned under the stream's write lock*, survives this ADR intact. What
changes is where the lock is, because a mutex serialises writers in one process
and the platform now runs two.

[RFC-0015 §2](../rfc/0015-the-submission-service-and-the-admission-race.md) said
this in advance:

> Two processes against one SQLite file reintroduce it exactly as ADR-0034
> described … This is not a gap this RFC closes, and it should not be papered
> over in a deployment story.

A local stack running `submitd` beside an operator CLI is that deployment story,
and it is the next milestone rather than a hypothetical.

### The measurement

`darwin/arm64`, Go 1.26.5. Each child process opens `libs/ledger-sqlite` at one
path and constructs its own `app.Ledger`, hence its own lock table — the
deployment topology, not a simulation. 128 concurrent admissions of a valid
claim to one stream, redistributed:

```
procs= 1   admitted=128   nonMono=0
procs= 2   admitted=127   nonMono=1
procs= 4   admitted=125   nonMono=3
procs= 8   admitted=121   nonMono=7
procs=16   admitted=106   nonMono=22
```

**The mildness is the finding, not the reassurance.** ADR-0036 measured 3 to 9
of 32 in one process before its lock existed. Two processes here lose one append
in 128 — and they share no lock, so what holds the line is SQLite's
single-writer file lock serialising the transactions themselves.
[ADR-0035](0035-the-sqlite-driver-and-its-provenance-risk.md) recorded that
property as a cost when it chose the engine, and it is now the only thing
standing in for the mechanism this ADR moves.

### Why ADR-0036's own reasoning does not defend its placement here

ADR-0036 argued the lock belongs in `libs/ledger/app` rather than in a
composition root, on three grounds. Two of them are untouched, and the third is
the one that turns:

- *`apps/README.md` forbids logic that cannot be tested without starting a
  process.* Still true, and this ADR does not put anything in `apps/`.
- *A second composition root would have to reimplement it.* Still true, and a
  port with a conformance suite is the stronger version of that argument, not a
  weaker one.
- *`app` already owns the clock.* **This is the one that flips.** The clock is
  owned by `app` and the serialisation is owned by whatever every writer can
  see. While those were the same process they could be the same place. They are
  not the same process any more, so the clock read moves to the serialisation
  point rather than the serialisation moving to the clock.

Constitution §10 is the principle underneath: the domain must not depend on
infrastructure. An application-layer mutex whose guarantee depends on how many
processes the deployment runs is an infrastructure fact leaking upward wearing a
`sync.Mutex`.

## Decision

**FDOS moves mutual exclusion into the `app.Store` port.**

```go
// Serialise runs fn with exclusive write access to `name`, against every writer
// holding this store — in this process or any other.
//
// The Store passed to fn is this store scoped to the serialised region. Calling
// Serialise on it deadlocks, and implementations refuse it.
Serialise(ctx context.Context, name string, fn func(context.Context, Store) error) error
```

Every write use case reads the clock, builds its envelope and appends **inside**
the callback. `streamLocks` leaves `libs/ledger/app` and becomes the in-memory
adapter's implementation of `Serialise`, carrying ADR-0036's sharding argument
with it — stream names are producer-supplied and unbounded while D2 is open, so
a bounded array stays the answer and a per-name map stays refused.

### What moves, and what deliberately does not

**Only the mutual exclusion moves.** `domain.NewEnvelope` still constructs the
envelope, in the layer that defines what well-formed means, and `Append` still
takes a built `domain.Envelope`.

This is the entire distinction from the repair ADR-0034 and RFC-0015 both
rejected — *the store assigns knowledge time* — and it is stated here because
the two are indistinguishable from a distance. That alternative relocates
**envelope construction** into the store, which would move the one constructor
that makes a fact well-formed outside the domain that defines it. This
relocates **mutual exclusion**, which the domain never defined.

### What each engine provides

| Engine | `Serialise` | Granularity |
|---|---|---|
| `adapters/memory` | the sharded mutex table, relocated | per shard, one process |
| `libs/ledger-sqlite` | holds the `BEGIN IMMEDIATE` transaction across `fn` | whole database, all processes |
| `libs/ledger-postgres` | `pg_advisory_xact_lock` inside the transaction | per stream, all processes |

**SQLite becomes correct across processes under this decision**, because the
write lock is already held when the callback runs and the clock is read inside
it. The measured refusals go to zero with no second engine involved. That is
deliberate: this ADR is shippable alone, and
[ADR-0042](0042-postgresql-is-the-second-engine.md) is a capacity and topology
decision that depends on it rather than the other way round.

## Consequences

### Positive

- Two processes can write to one ledger, which is what a local stack is.
- The application layer stops owning a concurrency primitive it cannot make
  honest, so Constitution §10 holds at the layer boundary rather than
  approximately.
- The serialisation point becomes an explicit, named region in the port. If the
  hybrid clock RFC-0015 parked as its alternative E is ever needed, there is now
  one place for it to live.
- A store that cannot serialise cannot compile, so the property is rung 1 rather
  than a convention every adapter is trusted to know about.

### Negative

- **The port grows a method**, and every implementation must provide it —
  including any out of tree. `app.Store` is published under ADR-0025 with no
  compatibility promise, which makes this permitted rather than free.
- **The critical section is longer.** It now spans the clock read, and for
  `ObserveClaimedHolding` a full `Load` of the stream. On SQLite that section is
  guarded by a database-wide lock, so throughput becomes a platform ceiling
  rather than a per-stream one. This is the cost ADR-0042 exists to pay down.
- **A callback-shaped API invites a deadlock** that the previous design could
  not express: calling `Serialise` on the scoped store, or capturing the outer
  store inside the callback. Implementations refuse the first; the second is
  documented and unenforced.
- **`SetMaxOpenConns(1)` on SQLite now blocks readers behind a writer** for the
  whole region. Whether to raise it is left open below.
- What would make this wrong later: a platform whose wall clock is coarser than
  an append. The region makes writers sequential; it does not make the clock
  finer, and two appends inside one tick still collide.

### Enforcement

| Property | Rung | Mechanism |
|---|---|---|
| Every store serialises | 1 | on the interface; an implementation without it does not compile |
| Envelope construction stays in the domain | 1 | `Append` takes a built `domain.Envelope`; the store has no constructor |
| Serialised writers in one process do not refuse each other | 3 | a `storetest` case, every engine |
| Serialised writers in separate processes do not refuse each other | 3 | a process-spawning case, per driver |

The fourth row carries a warning earned while preparing RFC-0017. **A first
version of that test used goroutines and passed against the defect it existed to
catch** — two `*sql.DB` handles in one process do not contend the way two
operating-system processes do. The working pattern re-executes the test binary
via `TestMain` and an environment variable, and it exists in
`libs/ledger-sqlite` to copy.

`storetest.Run` takes `func(*testing.T) app.Store`, which cannot be handed to a
child process. The shared suite gains a second entry point taking a DSN and a
re-exec hook rather than each driver carrying its own cross-process cases: a
per-driver copy is how two engines begin disagreeing about what the port means,
which is the failure the suite exists to prevent (ADR-0034).

## Alternatives considered

Each is argued in full in RFC-0017; the reason it lost is recorded here.

- **One writer process — the CLI submits over the wire.** Rejected: it makes the
  operator surface require a running service, a network and an answer to D2,
  which are the three things that surface is valuable for not needing. It also
  fails its own premise, since two service instances reintroduce the problem, so
  the real constraint is "deploy exactly one process, forever", enforced by
  nothing.
- **Retry with backoff.** Rejected: `ErrNonMonotonicKnowledge` is not contention
  over a resource that frees, it is a statement that this writer's clock reading
  is now in the past. Retrying busy-waits until the wall clock passes another
  process's reading, converting a correctness signal into latency.
- **The store assigns knowledge time.** Rejected, for the third time.
  `domain.NewEnvelope` requires `temporal.Coordinates`, which include knowledge
  time, so an envelope cannot exist before it is known and a store assigning it
  would construct one.
- **A hybrid clock, `max(now, last + ε)`.** Not taken. It removes the failure
  entirely and changes what a recorded number means, from *the instant the
  application read the clock* to *an instant not before that, nudged forward
  under write pressure*. RFC-0015 called this its least comfortable position and
  left the signpost; this decision does not foreclose it and makes it cheaper to
  reach.
- **Keep SQLite, accept the refusals, document them.** Rejected on the
  measurement. 106 of 128 at sixteen processes is not a documentation problem.

## Notes

**Release sequence.** `app.Store` is in `libs/ledger` and `libs/ledger-sqlite`
pins a published version of it, so this is a cross-module break and travels in
the order the ADR-0040 sequence learned: `libs/ledger` released first, then
`libs/ledger-sqlite`, then `libs/ledger-postgres`, then `apps/submitd` and
`examples/ingest`. `docs/ecosystem/contracts.md` records each release, because
ADR-0024 calls the registry part of the interface.

**`make verify` cannot see step one breaking step two.** `FOR_EACH_MODULE` runs
`GOWORK=off` and resolves siblings from the proxy at published versions, and
`libs/ledger-sqlite` is additionally absent from `go.work`. Both are
pre-existing, both bite here, and both belong to
[#79](https://github.com/FabioCaffarello/fdos/issues/79).

**Left open deliberately.** Whether SQLite keeps `SetMaxOpenConns(1)` now that a
region holds the connection — a capacity question, not a correctness one.

**Not decided here.** Who may write to a named stream. `Serialise` decides
ordering, never permission; D2
([#64](https://github.com/FabioCaffarello/fdos/issues/64)) is untouched.
