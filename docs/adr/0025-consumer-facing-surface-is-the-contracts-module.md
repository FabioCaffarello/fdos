---
id: ADR-0025
title: The consumer-facing surface is the contracts module; other published modules are not offered
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0025 — The consumer-facing surface is the contracts module; other published modules are not offered

## Context

[ADR-0004](0004-module-granularity.md) makes every `libs/*` an independent Go
module, and every release tags one. Six modules exist and five are tagged:
`libs/contracts`, `libs/kernel`, `libs/ledger`, `libs/kernel-wire`,
`libs/ledger-wire`. All are Apache-2.0 and all are resolvable through the Go
module proxy by anyone who types the path.

[ADR-0018](0018-contract-surface-is-protobuf.md) says the contract surface is
protobuf. `NOTICE` says private components depend on this repository
"exclusively through published, versioned contract modules". Neither states
which of the five that is, and the plural does not help.

So four modules are published, importable, and unaccounted for. Today nothing
outside imports them — `fdos-connectors` requires `libs/contracts` and nothing
else — which is exactly why this is cheap to settle now and expensive to settle
after somebody has built against `libs/kernel`.

The gap is a side effect rather than a decision. ADR-0004 chose module
granularity for dependency isolation and picked up publication as a consequence,
because Go has no concept of a module that exists but is not offered. A tag is a
tag.

This is D5's second half, recorded in
[`../ecosystem/boundary.md`](../ecosystem/boundary.md).

## Decision

**`github.com/FabioCaffarello/fdos/libs/contracts` is the consumer-facing
surface. It is the only module FDOS offers to code outside this repository.**

`libs/kernel`, `libs/ledger`, `libs/kernel-wire` and `libs/ledger-wire` are
published as a consequence of ADR-0004, not as an offer. They carry **no
compatibility promise across versions**, and a consumer importing one is
depending on FDOS's internal structure rather than on its contract.

`libs/analysis` is developer tooling and is not published at all.

The registry at [`../ecosystem/contracts.md`](../ecosystem/contracts.md) is
where this is discoverable, because a consumer reads the registry rather than
the decision log.

**What this ADR does not decide.** D5's first half — whether a proto contract
that is *not* canonical, such as an acquisition-internal host↔plugin boundary,
may be defined outside `fdos` — is unaffected and stays open. That half needs an
ADR in both repositories; this one is entirely within FDOS's own ownership of
its module topology and settles only that.

### Why the domain modules are not offered

A consumer importing `libs/kernel` gets `Money`, `EntityId` and `Explained[T]`
as Go types with Go semantics. That sounds helpful and is the problem: it
couples the consumer to FDOS's *implementation* of the canonical model rather
than to the model. A rename, a constructor signature, an unexported field —
none of which changes any contract — becomes a breaking change for somebody FDOS
does not know about.

The protobuf surface exists precisely so the canonical model can be consumed
without that coupling, and `buf breaking` gates it (`make proto-check`). No
equivalent gate exists or could exist for the Go API of five modules.

It also runs the wrong way across the open-core line. Constitution §13 says
private implementations depend on published *contract* versions. A private
connector importing `libs/ledger/domain` would be executing FDOS's domain rules
inside acquisition, which is the second boundary test failing.

## Consequences

### Positive

- A consumer can tell what it is allowed to depend on by reading one row of the
  registry, instead of inferring it from what happens to be tagged.
- FDOS can refactor the kernel and ledger freely. That freedom was assumed
  already; it is now stated, so acting on it is not a surprise.
- D5 gets smaller. What remains disputed is genuinely cross-repository, rather
  than a mix of that and a question FDOS could have answered alone.

### Negative

- **It forbids something that would sometimes be useful.** A future FDOS-adjacent
  tool in Go now has to go through protobuf to reach types it could have
  imported directly. That is a real tax paid for a boundary that is real.
- **The modules stay published regardless.** This decision changes what is
  *offered*, not what is *reachable*. Anyone can still import them, and their
  builds will work.
- **It creates a category the tooling cannot express**: published, tagged,
  Apache-2.0, and unsupported. Go has no word for that. The decision therefore
  lives entirely in prose, which is the weakest place for a rule to live.
- If FDOS ever wants a supported Go SDK over the canonical model, this ADR is
  what has to be superseded — and the supersession should say what changed about
  the coupling argument, not merely that it became convenient.

### Enforcement

**Rung 6 — human discipline. There is no mechanism, and there cannot be one
from here.**

Go offers no way to publish a module while withholding it. `internal/` is
path-scoped and would break `libs/kernel-wire` importing `libs/kernel` for the
same reason it would stop anyone else. Not tagging the modules would contradict
ADR-0004 and break the release flow. Unpublishing is not a thing.

The only place this could be enforced is in a consumer's own build, and FDOS
cannot reach into a private repository to check — the same structural
one-directionality [ADR-0023](0023-ecosystem-boundary-and-one-way-contract-flow.md)
records for the boundary itself.

What is enforced, and is the substantive half: `make proto-check` gates the
surface that *is* offered, and `make consumer-check` proves it resolves from
outside with no workspace and no `replace`.

Stated plainly because the alternative is implying coverage: **if a consumer
imports `libs/kernel` tomorrow, nothing in this repository will report it.**

## Alternatives considered

**Offer all published modules.** Rejected: it couples consumers to Go-level
implementation detail with no breaking-change gate, and puts FDOS domain rules
inside acquisition, failing the tax test.

**Stop tagging the non-contract modules.** Rejected: it contradicts ADR-0004,
breaks the release workflow's tag trigger, and would make `libs/ledger-wire`
unable to depend on a pinned `libs/kernel` — the workspace hides that today, CI
with `GOWORK=off` would not.

**Move the domain modules under an `internal/` path.** Rejected on mechanics:
Go's internal rule is path-scoped, so it would equally block `libs/kernel-wire`
and `libs/ledger`, which are the intended importers.

**Say nothing until somebody asks.** Rejected, and it was the tempting one —
nothing is broken today. But the cost curve is the argument: the answer is free
while no external code imports these modules and involves someone else's
migration afterwards. Deciding a boundary question while it is still hypothetical
is the whole point of having a decision log.

## Notes

Settles the FDOS-owned half of D5. The disputed half — whether a non-canonical
proto contract may be defined outside `fdos` — remains open in
[`../ecosystem/boundary.md`](../ecosystem/boundary.md) and needs an ADR in both
repositories.

This decision is invisible to `fdos-connectors` today, because it already
depends on `libs/contracts` alone. It is announced rather than assumed: the
registry states it, and the registry is what a consumer reads.
