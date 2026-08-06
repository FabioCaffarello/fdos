# Ecosystem roadmap

Milestones are aligned by **dependency, not by date**. This file states what
each `fdos` milestone unblocks downstream, and what `fdos` is relied upon for
but has not scheduled.

Both sequences are kept here because a milestone that unblocks nothing needs
justifying, and one that blocks someone needs to be visible to them (I3).

## `fdos`

| Milestone | Objective | Status |
|---|---|---|
| M0 | Repository genesis — governance and enforcement substrate | ✅ |
| M1 | Governance substrate — `.context`, contribution and release process | ✅ |
| M1.5 | Canonical domain architecture — RFCs only | ✅ |
| M2 | Determinism toolchain — layer boundaries, analysers, reproducible builds | ✅ |
| M3 | CI/CD and supply chain — pipeline, SBOM, provenance attestation, signing | ✅ built, ⚠️ has never produced a release; cause fixed, unproven until a tag runs (B-008) |
| M2.5 | AI engineering — agent playbooks, prompt contracts, staleness checks | ✅ |
| M3.5 | Developer experience — devcontainer, IDE configuration, task ergonomics | ✅ |
| M4 | Contracts — protobuf schemas, `buf breaking` gate, generated Go SDK | ✅ |
| M5 | Open core boundary — published contract module, consumer proof | ✅ |
| M6 | First domain — the Ledger as a vertical slice | ✅ |
| M7 | Wire conformance — codecs and round-trip suites | ✅ |
| — | *No milestone is currently defined beyond M7.* | |

## `fdos-connectors`

Read from that repository's committed `README.md` on 2026-08-05. It is the
authority on its own sequence; this copy is for alignment and will go stale.

| Milestone | Objective | Status |
|---|---|---|
| C0 | Genesis — inherited platform, Charter, governance | ✅ |
| C1 | Boundary proof — a module consuming the published FDOS contract | ✅ |
| C1.5 | RFCs — plugin contract, session model, pipeline, lifecycle, isolation | ✅ |
| C2 | Contract surface — `plugin-api`, `connector-sdk`, capture, session, testkit | in progress |
| C3 | Determinism toolchain over the pure layers | |
| C4 | Runtime and first plugin — publishes claims | blocked, see below |
| C5 | Browser runtime and first authenticated plugin | |
| C6 | Private plugin repository consuming the published SDK | |

## What `fdos` unblocked, and when

| `fdos` delivered | Unblocked downstream |
|---|---|
| M4–M5 — `libs/contracts` published and proxy-resolvable | C1, which is done: the consumer resolves the module with no `replace` in its graph |
| M6–M7 + ADR-0022 — `HoldingClaimed`, `EntityMinted`, `IdentifierClaim` at `v0.3.0` | C2 and C4 have a hand-off shape; the consumer's own B-009 closed |

Before `v0.3.0` no published message could be fully populated by a connector —
every one carrying identity required an `EntityId`, which a connector cannot know
and must not mint. That is the worked example of a downstream need travelling
upstream correctly: issue → [RFC-0007](../rfc/0007-identity-resolution-and-the-acquisition-boundary.md)
→ [ADR-0022](../adr/0022-minting-an-identity-is-a-fact.md) → additive release,
with no type defined downstream. See [`../blocked.md`](../blocked.md) — B-007.

## What downstream relies on that `fdos` has not scheduled

**This is the part of the roadmap that matters**, and the reason this file
exists rather than two independent roadmaps.

- **Nothing consumes a claim.** `libs/ledger` can resolve a claim, mint an
  identity and derive a `HoldingObserved`. No process ingests, and nothing
  reports a claim that was never resolved — a connector can publish faithfully
  into silence. The consumer's C4 states that no position projects until FDOS
  resolves claims. That dependency currently points at an unscheduled milestone.
- **No release carries what M3 promised.** Every tag resolves through the Go
  proxy, so builds work; but no SBOM, attestation or signature has ever been
  published, because the release workflow failed on all fourteen tags (B-008).
  The cause is fixed and the next tag is the proof; until then a consumer that
  wants to verify the provenance of the contract module it pins still cannot.
- **The governance corpus has never been published as a version.** The consumer
  vendors the Constitution and script manifest byte-for-byte, pinned to nothing
  (B-009).

## The acquisition-contract promotion, and why it is not on this roadmap

The governance brief carries a program-management ruling: promote
`fdos.acquisition.v1` — `AcquisitionEnvelope`, `ProviderObservation`, a
provenance block and a connector runtime service — to its own milestone,
published *ahead* of the rest of the contract surface. The reasoning was that
contracts arriving late would starve the parallel team, which would fill the gap
with local types that could never be removed.

**The ruling is declined, on the grounds that the condition it was written for
did not occur.**

- **Its sequencing premise has expired.** It orders acquisition *before* the
  wider contract surface. The wider surface shipped at M4 and grew through M7.
  There is nothing left to be published ahead of.
- **The starvation it predicted did not happen.** The consumer was not starved:
  it consumed `libs/contracts` from C1, and when it hit a genuine gap it raised
  an issue rather than defining a type. B-007 is the receipt.
- **The types it names would arrive with no producer and no consumer.**
  `fdos` has no ingestion path, and the consumer already has a host↔plugin
  contract of its own that the four boundary tests assign to it. Publishing an
  `AcquisitionEnvelope` now would add an unexercised surface to the one module
  that is pinned by an external build — a contract with nothing on either end of
  it, which [ADR-0018](../adr/0018-contract-surface-is-protobuf.md) and B-002
  both give reasons to avoid.

What survives the ruling is its real question, and it is not a sequencing
question: *does `fdos` specify what provenance must resolve to, or does it stay
opaque?* That is D4 in [`boundary.md`](boundary.md), it is already open in
[ADR-0010](../adr/0010-provenance-envelope-reference-versioning.md), and it
wants an ADR in both repositories rather than a milestone here.
