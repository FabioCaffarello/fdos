---
id: RFC-0017
title: The write path serialises in the store, and the second engine
status: Accepted
date: 2026-08-12
authors:
  - "@FabioCaffarello"
---

# RFC-0017 — The write path serialises in the store, and the second engine

## Summary

[ADR-0036](../adr/0036-knowledge-time-is-assigned-under-the-streams-write-lock.md)
closed the window between reading the clock and appending, using a lock table
inside `app.Ledger`. That lock serialises writers **in one process**, and
[RFC-0015 §2](0015-the-submission-service-and-the-admission-race.md) said so
explicitly, adding that two processes reopen the window and that this "should
not be papered over in a deployment story."

A local stack is that deployment story. `submitd` holds the database while an
operator runs a CLI against it, and the two have no lock in common.

This proposes that **the mutual exclusion move from the application into the
store port**, as one new method, and that **PostgreSQL become the second
engine** — in that order, because the first is a correctness fix that works on
the engine already shipped, and the second is a topology fit that is only safe
once the first exists.

It cannot be settled with an ADR alone because the obvious repair — let the
store assign knowledge time — is already rejected twice, and the repair that
removes the failure entirely changes what a recorded number *means*. What is
proposed here is a third shape that does neither.

## Motivation

### What breaks

Measured on `darwin/arm64`, Go 1.26.5, with a harness that spawns *processes*:
each child opens `libs/ledger-sqlite` at one path and constructs its own
`app.Ledger`, and therefore its own lock table. That is the deployment topology
rather than a simulation of it. Total load held at 128 concurrent admissions of
a valid claim to one stream, redistributed across processes:

```
procs= 1   n/proc=128   admitted=128   nonMono=0
procs= 2   n/proc= 64   admitted=127   nonMono=1
procs= 4   n/proc= 32   admitted=125   nonMono=3
procs= 8   n/proc= 16   admitted=121   nonMono=7
procs=16   n/proc=  8   admitted=106   nonMono=22
```

**The degradation is far milder than ADR-0036 measured, and the reason is the
finding.** ADR-0036 recorded 3 to 9 admitted of 32 in one process before the
lock table existed. Here two processes lose one append in 128. What is holding
the line is not the lock — the two processes share none — it is SQLite's
single-writer file lock, which serialises the transactions themselves.

[ADR-0035](../adr/0035-the-sqlite-driver-and-its-provenance-risk.md) wrote this
down as a cost at the time it chose the engine:

> SQLite is single-writer, so ADR-0034's `Expectation` and knowledge-time
> monotonicity are exercised against a store that serialises anyway.

That sentence is now load-bearing in the other direction. **The property keeping
the two-process case at 127 of 128 is exactly the property a client/server
engine removes.** Adopting PostgreSQL without moving the lock would not inherit
this table; it would return to ADR-0036's, across processes, with nothing in the
application able to help.

### A second failure, already fixed, that shows the case was untested

While measuring the above, `libs/ledger-sqlite`'s `Open` was found to fail
outright under concurrent process startup: four processes opening one database
lost two of six rounds to
`PRAGMA journal_mode = WAL: database is locked (261)` — SQLITE_BUSY_RECOVERY,
raised before a single fact was appended. `busy_timeout` was the last of four
pragmas, so the three that can take a lock all ran under the default timeout of
zero.

It is fixed and negative-tested. It is cited here because of what it implies:
**no test in the repository had ever opened one database from two processes.**
The conformance suite covers serialised access, which is the case a test has;
concurrent access is the case a deployment has. A design that assumes the
multi-process path is merely unexercised should assume instead that it is
unknown.

### Why it happens

Unchanged from RFC-0015, and worth restating because the fix follows its shape:

```go
defer l.writes.hold(cmd.Stream)()                                   // (0)
coordinates, err := temporal.Assign(cmd.Effective, l.clock.Now())   // (1)
…
ref, err := l.store.Append(ctx, cmd.Stream, Any(), envelope, …)     // (2)
```

(0) makes (1) and (2) one step for every writer that shares the mutex. A writer
in another process shares nothing, so it reads its clock at *t₁*, arrives after
a writer that read at *t₂ > t₁*, and is refused by the store's monotonicity
check — correctly, and for a reason having nothing to do with clock skew.

### Which principle is at stake

Not correctness. Nothing here admits a fact it should refuse; the refusals are
the invariant of Constitution §7 and
[ADR-0009](../adr/0009-universal-bitemporality.md) doing their job.

