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
| M3 | CI/CD and supply chain — pipeline, SBOM, provenance attestation, signing | ✅ built, and now proven end to end (B-008). Versions v0.1.0–v0.3.0 carry no evidence and are not back-filled |
| M2.5 | AI engineering — agent playbooks, prompt contracts, staleness checks | ✅ |
| M3.5 | Developer experience — devcontainer, IDE configuration, task ergonomics | ✅ |
| M4 | Contracts — protobuf schemas, `buf breaking` gate, generated Go SDK | ✅ |
| M5 | Open core boundary — published contract module, consumer proof | ✅ |
| M6 | First domain — the Ledger as a vertical slice | ✅ |
| M7 | Wire conformance — codecs and round-trip suites | ✅ |
| M8 | **Ingestion** — how a fact produced outside FDOS enters the ledger | next |

### M8 — Ingestion

**Chosen by dependency, not by preference.** It is the only candidate that
unblocks another repository: `fdos-connectors` C4 states that no position
projects until FDOS resolves claims, and today nothing does.

The domain half already exists. `libs/ledger/domain` has `Resolve`, `MintFor`
and `DeriveHoldingObserved`. **Nothing in `app/` calls any of them** —
`app.Ledger` offers `ObserveHolding`, `CorrectFact` and `ProjectPosition`, all
of which presume identity has already been resolved by someone. That gap is the
milestone.

Deliverables, in this order:

| # | Deliverable | Why it is here |
|---|---|---|
| 1 | **D4 decided** — RFC plus an ADR in both repositories | Gate. Accepting a fact from outside *is* the moment provenance becomes an admission criterion (I4). Building intake first would hard-code an answer to an open question, which is the accident ADR-0023 exists to prevent |
| 2 | An app-layer use case taking a claim through resolve → mint → derive → append | The missing call path above |
| 3 | An unresolved claim is observable | Closes the open item under B-007: claims can accumulate today with nothing derived and nobody told — a connector can publish faithfully into silence |
| 4 | Admission conformance | A fact whose provenance is inadmissible is rejected, and the rejection is testable rather than assumed |

**Where M8 stops: the `libs/` boundary.** No transport, no persistence, no
`apps/`. Those are M9.

Stopping there is not timidity, it is what the dependencies say. A transport
requires D2 — platform identity, who may send a fact — which is undecided. A
durable store requires choosing a technology, and ADR-0013 puts it in a separate
module precisely so that choice can be made after the shape it serves is known.
And what `fdos-connectors` needs to proceed is the *shape* and the *contract*,
not a running server.

The offline test stays intact: M8 is developable, buildable and testable with
every provider unreachable and `fdos-connectors` deleted.

**What would make this the wrong milestone:** if the in-memory-only store turns
out to make the intake path unrepresentative — if durability changes the
use case rather than just backing it. That is the thing to watch, and the
signal to fold M9 forward.

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

- **Nothing consumes a claim — now scheduled as M8.** `libs/ledger` can resolve
  a claim, mint an identity and derive a `HoldingObserved`, but no use case
  calls any of it and nothing reports a claim that was never resolved. The
  consumer's C4 states that no position projects until FDOS resolves claims.
  This was the one dependency pointing at an unscheduled milestone, and defining
  M8 is what removed it.

  **M8 is gated on D4, which is gated on them.** The RFC deciding what a
  `SourceRef` must resolve to is asked for from `fdos-connectors`, because the
  implementation experience is there. Until it exists, M8 cannot honestly start
  — so the milestone that unblocks C4 is itself waiting on the repository C4
  belongs to. That is not a deadlock: it is one RFC, and both sides know it.
- **No *already-published* release carries what M3 promised.** `v0.1.0` through
  `v0.3.0` resolve through the Go proxy and offer nothing else, because the
  release workflow failed on all fourteen tags (B-008). That is now fixed and
  proven by a disposable tag, so the next version published will carry a signed
  manifest, an attestation and an SBOM. The versions `fdos-connectors` pins
  today still carry none, and are not back-filled.
- ~~The governance corpus is published, and not yet vendored.~~ **Done.**
  `fdos-connectors` vendors `invariants.md` and `boundary.md` pinned to
  `ecosystem/v0.1.0` and byte-compared, under `fdos-connectors:ADR-0026`, with a second
  pin tracked separately from the platform pin. B-009 is closed.

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
