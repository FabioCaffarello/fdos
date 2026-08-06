---
id: ADR-0030
title: A producer submits a claim submission, and knowledge time is not in it
status: Accepted
date: 2026-08-06
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0030 — A producer submits a claim submission, and knowledge time is not in it

## Context

Records what [RFC-0012](../rfc/0012-the-submission-shape.md) settled.

[ADR-0029](0029-the-public-surface-receives-a-claim.md) decided the public
surface receives a claim, and `app.Ledger.AcceptHoldingClaim` was built. Writing
the conformance kit found that **nobody can reach it**:

- `AcceptHoldingClaimCommand` is a Go struct, and no published message
  substitutes. `ledger.v1.Fact` requires an `Envelope` carrying **knowledge
  time**, which [ADR-0009](0009-universal-bitemporality.md) forbids a producer
  from supplying; `ledger.payload.v1.HoldingClaimed` is the payload alone, with
  no effective interval and no provenance.
- The entry point lives in `libs/ledger`, which
  [ADR-0025](0025-consumer-facing-surface-is-the-contracts-module.md) publishes
  **without offering**.

So a producer in any language but Go had nothing to send, and a producer in Go
could only reach it by importing a module this repository tells people not to
depend on.

## Decision

**FDOS publishes a claim submission message.** It carries everything
`AcceptHoldingClaimCommand` carries — the two identifier claims, the quantity,
the effective interval, the provenance block and any reference bindings — and
**omits knowledge time**.

### The omission is the decision

Knowledge time is what FDOS learned and when. It is assigned by the ledger's
clock at admission, and a submission message that *could* carry it is a message
a producer can backdate with. No amount of documentation makes an available
field unused — the same reasoning that produced the `unmediated` sentinel in
[ADR-0028](0028-provenance-admissibility.md) rather than an optional
interpreter.

This is also why `Fact` cannot be reused. `Fact` is what the ledger holds
*after* admission; a submission is what arrives *before*. **They differ by
exactly the field a producer must not set**, which is the clearest available
signal that they are two messages rather than one with a caveat.

### The alternative that was not taken, and what it would have conceded

RFC-0012 named a cheaper option: no submission message, a copyable Go example,
and `E9` declared partially met. It was a real candidate and this ADR does not
treat it as a straw man.

Taking it would have conceded something specific — *the truth engine is open and
auditable, if you write Go.* `E9`'s words are "any third party", and a Go-only
ingress is a materially weaker claim than the invariant makes. The concession
was declined rather than deferred.

### What this does not decide

**No transport.** The message is a shape, not an endpoint. How bytes arrive is
undecided and depends on D2 — who may send a fact at all — which remains open.

**No service.** ADR-0029 settled that: a service FDOS operates re-fails `E9`.

## Consequences

### Positive

- The conformance kit becomes buildable **against bytes rather than against a Go
  struct**, so it tests a producer's output in any language. That is what
  ADR-0029 meant by "the kit tests the producer, not the ledger".
- ADR-0025 stays intact. A producer depends on `libs/contracts` and nothing
  else, which is already the arrangement the one existing consumer uses.
- The one field a producer must never set is absent from the shape rather than
  documented as forbidden, which is the difference between a rule and a type.

### Negative

- **Contract surface grows before any producer exists.** This is the one message
  FDOS publishes that nobody yet sends. Justified only because `E9` makes the
  absent producer the point, and a shape nobody can fill is precisely what keeps
  them absent.
- **A second way to express a holding claim.** The submission and the stored
  `Fact` are two shapes for one concept, which is the divergence B-003 exists to
  catch. A codec with round-trip conformance in `libs/ledger-wire` is what keeps
  them honest, and without it this decision creates the drift it was warned
  about.
- **`stream` is producer-supplied and unguarded.** Nothing validates who may
  write to a named stream. That is D2, it stays open, and this message must not
  be read as answering it.
- **`E9` is not cleared by this.** It removes the blocker in front of the kit; it
  does not put the kit in anyone's hands.

### Enforcement

**Publishing a message enforces nothing**, and the rungs belong to what follows
it:

| Property | Rung today |
|---|---|
| A producer cannot supply knowledge time | **1 — the field does not exist in the shape** |
| A submission decodes to exactly what was sent | **none — no codec yet** |
| Submission and stored fact do not diverge | **none — no round-trip suite yet** |
| Only an authorised party may write to a stream | **none — D2 is open** |

**Execution-context question.** The first row is the only one this decision
delivers, and it is delivered by absence: there is no code path that could
accept a knowledge time from a producer, because there is no field to carry one.
That is the strongest rung available and it costs nothing to maintain.

The other three are **not built**, and would report nothing if violated today —
because nothing yet decodes a submission, so nothing yet can decode one wrongly.
The codec and its conformance suite are the next slice, and until they exist the
second and third rows are absent rather than passing.

## Alternatives considered

**Offer `libs/ledger` or a thin `libs/ingest` as a second consumer-facing
module.** Rejected: it amends ADR-0025 to widen the supported Go surface
permanently, and serves Go producers only.

**No submission message; a copyable Go example; `E9` partially met.** Declined,
with the concession it carries stated above.

**Reuse `Fact` and document that knowledge time is ignored.** Rejected: an
ignored field is a lie-shaped field. A value meaning "do not read this" gets
filled anyway, and nothing detects it.

**Make knowledge time optional in `Envelope`.** Rejected: it weakens the
guarantee for every stored fact in order to shape a submission — the same trade
rejected for `InterpreterRef` in RFC-0011.

## Notes

**How this acceptance arrived.** RFC-0012 was merged while still `Draft`, ahead
of the decision it was waiting for. Nothing was broken by that — a `Draft` RFC on
`main` is a valid state and `make rfc-check` only requires an *accepted* RFC to
have an ADR — but the decision and its record arrived in separate changes rather
than together. Recorded because the sequence is visible in the log and would
otherwise look like an omission.

The message name, field numbering and the codec are the next slice. D1, D2 and
D3 remain open.
