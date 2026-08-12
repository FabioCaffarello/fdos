---
id: ADR-0042
title: PostgreSQL is the second engine, and pgx is its binding
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0042 — PostgreSQL is the second engine, and pgx is its binding

## Context

Records the engine half of
[RFC-0017](../rfc/0017-the-write-path-serialises-in-the-store.md).
[ADR-0041](0041-the-write-path-serialises-in-the-store.md) is the correctness
half and this decision depends on it, not the reverse.

**A second engine was anticipated rather than invented here.**
[ADR-0034](0034-the-ledger-event-store.md) put the conformance suite in the
context module beside the port instead of inside the adapter, and
`libs/ledger/storetest` says why in its own opening lines: *"ADR-0034
anticipated a second engine — one conformance suite, two implementations"*, and
names `libs/ledger-postgres` as the case that would otherwise have to depend on
`libs/ledger-sqlite` merely to be tested. The mechanism has been waiting for the
implementation since M10.

[ADR-0035](0035-the-sqlite-driver-and-its-provenance-risk.md) recorded the gap
that makes waiting expensive:

> SQLite is single-writer, so ADR-0034's `Expectation` and knowledge-time
> monotonicity are exercised against a store that serialises anyway.

Two mechanisms central to the ledger's write path have never been tested against
a store that permits genuine concurrency. One conformant implementation is not a
contract; it is a description of itself.

### What this decision is not

**PostgreSQL is not the fix for the multi-process failure.** ADR-0041 is, and it
works on the engine already shipped: with the write lock held across the clock
read, the measured refusals on SQLite go to zero. Anyone reading this ADR as
"the ledger needed a real database to be correct" has it backwards, and the
inverse error is the dangerous one — adopting PostgreSQL *without* ADR-0041
removes SQLite's accidental cross-process serialisation and returns the write
path to ADR-0036's numbers with nothing in the application able to help.

## Decision

**FDOS adds `libs/ledger-postgres` as a second `app.Store`, bound with
`github.com/jackc/pgx/v5`. `libs/ledger-sqlite` stays and remains the default.**

### Why PostgreSQL, in the three terms that are not "it scales"

1. **Lock granularity.** `pg_advisory_xact_lock` is per key, so concurrent
   streams write concurrently. This matters precisely because ADR-0041 lengthens
   the critical section — it now spans the clock read and, for a derivation, a
   full `Load`. A database-wide lock over a longer region is a platform ceiling.
2. **The local stack puts the file on a volume.** SQLite's cross-process locking
   *is* the filesystem's advisory locking, and SQLite's own documentation is
   explicit that this is unreliable on network filesystems. A container volume
   driver is not something the ledger should have to audit in order to know
   whether its ordering holds. PostgreSQL takes the ledger's correctness out of
   the filesystem's hands.
3. **The workload is already client/server.** A service, an operator CLI and
   every future reader against one store is the shape PostgreSQL is and the
   shape SQLite tolerates.

The advisory lock is **transaction-scoped**, not session-scoped: it is released
by commit, by rollback, and by the connection dying, so a crashed writer cannot
leave a stream locked. The key is a bounded hash of the stream name, mirroring
ADR-0036's 256 shards rather than a lock table keyed by name — stream names are
producer-supplied and unbounded while D2 is open, so a per-name row is an
allocation primitive handed to a stranger. Two unrelated streams occasionally
serialising is throughput, not correctness, and the two engines' reasoning
staying comparable is worth more than a marginally lower collision rate.

### Why pgx

`pgx/v5` is pure Go, so `CGO_ENABLED=0` — pinned in the Makefile by ADR-0035 to
keep byte reproducibility from becoming accidental — continues to hold with no
new argument. It is the actively developed driver, and `lib/pq`, the only other
candidate with real deployment history, is in maintenance mode by its own
maintainers' statement and points new work at pgx. Choosing a driver whose
maintainers say to use a different one requires a reason this repository does
not have.

**The dependency cost is recorded when the module lands, not estimated here.**
ADR-0035 recorded 245 MB and ten modules because it had measured them, and an
ADR that guesses the number it exists to be honest about is worse than one that
says the measurement is owed.

### The schema, and the version marker

Same shape as `libs/ledger-sqlite`: one table keyed `(stream, sequence)`, an
index over `(stream, effective_from, knowledge, sequence)`, temporal columns as
**integer nanoseconds** since the Unix epoch — because byte order must equal
chronological order and
[ADR-0040](0040-encoding-integrity-and-the-fdos-root-namespace.md) settled that
a rendered timestamp does not give it.

PostgreSQL has no `PRAGMA user_version`, so the encoding marker is an explicit
single-row table, carrying the same rule ADR-0040 established: a database whose
encoding this build cannot order correctly is **refused**, naming the migration,
rather than opened and answered from. There is no legacy case today — a fresh
store starts at the current encoding — and the mechanism exists so that the next
encoding change has one, which is the position ADR-0040 wished it had been in.

