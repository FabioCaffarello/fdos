---
directory: ledger
purpose: The ledger bounded context — facts, streams, claims, resolution, and the position projection derived from them.
owner: "@FabioCaffarello"
allowed:
  - domain/ — pure rules over kernel types
  - app/ — use cases and the ports they depend on
  - adapters/ — pure adapters only, such as in-memory stores and injected clocks
  - Imports of libs/kernel
forbidden:
  - Importing another bounded context's domain or app (ADR-0013)
  - Any dependency needing a driver, broker, browser or SDK — those live in libs/ledger-<tech>
  - Storing a position, balance or any derived state; they are projections (ADR-0007)
  - Clocks, I/O, concurrency or binary floating point in domain/
  - Mutating or deleting an appended fact — corrections are new facts
---

# libs/ledger

The first bounded context, and the vertical slice that validated everything
above it. Facts are appended and never amended; every figure a caller receives
is derived from them, with the trace that produced it.

| Layer | Holds | May import |
|-------|-------|------------|
| `domain/` | Facts, streams, claims, resolution, the position projection | `kernel` |
| `app/` | Use cases and the ports they depend on | `kernel`, own `domain` |
| `adapters/` | In-memory store, injected clock | `kernel`, own `domain`, own `app` |

The dependency rule is ADR-0013's and is enforced by the `layering` analyser,
not by review.

## Infrastructure lives in another module

`adapters/` here holds *pure* adapters only. Anything needing a driver, broker
or SDK belongs in `libs/ledger-<tech>`, and the reason is dependency resolution
rather than taste.

Go resolves dependencies per module, not per package. An `adapters/postgres`
inside this module would put the Postgres driver into the module graph and
`go.sum` of every consumer importing `libs/ledger/domain` — including consumers
that import none of it. That would make Constitution §10 true at the package
level and false at the level that decides what a consumer is actually coupled
to.

## State is never stored

A position is a projection over a fact stream at an as-of coordinate, computed
on demand. There is no table of positions to fall out of step with the facts,
because a stored balance is a second source of truth that will disagree with the
first and give no indication which is wrong.

`Explained[Position]` or nothing at all: a projection that cannot explain itself
does not compile (ADR-0012).

## Claims, mints and resolution

A connector cannot know an `EntityId` and must not mint one — deriving identity
from a ticker makes the ticker the primary key, and a reused ticker then merges
two instruments silently inside an append-only ledger (ADR-0007).

So a connector emits a **claim**; minting an identity is itself a **fact**; and
resolution is a **derivation recorded in the ledger** rather than a
precondition of appending (ADR-0022). `Resolve`, `MintFor` and
`DeriveHoldingObserved` are that path.

**Nobody notices an unresolved claim yet.** Claims can accumulate with no
`HoldingObserved` derived and nothing reporting it — a connector can publish
faithfully into silence. Who is told, and how, is operational and undecided; it
is recorded in [`../../docs/blocked.md`](../../docs/blocked.md) under B-007
rather than left to be discovered.

## Adding a payload

A new fact payload needs the ADR that sequences it, the published message in
`libs/contracts`, and a codec with round-trip conformance in
`libs/ledger-wire` — in both directions. A payload added without the second
direction passes forever while silently dropping whatever it never learned to
read.