What is at stake is §14 and the enforcement ladder's obligation to climb. A
ledger that refuses a fraction of the traffic it was built to accept, for a
reason the producer cannot act on and the operator cannot see, is a ledger whose
correctness is paid for by somebody else. RFC-0015 accepted that framing for one
process and fixed it. The same argument applies one topology out.

Constitution §10 is the second: the domain must not depend on infrastructure.
`app.Ledger` currently owns a concurrency primitive whose guarantee depends on
how many processes the deployment runs — which is an infrastructure fact leaking
into the application layer while wearing a `sync.Mutex`.

### Is this retrofittable?

**Yes, and that is the strongest argument for the shape proposed here.**

No fact changes meaning. Knowledge time stays "the instant the application read
the clock", so a fact written before this and a fact written after are the same
kind of thing, and no migration exists to be owed. Alternative E below does not
have this property, which is why it is not proposed even though it is simpler.

What is *not* retrofittable cheaply is the port. `app.Store` is published, and
adding a method breaks every implementation of it — see §5.

## Design

### §1 — The port gains one method

```go
type Store interface {
	Load(ctx context.Context, name string) (domain.Stream, error)

	Append(ctx context.Context, name string, expect Expectation,
		envelope domain.Envelope, kind domain.Kind, payload domain.Payload) (domain.Ref, error)

	// Serialise runs fn with exclusive write access to `name`, against every
	// writer holding this store — in this process or any other.
	//
	// The Store passed to fn is this store scoped to the serialised region.
	// Calling Serialise on it deadlocks and implementations refuse it.
	Serialise(ctx context.Context, name string, fn func(context.Context, Store) error) error
}
```

The application changes from taking a mutex to entering a region:

```go
var ref domain.Ref
err := l.store.Serialise(ctx, cmd.Stream, func(ctx context.Context, s app.Store) error {
	coordinates, err := temporal.Assign(cmd.Effective, l.clock.Now())
	…
	envelope, err := domain.NewEnvelope(coordinates, prov, cmd.References)
	…
	ref, err = s.Append(ctx, cmd.Stream, app.Any(), envelope, domain.KindObservation, payload)
	return err
})
```

**The clock read moves inside the region. Nothing else moves.** The envelope is
still constructed by `domain.NewEnvelope`, in the layer that defines what
well-formed means. This is the entire difference between this proposal and
alternative D, and it is worth stating twice because the two look identical from
a distance: D relocates *envelope construction* into the store; this relocates
*mutual exclusion* into the store.

`streamLocks` leaves `libs/ledger/app` and becomes the in-memory adapter's
implementation of `Serialise`. The application stops owning a lock it cannot
make honest, and the shard-count reasoning ADR-0036 recorded moves with the code
it justifies.

### §2 — What each engine does

| Engine | `Serialise` | Granularity |
|---|---|---|
| `adapters/memory` | the sharded mutex table, relocated verbatim | per shard, one process |
| `libs/ledger-sqlite` | hold the `BEGIN IMMEDIATE` transaction across `fn` | whole database, all processes |
| `libs/ledger-postgres` | `pg_advisory_xact_lock` inside the transaction | per stream, all processes |

Two consequences follow from the table and neither is incidental.

**SQLite becomes correct across processes under this design.** The write lock is
already held when the callback runs, so the clock is read inside it. The
measured refusals go to zero without PostgreSQL existing. §1 is therefore
shippable on its own, and the demo it unblocks does not wait for an engine.

**SQLite's lock is the whole database.** Under §1 the critical section grows: it
now spans the clock read, and for `ObserveClaimedHolding` a full `Load` of the
stream. A global lock over a longer section is a platform-wide throughput
ceiling rather than a per-stream one. That is what the second engine is for, and
it is a capacity argument rather than a correctness one — stated that way so
nobody later reads PostgreSQL as the thing that made the ledger correct.

### §3 — PostgreSQL, and the three reasons that are not "it scales"

1. **Lock granularity.** Per stream rather than per database, which matters
   precisely because §1 lengthens the critical section.
2. **The compose topology puts the file on a volume.** SQLite's cross-process
   locking is the filesystem's advisory locking, and its own documentation is
   explicit that this is unreliable on network filesystems. A container volume
   driver is not something the ledger should have to audit to know whether its
   ordering holds. PostgreSQL takes the ledger's correctness out of the
   filesystem's hands.
3. **The workload is already client/server.** A service, an operator CLI and
   every future reader against one store is the shape PostgreSQL is, and the
   shape SQLite tolerates.

