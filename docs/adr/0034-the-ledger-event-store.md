---
id: ADR-0034
title: The ledger event store appends with a precondition, on SQLite
status: Accepted
date: 2026-08-06
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0034 — The ledger event store appends with a precondition, on SQLite

## Context

Records what [RFC-0014](../rfc/0014-the-ledger-event-store.md) settled, in answer
to the M10 gate ([PR #58](https://github.com/FabioCaffarello/fdos/pull/58)).

Facts did not survive process exit. Everything the ledger asserted was true only
until the process ended, and `libs/ledger/adapters/memory` says so in its own
package comment.

Two findings made this a decision rather than a technology selection.

**The sequence was assigned in the wrong place.** `Ref` is `{stream, sequence}`
and `domain.Stream.Append` assigns the sequence — a pure value that cannot know
what any other writer is doing. Measured at the gate: two callers appending from
the same base both computed `acct-1#1`, so a `Ref` already handed to a caller,
or already recorded as a `DerivationRef` input, silently re-pointed. A sequence
can only be assigned correctly where writes serialise.

**The write path is a correctness question about identity.** `MintIdentity`
resolves before minting (ADR-0033). Resolve-then-append without a precondition is
a time-of-check-to-time-of-use race whose output is two mints for one claim —
precisely what ADR-0033 refuses, arriving through a race rather than a caller
error. [`e08c4d7`](https://github.com/FabioCaffarello/fdos/pull/58) closed the
silent half of the loss, but rejects for the wrong reason — *"the stream did not
grow"* rather than *"what you read has changed"* — and only when both writers
append exactly one fact.

Constitution §1 (Financial Truth), §4 (Immutable Ledger) and §9
(reproducibility) are what is at stake.

## Decision

### The port appends, and the store assigns the sequence

`Save` is removed. There is no operation that writes a whole stream, because
there is no legitimate caller for one: a stream is only ever extended.

```go
type Store interface {
    Load(ctx context.Context, name string) (domain.Stream, error)
    Append(
        ctx context.Context,
        name string,
        expect Expectation,
        envelope domain.Envelope,
        kind domain.Kind,
        payload domain.Payload,
    ) (domain.Ref, error)
}
```

**The store assigns the sequence**, because the store is where writes serialise.
`domain.Stream.Append` remains — it is how a stream is built in a projection or a
test — but it is no longer how a fact reaches storage, so two callers can no
longer compute the same `Ref`.

### An append carries what the caller believes it read

`Expectation` is `Any()` or `AtLength(n)`. Some appends depend on a prior read
and some do not:

| Caller | Expectation | Why |
|---|---|---|
| `AcceptHoldingClaim` | `Any()` | a claim is admitted on its own merits; nothing it read can go stale |
| `MintIdentity` | `AtLength(n)` | it resolved at length *n*; a mint since may already answer the claim |
| `ObserveClaimedHolding` | `AtLength(n)` | it derived from mints visible at *n* |
| `CorrectFact` | `AtLength(n)` | it read the fact being corrected |

A violated expectation returns **`ErrStaleRead`**, naming expected and actual.
That is deliberately distinct from an attempt to rewrite history: one means
re-read and retry, the other means a bug, and a caller that cannot tell them
apart will retry the bug forever.

### Knowledge time stays app-assigned; the store refuses non-monotonic

ADR-0009 requires knowledge time to be machine-assigned, monotonic per stream and
never caller-supplied. The application layer keeps assigning it from the injected
`Clock`, and the store rejects an append whose knowledge time is not strictly
greater than the stream's last.

The reason for this split rather than "the store assigns" is structural, and is
recorded because it will look like an oversight later: `domain.NewEnvelope`
requires `temporal.Coordinates`, which include knowledge time, so an envelope
cannot exist before knowledge time is known. A store that assigned it would have
to construct the envelope, moving the one constructor that makes a fact
well-formed outside the domain that defines well-formed.

### The index is derived state

The store indexes `(stream, effective_from, knowledge_time, sequence)` — the
total order ADR-0009 already defines — so an as-of read is a range scan rather
than a full scan.

Constitution §1 governs it: the index is **not** a second source of truth. It
must be rebuildable from the facts alone, and a test proves that rebuilding
produces identical answers. An index that cannot be rebuilt is a second copy of
the ledger that will eventually disagree with the first, with nothing to say
which is right.

### Facts are persisted through `libs/ledger-wire`

A fact is stored as the protobuf the wire codec already produces. A
storage-native encoding would mean two encodings of every fact and a conformance
suite proving they agree — a cost M7 paid once for domain-versus-wire and should
not pay twice.

### SQLite, through a pure-Go driver, with `CGO_ENABLED=0` pinned

The store is **`libs/ledger-sqlite`**, a separate module per ADR-0013: a context
module stays dependency-light, and a driver inside `libs/ledger` would land in
the `go.sum` of every consumer of `libs/ledger/domain`.

The discriminating criterion was reproducibility, measured rather than assumed:
`CGO_ENABLED` is `1` by default in this repository and the modules depend on
**zero** cgo packages, so the tree is cgo-free by accident rather than by rule. A
cgo SQLite driver would make the build depend on the host C toolchain and put
`make repro-check` at the mercy of a system compiler.

So the driver is pure Go, and **`CGO_ENABLED=0` becomes a pinned build
setting** — turning an accident into a rule while it is still cheap, which it
will not be after the first cgo dependency lands.

### Reproducibility is asserted as replay equivalence

> Replaying a fact stream into a fresh store and reading at the same coordinate
> produces the same answer, with the same derivation content addresses, as
> reading the original.

A test rather than a new `make` target: it needs a store, and `repro-check`
deliberately needs only a compiler.

## Consequences

### Positive

- Facts survive process exit, which is the whole milestone.
- **Two writers cannot compute the same `Ref`.** The sequence moves to the only
  place that can assign it correctly.
- A stale read fails loudly and retryably instead of producing a duplicate mint.
- Bitemporal reads stop being full scans.
- The cgo-free property becomes enforced rather than accidental, which protects
  §9 reproducibility against a dependency nobody reviewed for it.

### Negative

- **Every write call site changes**, including the M9 Track A use cases written
  one milestone ago. The application layer must now decide, per use case, what
  it expects — which is the point, and it is new thinking at each site.
- **Clock skew becomes write unavailability.** With two processes and two
  clocks, a writer whose clock lags has a *legitimate* append rejected. The
  rejection is honest — FDOS genuinely cannot order those two facts — but it is
  an operational failure mode that does not exist today. **This was the trade
  flagged for pushback when the RFC was presented, and it is accepted as
  proposed.** If it proves worse in practice than the layering cost it buys,
  the alternative and its starting point are recorded in RFC-0014.
- **`Load` stays whole-stream**, with the same O(history) cost `Save` had. This
  ADR does not fix it: the write path is a correctness question and the read
  path a performance one, and bundling them would have made the decision
  undecidable. It is recorded as open rather than settled by omission.
- **The store inherits the consumer-facing versioning surface** through
  `libs/ledger-wire` and `libs/contracts`. ADR-0011's upcast-on-read becomes
  load-bearing for storage, not only for transport.
- **`kernel.v1.DerivationRef` carries only `content_hash`**, so persisting
  through the wire does not persist a derivation's method or parameters. *Which
  canonicalisation ruleset minted an identity* is therefore not durable — which
  ADR-0033 accepted knowingly, and which this decision does not change. Making
  it durable is a `contracts` release, not a storage change.
- **The first heavy dependency in the repository**, with the audit that implies.
- Two coordinated releases: `libs/ledger` for the port, then `libs/ledger-sqlite`
  (ADR-0004).

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| A whole-stream write is unrepresentable | 1 | `Save` is removed; there is no operation to call |
| Two writers cannot compute the same `Ref` | 1 | the sequence is assigned by the store, not by a pure value |
| A stale read cannot append silently | 3 | test: a violated `Expectation` returns `ErrStaleRead` |
| Knowledge time is monotonic per stream | 3 | test: a non-increasing knowledge time is refused |
| The index is not a second source of truth | 3 | test: rebuild from facts, assert identical answers |
| Builds stay reproducible | 2 | `CGO_ENABLED=0` pinned in the Makefile; `repro-check` compares digests |
| The store reads nothing but the ledger | **6** | review; a store that dialled out would pass every check above |

The last is the honest weak point.

## Alternatives considered

Recorded in full in RFC-0014. The three that were closest:

**Keep `Save(stream)` and add an expected-length parameter.** The smallest
change; callers keep `load → append → save`. Rejected because it leaves the
sequence assigned by `domain.Stream.Append`, so two writers still compute the
same `Ref` and the store cannot correct it without rewriting refs already handed
out.

**No precondition at all — the store assigns sequences and accepts every
append.** No conflicts, no retries, and no fact ever lost. Rejected because it
silently breaks `MintIdentity`: the absence of conflicts would be purchased by
letting a real one through, producing two mints for one claim.

**PostgreSQL.** Rejected as premature: it imposes a running server on every
consumer of an open-core library and on every test run, to buy multi-writer
concurrency nothing yet needs. The port is engine-agnostic, so this is
reversible — which is the argument for choosing the smaller thing first.

Also rejected: a cgo SQLite driver, a storage-native encoding, and having the
store assign knowledge time.

## Notes

Accepted by @FabioCaffarello. The technology choice was accepted explicitly; the
knowledge-time trade in §"Negative" and the decision to leave `Load` whole-stream
were accepted as RFC-0014 proposed them.

Open and deliberately not decided here:

- whether `Load` stays whole-stream;
- the retry contract for `ErrStaleRead` — whether the application layer or the
  caller retries;
- whether clock skew needs an operational answer (a leader, a bounded tolerance)
  before more than one writer is deployed; nothing deploys two today;
- **which** pure-Go SQLite driver, and whether its dependency footprint passes
  the supply-chain posture. It is the first heavy dependency in the repository
  and gets the audit rather than a recommendation in passing.
