# Ecosystem roadmap

Milestones are aligned by **dependency, not by date**. This file states what
each `fdos` milestone unblocks downstream, and what `fdos` is relied upon for
but has not scheduled.

Both sequences are kept here because a milestone that unblocks nothing needs
justifying, and one that blocks someone needs to be visible to them (E3).

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
| M8 | **Ingestion** — how a fact produced outside FDOS enters the ledger | ✅ |
| M9 | **Resolution** — who mints, when, on whose authority; per-scheme canonicalisation before `Derive`. Track B (agent knowledge and harness) delivered in-milestone | Track A gated on [PR #51](https://github.com/FabioCaffarello/fdos/pull/51); open items in [#57](https://github.com/FabioCaffarello/fdos/issues/57) |
| M9.5 | Governance ops — PREVC adopted (ADR-0031), blocked register moved to issues (ADR-0032), issue templates, hygiene | in progress, gate [PR #52](https://github.com/FabioCaffarello/fdos/pull/52) |
| M10 | **Persistence** — the ledger event store: durable facts, bitemporal as-of reads, reproducible. RFC before code | after M9 Track A |
| M11 | **Transport** — first composition root in `apps/`: a submission service over `fdos.ingest.v1`; makes D2 a live decision | plan gate open; two preconditions named there — ADR-0029's "never a service" clause, and D2 |
| M12 | **Consumer enablement** — validator binary and conformance kit distributed; unblocks `fdos-connectors` C4 | after M11 and the D4 decision ([fdos#28](https://github.com/FabioCaffarello/fdos/issues/28)) |

### The `content_hash` rename, and why it has no deadline

[ADR-0028](../adr/0028-provenance-admissibility.md) decided `SourceRef.value`
should be `content_hash`. It could not ship with that decision: a field rename
is breaking under this repository's `buf breaking` configuration, and
[ADR-0024](../adr/0024-contract-lifecycle-and-versioning.md) puts a breaking
change in a new package path. **The rename is a blocking obligation on whatever
eventually warrants `fdos.kernel.v2`**, and nothing warrants one today.

An earlier version of this section declared a hard dependency: the public
ingress must not publish before the rename, because third-party adoption would
render the rename unaffordable. **That was wrong, and the error is worth
keeping.**

A rename can only ride a major boundary, and a major boundary migrates every
consumer by construction — anyone moving from `kernel/v1` to `kernel/v2`
reimports regardless. So third-party adoption of `v1` does not make the rename
more expensive by a single unit. It makes the *`v1`→`v2` migration* more
expensive, which is a separate and already-known cost that exists with or
without the rename. **There is no point of no return for a change that can only
travel on a major.**

What made the deadline look real was the argument that the identifier was
carrying the enforcement — that `value` invites a URL or an account id. That
held while admissible provenance sat at rung 6. It no longer does: the grammar
is checked at admission, and `value` now refuses a URL because the *grammar*
refuses it, not because of what the field is called. The rename is a call-site
readability improvement, which is real and does not pay for a major version.

So the ingress and the conformance kit publish on `v1`. The obligation stays
recorded; the invented deadline is gone.

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
| 1 | **D4 decided** — RFC plus an ADR in both repositories | Gate. Accepting a fact from outside *is* the moment provenance becomes an admission criterion (E4). Building intake first would hard-code an answer to an open question, which is the accident ADR-0023 exists to prevent |
| 2 | An app-layer use case taking a claim through resolve → mint → derive → append | The missing call path above |
| 3 | An unresolved claim is observable | Closes the open item under B-007: claims can accumulate today with nothing derived and nobody told — a connector can publish faithfully into silence |
| 4 | Admission conformance | A fact whose provenance is inadmissible is rejected, and the rejection is testable rather than assumed |

**Where M8 stops: the `libs/` boundary.** No transport, no persistence, no
`apps/`. When this section was written those were expected to be M9; resolution
claimed M9 instead, and they are scheduled as M10 (persistence) and M11
(transport) in the table above.

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
| C2 | Contract surface — plugin API, connector SDK, capture, session, testkit | in progress |
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

> **Premise invalidated by [ADR-0027](../adr/0027-invariant-renumbering-and-matrix-redaction.md).**
> `E9` requires the open core to be useful with the private repository absent,
> which supplies the producer this rejection assumed did not exist: any third
> party. The argument below was sound on the evidence available and is wrong on
> this evidence. Reversing it needs its own RFC; until then, do not implement
> against it.

**The ruling was declined, on the grounds that the condition it was written for
had not occurred.**

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