`pg_advisory_xact_lock` takes a 64-bit key, so a stream name must be hashed onto
it and two names can collide. That is the same trade ADR-0036 already accepted
for its 256 shards — two unrelated streams occasionally serialise, which is
throughput and not correctness — and it is preferable to a lock table keyed by
name for the same reason ADR-0036 gave: stream names are producer-supplied and
unbounded while D2 is open, so a per-name row is an allocation primitive handed
to a stranger.

The lock is transaction-scoped rather than session-scoped deliberately: it is
released by commit *or* rollback *or* the connection dying, so a crashed writer
cannot leave a stream locked.

### §4 — What this does not fix, stated rather than discovered

- **Who may write.** `Serialise` decides *ordering*, never *permission*. D2
  ([#64](https://github.com/FabioCaffarello/fdos/issues/64)) is untouched.
- **The clock's resolution.** Monotonic means *strictly* greater, so two
  appends inside one tick still collide. The region makes them sequential rather
  than concurrent, which is what removes the failure; it does not make the clock
  finer. On a platform whose clock is coarser than an append, the ceiling
  returns — and alternative E is still where to go.
- **Cross-stream atomicity.** A multi-account portfolio still cannot be written
  in one step, and refs across streams are still refused.
- **Read scaling.** No query surface exists to scale.
- **`ErrStaleRead`.** `Expectation` still fires for `MintIdentity` and
  `CorrectFact`, and the retry contract RFC-0014 left open is still open.

### §5 — Scope, and the release sequence the port change forces

`app.Store` lives in `libs/ledger`, and `libs/ledger-sqlite` is a separate
module pinned to a published version of it. Adding a method is therefore a
cross-module break, and it must travel in the order the ADR-0040 sequence
learned:

1. `libs/ledger` — port, in-memory adapter, application, released.
2. `libs/ledger-sqlite` — implements `Serialise`, pinned to the new ledger,
   released.
3. `libs/ledger-postgres` — new module.
4. `apps/submitd`, `examples/ingest` — adopt.

`make verify` cannot see step 1 breaking step 2, because `FOR_EACH_MODULE` runs
`GOWORK=off` and resolves siblings from the proxy at published versions
([#79](https://github.com/FabioCaffarello/fdos/issues/79)). `libs/ledger-sqlite`
is additionally absent from `go.work`, so even a workspace build does not cover
it. Both are pre-existing and both bite here.

Out of scope: replication, failover, connection pooling policy, and any
PostgreSQL topology beyond a single instance.

## Enforcement

| Property | Rung | Mechanism |
|---|---|---|
| Every store implements `Serialise` | 1 | it is on the interface; a store that omits it does not compile |
| Envelope construction stays in the domain | 1 | `Append` takes a built `domain.Envelope`; the store has no constructor for one |
| Serialised writers in one process do not refuse each other | 3 | a `storetest` case, run against every engine |
| Serialised writers in **separate processes** do not refuse each other | 3 | a process-spawning case, per driver |
| The advisory lock is released on crash | 3 | kill the connection mid-region and assert the next writer proceeds |

The fourth row is the one that needs care, and the reason is a mistake made
while preparing this RFC. **A first version of the concurrent-open test used
goroutines and passed against the defect it existed to catch.** Two `*sql.DB`
handles in one process do not contend to recover a write-ahead log; two
operating-system processes do. The working test re-executes the test binary via
`TestMain` and an environment variable, and that pattern now exists in
`libs/ledger-sqlite` to copy.

`storetest.Run` takes `func(*testing.T) app.Store`, which a child process cannot
be handed. The cross-process cases therefore cannot live in the shared suite as
it is shaped; either the suite gains a second entry point taking a DSN and a
re-exec hook, or each driver carries its own. **Recommended: a second entry
point**, because a per-driver copy is how two engines start disagreeing about
what the port means, which is the failure the suite was built to prevent.

## Alternatives

### A — One writer process; the CLI talks to the service

`fdosctl` submits over the wire instead of opening the database. The gap never
exists because there is only ever one writer.

**Rejected.** It makes the operator surface require a running service, a network
and an answer to D2 — and the operator surface is the one thing on the roadmap
that needs none of the three. It also does not survive its own premise: two
instances of the service reintroduce exactly this, so the constraint is
"deploy exactly one process, forever", enforced by nothing.

Worth recording that this is a respectable design elsewhere — a single
transactor is how some event stores define their write path — but those systems
make it the architecture, not a deployment note.

### B — This proposal

### C — Retry with backoff

The caller retries on `ErrNonMonotonicKnowledge`.

**Rejected.** The refusal is not contention over a resource that frees, it is a
statement that this writer's clock reading is now in the past. Retrying is a
busy-wait until the wall clock advances past another process's reading, which
converts a correctness signal into latency and hides the thing it is signalling.
RFC-0014 left the retry contract open and RFC-0015 narrowed it; this widens it
again for no property gained.

### D — The store assigns knowledge time

**Rejected, and rejected twice before.** ADR-0034 recorded the structural
reason: `domain.NewEnvelope` requires `temporal.Coordinates`, which include
knowledge time, so an envelope cannot exist before it is known. A store
assigning it would construct the envelope — moving the one constructor that
makes a fact well-formed outside the domain that defines well-formed.

Restated here because §1 will be mistaken for it, and the distinction is the
whole design: §1 moves the mutual exclusion, not the construction.

### E — Knowledge time becomes a hybrid clock

`max(now, last + ε)` per stream: monotonic by construction, no refusals, no
locks. RFC-0015 called this its least comfortable position and did not propose
it, on the grounds that it changes knowledge time from *"the instant the
application read the clock"* to *"an instant not before that, nudged forward
under write pressure"* — a change to what a recorded number means, in a ledger
built to answer what was known when.

**Still not proposed, and this RFC does not foreclose it.** RFC-0015 left the
signpost that E is where to go if the ceiling binds. §1 does not remove that
ceiling, it relocates it; and it makes E *cheaper* to adopt later, because the
serialisation point becomes an explicit region in the port rather than an
implicit consequence of a mutex. If E is ever taken, it is taken inside
`Serialise`.

### F — Keep SQLite, accept the refusals, document them

**Rejected on the measurement.** 106 of 128 at sixteen processes is not a
documentation problem, and the operator-facing failure — a CLI that will not
start, or an append refused naming a knowledge time the operator never chose —
is not one a note in a README repairs.

## Prior art

Serialising appends per stream is the ordinary shape of an event store. Kafka
assigns offsets at a per-partition leader, so the assignment point and the
serialisation point are the same place by construction — which is the property
§1 restores here and the reason the fix is a region rather than a retry.

Application-level advisory locks are PostgreSQL's documented mechanism for
exactly this: serialising work keyed by an application identifier rather than by
a row, when the thing being protected is not a row. The transaction-scoped
variant releasing on disconnect is the standard defence against a crashed holder,
and is why it is preferred here over the session-scoped one.

The pattern of an embedded engine hiding a concurrency question until a
client/server engine exposes it is old enough to be predictable, and ADR-0035
predicted it in writing. This RFC is that prediction arriving.

## Open questions

1. **The lock key.** `hashtext` and accept collisions, or a bounded shard count
   mirroring ADR-0036's 256? Proposed: mirror ADR-0036, because the reasoning is
   already written and the two mechanisms staying comparable is worth more than
   a marginally lower collision rate.
2. **`SetMaxOpenConns(1)` on SQLite.** With `Serialise` holding the one
   connection for a whole region, readers in the same process now block on
   writers. WAL permits concurrent readers; whether to raise the limit is a
   capacity decision this RFC does not take.
3. **How the PostgreSQL suite runs offline.** `make verify` must pass on a clean
   clone with no network and no tribal knowledge. A driver whose tests need a
   server cannot honour that unconditionally. Proposed: the suite runs when a
   DSN is present and is **skipped loudly and counted** otherwise, never
   silently — with CI supplying one. This needs deciding before the module
   lands, not after.
4. **Whether `libs/ledger-sqlite` joins `go.work`.** Unrelated to this design
   and surfaced by it. Belongs to [#79](https://github.com/FabioCaffarello/fdos/issues/79).

## Consequences

### Easier

- Two processes can write to one ledger, which is what a local stack is.
- The application layer stops owning a concurrency primitive, so
  Constitution §10 holds at the layer boundary rather than approximately.
- The conformance suite starts doing the job ADR-0035 said it was not doing:
  with a second engine that genuinely permits concurrent writers, `Expectation`
  and monotonicity are finally exercised against a store that does not serialise
  anyway.
- Alternative E, if ever needed, has one place to live.

### Harder

- The port grows a method, and every implementation — including any out of tree
  — must provide it.
- The critical section is longer, and on SQLite it is global.
- A second engine is a second thing to keep conformant, a second set of
  pins, and a service the developer environment must provide.
- `make verify`'s offline promise has to be defended explicitly rather than
  inherited.

### Impossible

- Reading the clock outside the serialised region. The callback shape is what
  makes that unavailable rather than discouraged.
- Claiming the multi-process path is untested. Whatever is decided, the cases in
  §Enforcement are the minimum that makes the claim checkable.
