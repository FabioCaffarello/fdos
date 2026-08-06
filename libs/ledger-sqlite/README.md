---
directory: ledger-sqlite
purpose: The durable ledger event store — a SQLite-backed app.Store, and the adapter tests that prove a fact survives process exit.
owner: "@FabioCaffarello"
allowed:
  - An app.Store implementation backed by SQLite through modernc.org/sqlite
  - Schema, migrations, and the index over the total order ADR-0009 defines
  - The driver dependency and what it transitively requires
  - The shared store conformance suite, run against this implementation
  - Adapter tests for durability, crash-safety and index rebuild
forbidden:
  - Business rules or financial calculations — those live in libs/ledger/domain
  - Storing derived state: a position, a balance, or a resolution result
  - An index that cannot be rebuilt from the facts alone
  - Consulting anything outside the ledger to answer a read
  - cgo, or any dependency that requires it
  - Assigning knowledge time, or accepting one from a caller
---

# libs/ledger-sqlite

The store that makes the ledger outlive the process. Until this module exists,
everything FDOS asserts is true only until the program ends.

A separate module, not a package inside `libs/ledger`, and the reason is
dependency resolution rather than taste (ADR-0013). Go resolves dependencies per
module: a driver inside `libs/ledger` would land in the `go.sum` of every
consumer that imports `libs/ledger/domain`, including consumers that never touch
storage.

## What it implements

`app.Store` as ADR-0034 defines it — `Load`, and `Append` carrying an
`Expectation`. There is no whole-stream write, because a stream is only ever
extended and the operation that wrote one entire is what lost facts.

Three obligations this module inherits, each of which is a test below rather
than a promise:

| Obligation | From |
|---|---|
| The **store** assigns the sequence, so two writers cannot compute one `Ref` | ADR-0034 |
| A stale read is refused, not applied | ADR-0034 |
| Knowledge time is monotonic per stream | ADR-0009 |

## The index is derived state

The store indexes `(stream, effective_from, knowledge_time, sequence)` so an
as-of read is a range scan rather than a full scan.

Constitution §1 governs it: **the index is not a second source of truth.** It
must be rebuildable from the facts alone, and a test proves rebuilding gives
identical answers. An index that cannot be rebuilt is a second copy of the ledger
that will eventually disagree with the first, with nothing to say which is right.

## The driver, and the risk that came with it

`modernc.org/sqlite`, chosen on a measured audit and recorded in ADR-0035. That
ADR also records what the audit could not fix: **the driver cannot be
independently re-derived from upstream SQLite.** `go.sum` pins what was received,
not that what was received is SQLite. Accepted, unmitigated, and stated here so
that nobody reading this module assumes otherwise.

`CGO_ENABLED=0` is pinned in the Makefile. A cgo dependency would make the build
depend on the host C toolchain and put `make repro-check` at the mercy of a
system compiler.

## Test plan

Two layers. The first is not specific to SQLite and should not live here.

### The shared store conformance suite

ADR-0034 anticipated a second engine — *"one conformance suite, two
implementations"* — so the suite that defines what implementing `app.Store`
*means* cannot live inside either adapter, or Postgres would depend on SQLite to
be tested.

**Proposed home: `libs/ledger/storetest`**, an exported package in the context
module. It needs `domain` and `app` and nothing else, so it adds no dependency
weight, and it is the same shape as the standard library's `testing/fstest`. The
argument for putting it in the public API rather than hiding it: `app.Store` is
public API, so the suite that says what implementing it means is public API too.

Confirm that placement when the suite is written; it is the one design choice in
this plan that is not already settled by an ADR.

The suite takes a factory and runs every case against it — the in-memory store
included, so the two implementations are held to one definition:

| # | Case | Proves |
|---|---|---|
| 1 | Two appends receive sequences 1 and 2, never equal | the measured defect stays fixed |
| 2 | `Load` of an unknown stream is `ErrStreamNotFound` | "we know nothing" ≠ "we hold nothing" |
| 3 | The first append creates the stream | a stream is the facts in it |
| 4 | `AtLength(n)` succeeds when the stream is at *n* | the precondition does not fire spuriously |
| 5 | `AtLength(n)` is `ErrStaleRead` otherwise, **and the fact is not appended** | a refusal refuses |
| 6 | `Any()` succeeds regardless of what landed since | admission is never blocked by another writer |
| 7 | An equal or earlier knowledge time is `ErrNonMonotonicKnowledge` | ADR-0009's axis is ordered |
| 8 | Facts reload with identical refs, envelopes and payloads | nothing is lost in the round trip |
| 9 | An as-of read matches the same projection computed in memory | the index does not change answers |
| 10 | Concurrent appends under `-race`: *N* goroutines, *N* facts, refs 1…*N* | the serialisation point serialises |

Case 5's second clause is the one worth writing carefully. A store that returns
the error *and* appends anyway passes a naive version of this test, and that is
precisely the failure the M10 gate measured in a different disguise.

### Adapter tests, which belong here

| # | Case | Proves |
|---|---|---|
| 11 | Append, close the database, reopen it, read the facts back | **the point of the module** — and the case the in-memory store cannot satisfy |
| 12 | Drop the index, rebuild from the facts, answers are identical | the index is derived, not a second truth |
| 13 | Replay a stream into a fresh database: same answers **and same derivation content addresses** | ADR-0034's reproducibility clause |
| 14 | A transaction interrupted mid-append leaves no partial fact | crash-safety — the gap ADR-0035 named as unaudited |
| 15 | The schema rejects a duplicate `(stream, sequence)` | the sequence is unique at the storage layer, not only in Go |

Case 14 is the one ADR-0035 flagged as the thing it would want tested before
trusting this with a ledger, and it is the hardest to write portably. At minimum
it must assert the durability settings the driver is opened with, rather than
assuming a default.

Case 13 is the storage analogue of `make repro-check`. It stays a test rather
than a `make` target because it needs a database, and `repro-check` deliberately
needs only a compiler.

### Not proven by any of these

**That the store reads the ledger and nothing else.** A store that dialled out
would pass every case above. ADR-0034 records it at rung 6, and no mechanism is
proposed here — saying so is better than implying coverage that does not exist.

## Release chain

This module pins `libs/ledger`, so it cannot compile until the port change is
released as `libs/ledger v0.4.0` (ADR-0004). `make verify` runs each module with
`GOWORK=off`, which is what makes that a build failure rather than something
`go.work` hides until CI.
