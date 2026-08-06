---
id: ADR-0024
title: Contracts are versioned per module, and a breaking change is a process
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0024 — Contracts are versioned per module, and a breaking change is a process

## Context

[ADR-0018](0018-contract-surface-is-protobuf.md) decided that the contract
surface is protobuf. [ADR-0004](0004-module-granularity.md) decided that every
`libs/*` is an independent Go module. Together those determine how a contract is
*published*, but nothing has ever written down how it is **versioned,
deprecated, or removed**, or what a consumer is entitled to expect when it
changes.

Three published versions and one external consumer exist, so the scheme is
already in force. It is in force by habit.

The governance brief describes a different scheme — a single `contracts/vX.Y.Z`
tag carrying the buf module, a generated Go module, a **generated Python
distribution**, an SBOM and a changelog entry. That describes an ecosystem this
is not: there is no Python anywhere in either repository, and Go's module
resolution does not permit the tag shape it names. Recording the actual scheme
matters more than restating the intended one, and the difference is not
cosmetic — a consumer that believed the brief would pin a tag that does not
exist.

## Decision

**A contract version is a Go submodule tag: `libs/contracts/vX.Y.Z`.**

The tag carries the module directory prefix because Go requires it — a module in
a subdirectory is only resolvable through a tag that names its path. The tag
names the path; the package path carries the concept.

**Semantic versioning, with the protobuf package as the compatibility unit.**
`fdos.<context>[.<sub>].<version>` — the major version lives in the package
path, so `v2` coexists with `v1` rather than replacing it. A new message or a
new field is a minor bump. Nothing that changes the meaning of an existing field
is ever a minor bump.

**All canonical packages version together.** `fdos.kernel.v1`,
`fdos.ledger.v1` and `fdos.ledger.payload.v1` ship in one module and share one
version. A `Fact` references a `Provenance` and a payload is only meaningful
inside a fact; three independently versioned tags could never legally disagree,
so the separation would be bookkeeping that invites incoherent combinations.

**`buf breaking` is the gate, and it runs on every pull request.**
`make proto-check` compares the surface against `main`. A breaking change cannot
reach the default branch by inattention — only by intent.

**A breaking change is a process, not an event** (I7):

1. RFC in `fdos` describing the change, the migration, and the cost accepted.
2. **An issue in every consuming repository, opened before the change merges**,
   referencing the RFC.
3. ADR accepted in `fdos`.
4. The new version published *alongside* the old. Both valid.
5. N-1 held for at least one full milestone.
6. The consumer closes its own migration issue. `fdos` does not close it.
7. Removal, in its own release.

Step 2 is the load-bearing one. Everything else recovers from being done late; a
consumer that discovers a break at build time starts defensively copying types,
and that is I1 gone.

**The registry at [`docs/ecosystem/contracts.md`](../ecosystem/contracts.md) is
part of the interface**, not documentation about it. A published version that is
not in the registry is undiscoverable by the only mechanism the other repository
is allowed to use (I3).

## Consequences

### Positive

- The tag shape is now stated, so a consumer pins the thing that exists rather
  than the thing the brief describes.
- Versioning all canonical packages together removes a class of bug nobody would
  enjoy debugging: a `Fact` from one version holding a payload from another.
- `buf breaking` means the first breaking change will be deliberate. That is
  worth more than the process steps around it, because it is rung 3 and they are
  rung 6.

### Negative

- **Coarse versioning bumps innocent consumers.** A payload-only addition bumps
  the version for everyone, including a consumer that touches only
  `fdos.kernel.v1`. Accepted: Go modules make this cheap, and coherence is worth
  more than churn.
- **Steps 1–7 are almost entirely rung 6.** Nothing checks that a consumer issue
  was opened before merge; nothing measures the N-1 window; nothing prevents
  removal in the same release as deprecation. With one consumer and one
  maintainer this is discipline, and it will be the first thing to fail under
  time pressure.
- **The `fdos` side cannot verify consumer migration at all.** The consumer is
  private. Step 6 is unobservable from here by design, so "the migration issue is
  closed" is a claim this repository can never check.
- **This ADR ratifies a release process that has never once completed.** See
  below. Writing the lifecycle down while its publishing half is broken risks
  implying a maturity that does not exist, which is precisely why it is stated
  in the registry too.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| No breaking change reaches `main` unnoticed | 3 — CI | `make proto-check` → `buf breaking --against main` |
| The module is resolvable from outside, unpinned to this workspace | 3 — CI | `make consumer-check`, `GOWORK=off`, no `replace` |
| Generated Go matches the schemas | 3 — CI | `make proto-check` proves regeneration is a no-op (I6) |
| Registry lists every published version | 6 — discipline | nothing checks it |
| Consumer issue opened before a breaking merge | 6 — discipline | nothing checks it |
| N-1 window held for a milestone | 6 — discipline | nothing checks it |

**A defect this ADR must not paper over.** The release workflow has failed on
**all fourteen tags**, and zero GitHub releases exist. Every tag is resolvable
through the Go module proxy — which is why the consumer builds — but no
published version carries the SBOM, build-provenance attestation, cosign
signature or release notes that M3 delivered the machinery for. `make
consumer-check` is a step inside that workflow, so it has never run on a tagged
commit either.

The cause is mundane: the release job installs Go but not the rest of the pinned
toolchain, so `make verify` fails at `toolchain-check` before anything else
runs. It is recorded as B-008 in [`../blocked.md`](../blocked.md) and is not
fixed by this ADR, which is about the lifecycle rather than the pipeline.

The general lesson is the one the enforcement ladder already states: a green
check is evidence about the check. Here there was no green check — a workflow
failed fourteen times on a trigger that blocks nothing, and nobody was told.

## Alternatives considered

**One tag for all contracts, `contracts/vX.Y.Z`, as the brief specifies.**
Rejected on mechanics: Go resolves a submodule only through a tag carrying its
directory prefix, so `contracts/v0.3.0` would not be fetchable. Fighting the
module system to match a document is the wrong way round.

**Version each protobuf package independently.** Rejected: they are not
independently meaningful, and the tags could never legally disagree. It would
add combinations that must never occur and no ability to prevent them.

**Publish a Python distribution alongside the Go module**, per the brief.
Rejected: there is no Python in this ecosystem. `fdos-connectors` is a Go
workspace with four Go modules. A published artifact with no consumer is a
maintenance obligation and a supply-chain surface bought for nothing.

**Ship a `CHANGELOG.md`.** Not rejected — deferred, and worth naming. There is
no changelog today, so a consumer's account of what changed is this ADR's
registry plus [`../blocked.md`](../blocked.md). A changelog becomes the right
answer once releases are actually published (B-008); adding one now would
document releases that do not exist.

## Notes

The version history table in
[`docs/ecosystem/contracts.md`](../ecosystem/contracts.md) records what each
published version gave a consumer. No breaking change has been published, so
steps 1–7 above are untested. The first one to run them will find out which of
the rung-6 rules survive contact.