### Running the suite without a server

`make verify` must pass on a clean clone with no network and no tribal
knowledge. A driver whose tests need a database cannot honour that
unconditionally, and the two ways around it are both refused:

- **A container-based harness** pulls an image, which is an opaque build input
  of exactly the class ADR-0035 rejected 103 KB of.
- **An embedded PostgreSQL** downloads a prebuilt server binary, which is the
  same objection several orders of magnitude larger.

So: the PostgreSQL conformance run executes when `FDOS_POSTGRES_DSN` is set and
is **skipped loudly and counted** otherwise — reported in `make verify`'s
output, never silent. CI supplies a DSN, so the pull request gate covers it.

## Consequences

### Positive

- The conformance suite starts doing the job it was built for. `Expectation` and
  knowledge-time monotonicity are finally exercised against a store that does
  not serialise anyway, which is the gap ADR-0035 wrote down and could not close
  alone.
- The local stack's ordering stops depending on a volume driver's advisory
  locking semantics.
- Per-stream write concurrency, which is what makes ADR-0041's longer critical
  section affordable.
- Two implementations make the port's meaning testable rather than descriptive,
  which is the argument ADR-0013 made for separate adapter modules in the first
  place.

### Negative

- **`make verify` on a clean clone no longer covers every engine.** The skip is
  counted and printed, so the gate is honest about what it did not run — but a
  green local run is now a slightly weaker prediction of a green pipeline, and
  that was a property the repository advertised without qualification.
- **A service enters the developer environment.** SQLite is a library; this is a
  process to run, a version to pin, and a thing that can be misconfigured in
  ways a test does not see.
- **Two engines, no migration path between them.** Facts do not move from a
  SQLite ledger to a PostgreSQL one, and building an export is not in scope
  here. A stack that switches engines starts empty, which is acceptable only
  while no stack holds anything worth keeping — and that window closes the first
  time one does.
- **A second dependency tree** enters the module graph of anyone importing the
  new module, against a repository whose stated first priority is build-input
  integrity. It is confined to a module nobody has to import, which is precisely
  why ADR-0013 puts adapters in their own modules.
- What would make this wrong later: if the local stack turns out to run one
  writer process after all, ADR-0041 alone covers it and this engine is carrying
  cost for a concurrency nobody exercises.

### Enforcement

| Property | Rung | Mechanism |
|---|---|---|
| The engine satisfies `app.Store` | 1 | it does not compile otherwise |
| It agrees with SQLite and memory on the port's meaning | 3 | `storetest.Run`, shared suite, no per-driver copy |
| Writers in separate processes do not refuse each other | 3 | the process-spawning case ADR-0041 requires |
| An advisory lock is released when a writer dies | 3 | kill the connection mid-region; the next writer proceeds |
| A database at an unorderable encoding is refused | 3 | a fixture at the older marker, asserting the refusal names the migration |
| The suite was actually run | 5 → 3 | printed and counted when skipped; CI supplies a DSN |

The last row is the honest one. Locally it is documentation — a line of output a
human may ignore. In CI it is rung 3. Climbing it for a clean clone would
require an opaque build input, which is the trade this decision declines.

## Alternatives considered

- **Stay on SQLite only.** Viable for correctness once ADR-0041 lands, and it is
  why that ADR is separable. It loses on the volume-locking argument, on lock
  granularity, and on leaving ADR-0035's conformance gap open indefinitely —
  a suite built for two implementations that only ever has one is a suite
  describing its single implementation.
- **MySQL or MariaDB.** `GET_LOCK` is session-scoped, so a crashed writer holds
  it until the connection times out — the exact failure the transaction-scoped
  advisory lock avoids. No compensating advantage was identified.
- **An embedded key-value store — bbolt, Badger, Pebble.** Single-writer or
  single-process by design, so they reproduce the property being moved away from
  while discarding the one thing SQLite gives back, which is a queryable
  as-of index the store does not have to hand-roll.
- **A hosted PostgreSQL service.** Refused for the developer environment on the
  offline test: the ledger must be developable and testable with every external
  provider unreachable. A hosted instance is a deployment choice an adopter may
  make, and it changes nothing here.
- **`lib/pq` instead of pgx.** Loses on being in maintenance mode and pointing
  at pgx itself.

## Notes

**Cross-repository.** `fdos-connectors` pins `libs/contracts` and is untouched
by this. What does reach it is the local stack: the compose file that composes
the published FDOS images now needs a PostgreSQL service beside them, which is
an issue on that repository when the images publish, not a contract change.

**Not decided here.** Replication, failover, pooling policy, and any topology
beyond a single instance. Also not decided: whether `libs/ledger-sqlite` keeps
`SetMaxOpenConns(1)` under ADR-0041's longer region — a capacity question left
open there.

**Owed.** The measured dependency count and module-cache footprint, recorded in
this ADR's follow-up when the module lands, in the form ADR-0035 used.
