---
id: RFC-0014
title: The ledger event store
status: Draft
date: 2026-08-06
authors:
  - "@FabioCaffarello"
---

# RFC-0014 — The ledger event store

## Summary

Facts must survive process exit, bitemporal as-of reads must work against
storage, and the result must be reproducible. Proposes the storage model, the
write path, and a technology, in answer to the M10 gate
([PR #58](https://github.com/FabioCaffarello/fdos/pull/58)).

This cannot be settled by an ADR alone because the write path is not a storage
detail. The current port makes concurrent appends lose facts, and the fix
interacts with a decision M9 Track A just took: `MintIdentity` resolves before
minting, and *"resolve, then append"* is only safe if the append can refuse when
the read went stale.

## Motivation

### What breaks without this

Everything the ledger asserts is currently true only until the process exits.
`libs/ledger/adapters/memory` says so itself — *"deliberately not a
general-purpose store… no query language, no index and no persistence, because
the slice exists to prove the boundaries hold, not to be a database."*

### What is already broken, and was measured

The M10 gate measured a defect and
[`e08c4d7`](https://github.com/FabioCaffarello/fdos/pull/58) closed the visible
half: two callers loading the same stream, each appending a different fact, each
saving at the same length — both saves returned `nil`, one fact was gone, and
**both facts carried the same `Ref`**.

That last part is the load-bearing one. `Ref` is `{stream, sequence}` and
`domain.Stream.Append` assigns the sequence. Two callers appending from the same
base therefore *both* compute `acct-1#1`. A reference handed to a caller — or
already recorded as a `DerivationRef` input — silently re-points.

**The sequence is assigned in the wrong place.** It is assigned by a pure value
type that cannot know what any other writer is doing, when it can only be
assigned correctly at the point where writes serialise. That is the store.

### The interaction with ADR-0033, which is not obvious

`Ledger.MintIdentity` resolves first and refuses if the claim already resolves.
Between that read and the append, another mint can land. Without a precondition
on the append, the result is two mints for one claim — the exact duplication
ADR-0033 refuses, arriving through a race rather than through a caller error.

`e08c4d7` accidentally hardened this: the second save is now rejected rather
than silently applied, so the race produces an error instead of a duplicate. But
it is rejected for the *wrong reason* — "the stream did not grow" rather than
"what you read has changed" — and only when both writers append exactly one
fact.

**So the write path is a correctness question about identity, not a performance
question about storage.**

### Retrofittable?

The storage engine is. The **write path is not, cheaply**: once callers depend on
`load → append → save`, changing it moves every call site, and every fact
written under the old path was written without a precondition. Deciding it now
costs one module's API; deciding it after `apps/` exists costs a migration.

## Design

### 1. The write path: append at the store, with an optional precondition

The port becomes:

```go
type Store interface {
    Load(ctx context.Context, name string) (domain.Stream, error)

    // Append records one fact and returns the ref the store assigned.
    Append(
        ctx context.Context,
        name string,
        expect Expectation,
        envelope domain.Envelope,
        kind domain.Kind,
        payload domain.Payload,
    ) (domain.Ref, error)
}

// Expectation is what the caller believes it read.
type Expectation struct{ /* Any() | AtLength(n uint64) */ }
```

`Save` goes. There is no operation that writes a whole stream, because there is
no legitimate caller for one: a stream is only ever extended.

**The store assigns the sequence**, because the store is where writes serialise.
`domain.Stream.Append` stays — it is how a stream is built in a test or a
projection — but it is no longer how a fact reaches storage, so two callers can
no longer compute the same `Ref`.

**`Expectation` exists because some appends depend on a prior read and some do
not.**

| Caller | Expectation | Why |
|---|---|---|
| `AcceptHoldingClaim` | `Any()` | A claim is admitted on its own merits; nothing it read can go stale |
| `MintIdentity` | `AtLength(n)` | It resolved against the stream at length *n*; a mint that landed since may already answer the claim |
| `ObserveClaimedHolding` | `AtLength(n)` | It derived from mints visible at *n* |
| `CorrectFact` | `AtLength(n)` | It read the fact being corrected |

A violated expectation returns **`ErrStaleRead`**, naming the expected and actual
lengths. That is deliberately distinct from *"you tried to rewrite history"*: one
means retry after re-reading, the other means a bug. A caller that cannot tell
them apart will retry the second forever.

### 2. Knowledge time: the app assigns, the store refuses non-monotonic

ADR-0009 requires knowledge time to be machine-assigned, monotonic per stream and
never caller-supplied. Today the app assigns it from an injected `Clock`, which
is correct for one process and unenforced across two.

**Proposal: the app keeps assigning, and the store rejects an append whose
knowledge time is not strictly greater than the stream's last.**

The reason for this split rather than the tidier "store assigns" is structural
and worth stating, because it is the kind of thing that looks like an oversight
later. `domain.NewEnvelope` requires `temporal.Coordinates`, and coordinates
include knowledge time — so an envelope cannot be constructed before knowledge
time is known. If the store assigned it, the store would have to construct the
envelope, moving envelope construction out of the domain and into an adapter.
That is a worse trade than the one below.

**The cost, stated plainly:** with two processes and two clocks, a writer whose
clock lags has a *legitimate* append rejected. The rejection is honest — FDOS
genuinely cannot order those two facts — but it is an operational failure mode
that does not exist today, and it converts clock skew into write unavailability.

The alternative, and the reason it is not proposed, is in **Alternatives**.

### 3. Bitemporal reads: an index, and it is derived state

`Stream.VisibleAt` filters and sorts the whole stream in memory. Against storage
that is a full scan per query.

Proposal: the store indexes `(stream, effective_from, knowledge_time, sequence)`
— the total order ADR-0009 already defines — so an as-of read is a range scan.

**The index is derived state, and Constitution §1 governs it:** it is not a
second source of truth. Concretely that means it must be **rebuildable from the
facts alone**, and a test must prove that rebuilding produces byte-identical
answers. An index that cannot be rebuilt is a second copy of the ledger that
will eventually disagree with the first, with nothing to say which is right.

### 4. Serialisation: reuse `libs/ledger-wire`

A fact is persisted as the protobuf `libs/ledger-wire` already encodes.

The alternative — a storage-native encoding — means two encodings of every fact
and a conformance suite proving they agree, which is the cost M7 paid once
already for domain-versus-wire and should not pay twice.

**Two costs this inherits, both real:**

- The store pins `libs/ledger-wire` and, through it, `libs/contracts`. Storage is
  now coupled to the *consumer-facing* versioning surface. ADR-0011's
  upcast-on-read is what keeps old rows readable across a contract release, and
  this makes that mechanism load-bearing for storage rather than only for
  transport.
- **`kernel.v1.DerivationRef` carries only `content_hash`.** A derivation's
  method and parameters do not survive the wire, so persisting through it does
  not persist them. For a mint that means *"which canonicalisation ruleset
  minted this"* is **not durable** — which ADR-0033 accepted knowingly on the
  grounds that `BornFrom` answers the question that matters. If storage is ever
  expected to answer it directly, that is a `contracts` change and a release,
  not a storage decision.

### 5. Technology: SQLite through a pure-Go driver

Proposed, and the criterion that discriminates is FDOS-specific rather than
general.

**Reproducibility.** `make repro-check` builds every command twice and compares
digests. Measured for this RFC: `CGO_ENABLED` is `1` by default here, and the
modules currently depend on **zero** cgo packages — the tree is cgo-free by
accident rather than by rule. A cgo SQLite driver would make the build depend on
the host C toolchain and put byte-reproducibility at the mercy of a system
compiler. A pure-Go driver keeps the property FDOS already has.

**So the proposal is a pure-Go SQLite driver, and `CGO_ENABLED=0` becomes a
pinned build setting** — turning an accident into a rule, which is cheap now and
expensive after the first cgo dependency lands.

| Option | For | Against |
|---|---|---|
| **SQLite, pure-Go driver** | One file, no server, transactional, real indexes, trivial test story, offline by construction | Single-writer; a large dependency to audit |
| SQLite, cgo driver | Mature, fastest | Breaks the cgo-free property and threatens `repro-check` |
| PostgreSQL | Real concurrency, operationally familiar | A running server at test time; a runtime dependency for an open-core library |
| Embedded KV (bbolt, pebble) | Pure Go, transactional, small | The bitemporal index is hand-rolled — writing half a database badly |
| Append-only files | No dependency | The index, crash-safety and atomicity are all hand-rolled |

**Offline boundary test:** the store reads the ledger and nothing else. Every
option above passes; a hosted database service would not, because replay would
depend on that service's availability and its history.

It lives in **`libs/ledger-sqlite`** per ADR-0013 — a context module stays
dependency-light, and a driver in `libs/ledger` would land in the `go.sum` of
every consumer of `libs/ledger/domain`.

### 6. Reproducibility: replay equivalence

`repro-check` today asserts that builds are byte-reproducible. The storage
analogue:

> Replaying a fact stream into a fresh store and reading at the same coordinate
> produces the same answer, with the same derivation content addresses, as
> reading the original.

That is a test rather than a new `make` target, because it needs a store and
`repro-check` deliberately needs only a compiler.

### Not covered

Transport (M11), `apps/`, any consumer-facing query surface, multi-node
replication, backup and restore, and the retention question. Also not covered:
whether `Load` should stay whole-stream at all — see **Open questions**.

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| A whole-stream write is unrepresentable | 1 | `Save` is removed from the port; there is no operation to call |
| Two writers cannot compute the same `Ref` | 1 | the sequence is assigned by the store, not by a pure value |
| A stale read cannot append silently | 3 | test: append with a violated `Expectation` returns `ErrStaleRead` |
| Knowledge time is monotonic per stream | 3 | test: an append with a non-increasing knowledge time is refused |
| The index is not a second source of truth | 3 | test: rebuild from facts, assert identical answers |
| Builds stay reproducible | 2 | `CGO_ENABLED=0` pinned in the Makefile; `repro-check` already compares digests |
| The store reads nothing but the ledger | 6 | review; no mechanism proposed, and a store that dialled out would pass every check above |

The last is the honest weak point.

## Alternatives

**Keep `Save(stream)` and add an expected-length parameter.** The smallest
change: no new method, callers keep `load → append → save`. Rejected because it
leaves the sequence assigned by `domain.Stream.Append`, so two writers still
compute the same `Ref` and the store cannot correct it without rewriting refs
the caller has already been handed.

**The store assigns knowledge time.** Structurally cleaner — the serialisation
point assigns the ordering — and immune to clock skew. Rejected because
`domain.NewEnvelope` requires coordinates, so the store would have to build the
envelope, moving envelope construction into an adapter and putting the one
constructor that makes a fact well-formed outside the domain that defines
well-formed. Worth revisiting if the skew cost proves worse in practice than the
layering cost; recorded so that revisit has a starting point.

**No precondition at all — let the store assign sequences and accept every
append.** Appealing: no conflicts, no retries, and no fact is ever lost.
Rejected because it silently breaks `MintIdentity`. Resolve-then-append without
a precondition is a TOCTOU race whose output is two mints for one claim, and
ADR-0033 exists to prevent exactly that. The absence of conflicts would be
purchased by letting a real one through.

**PostgreSQL now, SQLite never.** Rejected as premature: it imposes a running
server on every consumer of an open-core library and on every test run, to buy
multi-writer concurrency that nothing yet needs. The port above is engine-
agnostic, so this is reversible — which is the argument for choosing the smaller
thing first.

**Storage-native encoding.** Rejected in §4.

## Prior art

Event-sourced systems converge on append-with-expected-version — EventStoreDB,
Marten, and the Kafka transactional producer all expose the same shape, and for
the same reason: optimistic concurrency is the only way a command that read
state can safely write. The systems that omit it develop a reconciliation job.

The bitemporal side has fewer good examples. XTDB and the SQL:2011 temporal
tables both index the two axes as ordinary columns and rely on the query planner,
which is the argument for using a database rather than hand-rolling the index.

The failure this design is shaped to avoid is the one where a store's sequence
number is assigned by the writer: every system that has done it has eventually
discovered two records with one identifier, and the recovery is manual.

## Open questions

- **Does `Load` stay whole-stream?** It has the same O(history) problem as
  `Save` had, and a projection that only needs a coordinate range does not need
  every fact. Deliberately left open: the write path is the correctness
  question, the read path is a performance question, and bundling them would
  make this RFC undecidable.
- **What is the retry contract for `ErrStaleRead`?** Whether the application
  layer retries, or the caller does, is an operational decision this does not
  make.
- **Does the clock-skew rejection in §2 need an operational answer** — a leader,
  a bounded skew tolerance — before more than one writer is deployed? Nothing
  deploys two writers today, so this is not yet forced.
- **Which pure-Go SQLite driver, and does its dependency footprint pass the
  supply-chain posture?** Named as a decision, not answered here: it is the
  first heavy dependency in the repository and deserves the audit rather than a
  recommendation in passing.

## Consequences

**Easier.** Facts survive process exit. Two writers cannot silently lose a fact
or collide on a `Ref`. A stale read fails loudly and retryably. Bitemporal reads
stop being full scans.

**Harder.** Every write call site changes. The application layer must decide, per
use case, what it expects — which is the point, but it is new thinking at each
site. Clock skew becomes a write failure. The repository acquires its first
heavy dependency and the audit that comes with it.

**Impossible.** Writing a whole stream. Two writers computing the same `Ref`.
Appending on a read that has gone stale, silently.
