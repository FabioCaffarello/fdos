---
id: RFC-0012
title: What a producer submits — a claim submission that omits knowledge time
status: Accepted
date: 2026-08-06
authors:
  - "@FabioCaffarello"
---

# RFC-0012 — What a producer submits

> **Accepted**, recorded by [ADR-0030](../adr/0030-the-submission-shape.md),
> which chose option A.
>
> It was written as a `Draft` because the choice in §4 is a product statement as
> much as an engineering one, and the alternative was defensible. It reached
> `main` still in that state, ahead of the decision — nothing was broken by that,
> but the decision and its record arrived in separate changes rather than
> together, and ADR-0030 says so rather than leaving the sequence to look like an
> omission.

## 1. The problem, found by trying to write the conformance kit

`E9` requires the open core to deliver value with the private repository absent.
[ADR-0029](../adr/0029-the-public-surface-receives-a-claim.md) decided the
public surface receives a claim, and `app.Ledger.AcceptHoldingClaim` now exists.

**No third party can reach it.** Two independent blocks, either sufficient.

### There is no wire shape a producer can fill

`AcceptHoldingClaimCommand` is a Go struct. Nothing published substitutes for it:

- **`ledger.v1.Fact`** requires an `Envelope`, whose `TemporalCoordinates`
  carry **knowledge time**. [ADR-0009](../adr/0009-universal-bitemporality.md)
  forbids a producer from supplying one — accepting it reintroduces the
  backdating that bitemporality exists to prevent. A producer filling a `Fact`
  must either lie about knowledge time or leave a required field empty.
- **`ledger.payload.v1.HoldingClaimed`** is the payload alone. No effective
  interval, no provenance. It is what a submission *contains*, not a submission.

So a producer writing anything other than Go has nothing to send.

### The entry point is in a module nobody is offered

[ADR-0025](../adr/0025-consumer-facing-surface-is-the-contracts-module.md)
decided `libs/contracts` is the only module FDOS offers, and that `libs/ledger`
is published as a consequence of ADR-0004 with **no compatibility promise**.

`AcceptHoldingClaim` lives in `libs/ledger/app`. **The entry point is reachable
only by importing a module this repository tells people not to depend on.**

That is not a defect in the work that built it — the entry point had to exist
before the gap was visible — but it means `E9` was further from met than "the
kit is not written yet" suggested.

## 2. Proposal

**Publish a submission message: everything the command carries, minus knowledge
time.**

```
fdos.ledger.v1.HoldingClaimSubmission
  stream
  account      IdentifierClaim
  instrument   IdentifierClaim
  quantity     Quantity
  effective    Interval          — when the holding was true, per the source
  provenance   Provenance        — source, collected_at, interpreter, confidence
  references   [ReferenceBinding]
```

**The omission is the design.** Knowledge time is what FDOS learned and when,
and it is assigned by the ledger's clock at admission. A submission message that
*could* carry it would be a message a producer could backdate with, and no
amount of documentation makes an available field unused.

That is also precisely why `Fact` cannot be reused: `Fact` is what the ledger
holds *after* admission. A submission is what arrives before. They differ by
exactly the field a producer must not set, which is the clearest possible signal
that they are two messages rather than one.

`stream` is included and is a producer's choice — which account's stream this
belongs to. It is not identity: naming a stream is not asserting who owns it.

## 3. What this does not add

**No transport.** The message is a shape, not an endpoint. How bytes arrive —
a file, a queue, an RPC — is undecided and depends on D2, which asks who may
send a fact at all.

**No service.** ADR-0029 settled that: a service FDOS operates re-fails `E9`,
because ingestion requiring FDOS infrastructure makes the open core usable only
while the lights stay on.

A producer emits bytes and something admits them. This RFC decides only the
bytes.

## 4. The alternatives, and the one that is a product decision

**A — publish the submission message.** Proposed. Any language, and ADR-0025
stays intact: a producer depends on `libs/contracts` and nothing else, which is
already what the one existing consumer does.

**B — offer `libs/ledger`, or a thin `libs/ingest`, as a second consumer-facing
module.** Rejected. It amends ADR-0025 to widen the supported Go surface
permanently, and it serves Go producers only — for an invariant whose words are
*any* third party. A Go-only ingress means the open core is usable alone if you
happen to write Go, which is a materially weaker claim than `E9` makes.

**C — no submission message; the kit is a copyable Go example, and `E9` is
declared partially met.** Not refuted, and **this is the product decision**, not
an engineering one. It is faster and cheaper, and it concedes something specific:
that "the truth engine is open and auditable" comes with "if you write Go". That
concession may be correct for a period, and if it is taken it should be written
down as a stated limitation rather than left for an adopter to discover.

This RFC recommends A and does not pretend C is wrong.

## 5. Consequences

- The kit becomes buildable, and buildable **against bytes rather than against a
  Go struct** — so it tests a producer's output in any language, which is what
  ADR-0029 meant by "the kit tests the producer, not the ledger".
- A decoder is needed on the FDOS side, in `libs/ledger-wire`, with round-trip
  conformance like every other codec (B-003's standing rule).
- `libs/contracts` gains a message and a version. Additive: no existing message
  changes shape.
- `E9` remains at rung 6 until the kit ships. This RFC removes the blocker; it
  does not clear the invariant.

## 6. Costs

- **Contract surface grows before a producer exists.** The one message FDOS
  publishes that nobody yet sends. Justified only because `E9` makes the absent
  producer the point, and a shape nobody can fill is what blocks them.
- **`stream` is a producer-supplied name** and nothing validates who may write
  to it. That is D2 and stays open — this message does not answer it and must
  not be read as answering it.
- **A submission message is a second way to express a holding claim**, alongside
  the `Fact` the ledger stores. Two shapes for one concept is the divergence
  B-003 exists to catch, and the codec plus round-trip conformance is what keeps
  them honest.

## 7. Alternatives considered within A

**Reuse `Fact` and document that knowledge time is ignored.** Rejected: an
ignored field is a lie-shaped field. The same objection that produced the
`unmediated` sentinel applies — a value that means "do not read this" gets
filled anyway, and nothing detects it.

**Make knowledge time optional in `Envelope`.** Rejected: it weakens the
guarantee for every stored fact in order to shape a submission, which is the
same trade rejected for `InterpreterRef` in RFC-0011.
