---
id: ADR-0026
title: Only canonical contracts are FDOS's, and toolchain ownership is not named by language
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0026 — Only canonical contracts are FDOS's, and toolchain ownership is not named by language


> **Banner — invariant renumbering (`ecosystem/v0.3.0`).** This decision cites
> the ecosystem invariants as `I1`–`I8`. They were renumbered to `E1`–`E8` to
> stop colliding with the downstream Charter's own `I-1`…`I-10`. The mapping is
> the identity; read `I2` here as `E2`. Nothing about this decision changed —
> the text stands as accepted, per `fdos:ADR-0000`. See
> [`../ecosystem/invariants.md`](../ecosystem/invariants.md).

> **Banner — disclosure redaction (`ecosystem/v0.3.0`).** This decision names
> identifiers belonging to the private repository. They are retained here
> because they are the *subject* of the decision and removing them would make
> the reasoning unreadable; current documents no longer repeat them. The
> boundary rule and what remains permanently published are recorded in
> [`../disclosure.md`](../disclosure.md).

## Context

Records what [RFC-0008](../rfc/0008-narrowing-two-responsibility-matrix-rows.md)
settled, in answer to
[fdos#25](https://github.com/FabioCaffarello/fdos/issues/25).

Two rows of the Tier-0 responsibility matrix shipped at `ecosystem/v0.1.0` with
known defects. They were published anyway, listed under *Corrections pending*,
because Tier 0 is amendable only by an RFC here plus an ADR in both
repositories — and shipping a knowingly-wrong row honestly was preferable to
either delaying the corpus or quietly fixing text the consumer vendors verbatim.

The contracts row, read literally, forbade `fdos-connectors` from publishing the
wire contract between its own plugin host and its plugins. The Python row
asserted something untrue of an ecosystem that is entirely Go.

Both defects were identified independently on each side before either had seen
the other's reasoning: this repository flagged them while drafting the corpus,
and `fdos-connectors` filed the four boundary tests applied to its schema, with
the same proposed narrowing.

## Decision

**A contract is FDOS's when it is *canonical* — when it defines or constrains
the meaning of a financial fact.** Transporting a fact whose type and shape FDOS
defined is not defining it.

The matrix rows become:

> | **Canonical** contracts (proto, schemas, generated SDKs) | `fdos` | Single source, one-way flow (I2) |

> | Language toolchains beyond Go | the repository that uses one | A toolchain present in one repository only is owned there. Today both repositories are Go |

Two conditions make the first operable rather than a matter of taste, and both
already hold:

1. **No package outside `fdos` sits under `fdos.*`.** An import path is the
   most-read claim of ownership in any codebase.
2. **A non-canonical contract may carry canonical payloads but may not declare
   them.** `google.protobuf.Any` plus a registry of published messages is the
   permitted shape. A locally-defined message describing a holding, a trade or
   an instrument is not.

I2 is unchanged. Nothing outside `fdos` defines or redefines a canonical type,
and there is still no reverse edge.

The corpus is republished as `ecosystem/v0.2.0`.

**What this does not decide.** D1 (browser runtime provenance), D2 (the
authentication split) and D3 (where normalisation stops) are untouched. D4 —
what a `SourceRef` must resolve to — is untouched and remains M8's gate. This
ADR closes D5 and nothing else.

## Consequences

### Positive

- The authoritative text now matches what both repositories actually do, so the
  correction stops living downstream as an exception against an upstream rule —
  which is the inversion Tier 0 exists to prevent.
- The rule is testable rather than illustrative. "Does it define the meaning of a
  financial fact" answers the next case too, which naming `fdosconn.plugin.v1` as
  a permitted exception would not have.
- It vindicates the decision to ship the corpus with its defects listed. The
  errors were found and fixed *because* they were written down as errors; a
  silently-corrected row would have diverged from the charter with nobody
  looking.

### Negative

- **"Canonical" is a judgement, and judgements drift.** The operational test
  narrows it but does not eliminate it. A schema that starts as transport and
  accretes meaning one field at a time is the realistic failure, and it would
  arrive as a series of individually reasonable pull requests in a repository
  this one cannot review.
- **Nothing here can check either condition downstream.** That no package
  outside `fdos` sits under `fdos.*` is enforced in the consumer's own
  `proto-check` (`fdos-connectors:ADR-0019`), which is the only place it can be
  enforced — and is therefore a rule FDOS depends on and cannot verify.
- **Every Tier-0 amendment costs the consumer a reviewed re-sync.** That is the
  intended price of vendoring verbatim, but it means the corpus should not be
  amended casually, and two amendments inside a day is already close to the line.
- The second row now says "today both repositories are Go", which is a fact with
  an expiry date sitting inside a document meant to change rarely.

### Enforcement

**Rung 6 for the matrix itself**, as [ADR-0023](0023-ecosystem-boundary-and-one-way-contract-flow.md)
records: FDOS cannot verify that a private repository vendored anything, and
building a mechanism that could would violate the boundary being enforced.

**Rung 3 for the half that is mechanical, in the repository that owns it:**

| Condition | Where enforced |
|---|---|
| `fdos` publishes only `fdos.*` | `make proto-check` here |
| `fdos-connectors` publishes only `fdosconn.*` | their `proto-check` (`fdos-connectors:ADR-0019`) |
| A payload becomes a fact only if FDOS published it | their SDK's type registry |

None of the three was added by this ADR. All three predate it, which is why the
narrowing describes reality rather than requesting a change.

## Alternatives considered

**Leave the rows and rely on the consumer's expiring exception.** Rejected: the
authoritative text stays wrong while the correction lives downstream, and the
exception expires by its own terms, so inaction would strand it.

**Name `fdosconn.plugin.v1` as a permitted exception.** Rejected: it settles one
schema instead of a boundary, and the next non-canonical contract reopens the
question. The test belongs in the rule.

**Move `plugin-api` into `fdos`**, which is what the unamended row demands.
Rejected: `fdos` would carry a schema describing plugin-host mechanics it has no
use for, whose shape is driven by a runtime it does not own. The offline test
fails in the direction that matters.

**Delete the toolchain row.** Rejected narrowly — the rule will matter the first
time either repository adds a non-Go toolchain, and deleting it leaves that
unowned.

## Notes

`fdos-connectors` retires its own exception; this ADR does not close their
decision. The mirror is [fdos#25](https://github.com/FabioCaffarello/fdos/issues/25).

D5 is closed. D1, D2, D3 remain open in
[`../ecosystem/boundary.md`](../ecosystem/boundary.md); D4 remains M8's gating
deliverable.
