# AGENTS.md

Entry point for AI agents working on FDOS. Read this first, then the sources it
points at — this file is a signpost, not a substitute.

## Read before acting

1. [`docs/constitution.md`](docs/constitution.md) — fourteen principles and the
   enforcement ladder. **Highest authority in the repository.**
2. [`docs/adr/`](docs/adr/) — accepted decisions. Append-only and immutable.
3. [`docs/ecosystem/boundary.md`](docs/ecosystem/boundary.md) — who owns what
   across the two repositories, the four boundary tests, and the disputed items
   that are **not** settled by whoever writes code first (ADR-0023).
4. The `README.md` front matter of any directory you change — that front matter
   is the binding contract for the directory.
5. [`CONTRIBUTING.md`](CONTRIBUTING.md) — workflow and definition of done.
6. [`.context/`](.context/README.md) — structured knowledge derived from `docs/`.

## What exists

The canonical model has landed as code. RFC-0001 … RFC-0007 are accepted and
recorded in ADR-0007 … ADR-0012 and ADR-0022; M6 built the Ledger as a vertical
slice against them, and M7 added the conformance suites that keep the domain and
wire definitions of each type from diverging.

| Module | Holds |
|--------|-------|
| `libs/analysis` | The analysers that turn architectural principles into build errors |
| `libs/contracts` | The published contract surface — protobuf schemas and the Go generated from them |
| `libs/kernel` | Canonical types: identity, money, temporal, provenance, explained |
| `libs/kernel-wire` | Kernel ↔ protobuf codecs, and the round-trip conformance suite |
| `libs/ledger` | The first bounded context: facts, claims, mints, resolution |
| `libs/ledger-wire` | Ledger ↔ protobuf codecs, and the round-trip conformance suite |
| `libs/ledger-sqlite` | The durable event store (ADR-0034, ADR-0035). **Contract and test plan only** — it has no `go.mod` yet, because it cannot compile until the M10 port change is released as `libs/ledger v0.4.0` |

Each is an independent Go module published under its own tag (ADR-0004). A
private repository already builds against `libs/contracts` at a pinned version,
so a change to that module is a change to somebody else's build.

**What is undecided is still not yours to decide.** The rule that mattered when
the repository was empty has narrowed rather than lapsed: do not add a bounded
context, a canonical type or a published message ahead of the ADR that sequences
it. Adding a payload to `libs/contracts` because a consumer needs one is the
version of this that will look most reasonable at the time —
[`docs/blocked.md`](docs/blocked.md) B-007 (frozen; its open items live in
[issue #57](https://github.com/FabioCaffarello/fdos/issues/57), per ADR-0032)
records how that is meant to go instead, and it starts with an issue and an
RFC.

## Commands

```sh
make bootstrap   # validate the toolchain against the pins in mise.toml
make verify      # every enforcement mechanism available at this milestone
make help        # list targets
```

There is no npm, no Jest, no TypeScript and no `dist/`. The toolchain is Go plus
`make`, pinned in `mise.toml`.

Never report work as complete without `make verify` passing. If you cannot run
it, say so rather than implying you did.

## Repository map

| Path | Contents |
|------|----------|
| `libs/` | Reusable libraries, one Go module per subdirectory — except a directory carrying its contract ahead of its module, as `libs/ledger-sqlite` does. |
| `apps/` | Deployable applications, composition roots only. Empty. |
| `docs/` | Constitution, ADRs, RFCs, and the register of blocked work. Authoritative. |
| `deploy/` | Deployment topology. Empty. |
| `examples/` | Executable demonstrations of the public contracts. Empty. |
| `scripts/` | Enforcement mechanisms, invoked through `make`. |
| `.github/` | CI workflows — `verify`, `release`, `supply-chain`. They invoke `make` and hold no logic of their own (ADR-0014). |
| `.context/` | Knowledge for agents, derived from `docs/`. |

`apps/`, `deploy/` and `examples/` are empty **by design**, not by omission, and
each says in its `README.md` what may live there.

`make contracts-check` enforces that declaration for every top-level directory
**and every module under `libs/`**. It stops there: layers below
a module are packages rather than boundaries (ADR-0013), and a README per
package would produce contracts nobody reads to satisfy a check nobody
believes.

## Rules that will get a change rejected

- Editing an accepted ADR to change its meaning. Reverse by superseding
  (ADR-0000); `make adr-check` validates the link in both directions.
- A structural change with no ADR.
- A new enforcement mechanism with no negative test. A check that has never gone
  red is unverified.
- Documentation left stale by the same change.
- Anything routing LLM output toward the ledger. Models explain; they are never
  the source of financial truth (Constitution §2).
- Implementing against content marked *provisional*. That converts a hypothesis
  into a decision without an ADR.

## Commits and pull requests

Conventional prefixes (`feat`, `fix`, `docs`, `chore`, `refactor`, `test`,
`build`). The body records **why**, states the costs accepted, and names any
superseded decision. The diff already says what changed.

## When you disagree

If a request contradicts an accepted ADR, say so explicitly and propose the
superseding ADR. Never quietly work around one.

If the Constitution itself is the obstacle, say that too. It is amendable
through an RFC, a version bump and an ADR. It is not ignorable.
