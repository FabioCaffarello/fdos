---
id: ADR-0029
title: The public surface receives a claim, and the ledger admits what a library merely builds
status: Superseded
date: 2026-08-06
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by:
  - ADR-0037
---

# ADR-0029 — The public surface receives a claim, and the ledger admits what a library merely builds

> **Superseded by [ADR-0037](0037-delivery-includes-a-service-the-adopter-operates.md).**
> Delivery now includes a submission service **an adopter operates**; a service
> **FDOS operates** is still rejected, for the reason given below. The two other
> decisions here — that the public surface receives a claim, and that a library
> builds while the ledger admits — are carried forward unchanged and restated
> there, so that they are not read as withdrawn along with the third.
>
> The text below is preserved unaltered, as ADR-0000 requires: it records what
> was decided and why, not what turned out to be right. Two statements in it are
> known to be wrong now, and ADR-0037 says so rather than this document being
> edited — the flat rejection of *"a service"* in §Alternatives, and the closing
> note that the `SourceRef` rename *"binds this work"*, a deadline
> `docs/ecosystem/roadmap.md` has since retracted.

## Context

Records what [RFC-0010](../rfc/0010-the-public-surface-receives-a-claim.md)
settled. Its stated precondition — D4 — was settled by
[ADR-0028](0028-provenance-admissibility.md), which is what makes acceptance
mechanical rather than a judgement call.

`E9` requires the open core to build, test, run and **deliver value** with the
private repository absent. It was admitted unmet at `ecosystem/v0.3.0`, and the
reason is narrower and worse than "there is no ingestion path":

```go
type ObserveHoldingCommand struct {
    Account    identity.ID
    Instrument identity.ID
```

Every entry point on `app.Ledger` takes an already-resolved `identity.ID`, and
[ADR-0007](0007-internal-deterministic-identity.md) and
[ADR-0022](0022-minting-an-identity-is-a-fact.md) forbid an external producer
from minting one. **The only public application-layer surface is structurally
unusable from outside** — not undocumented, not inconvenient, unusable, by a
rule this repository is right to have.

## Decision

### The public surface receives a claim

An external producer asserts identifiers — `{scheme, value}`, verbatim as it
read them — and FDOS resolves them internally, where ADR-0007 puts resolution
and where the resulting mint is itself a recorded fact.

This is not a new mechanism. `domain.Resolve`, `domain.MintFor` and
`domain.DeriveHoldingObserved` already exist and already do this; nothing in
`app/` calls them. What is decided is that the path the domain already
implements is exposed to the only kind of caller that can use it.

**Packaging is a delivery question and not this decision.** A library exposing
`ObserveHolding`'s signature has exactly the defect above, now linked into
somebody else's process.

### The library builds; the ledger admits

**Admissibility never lives in linked code.** A third party can link a modified
build and post whatever it likes. Any library FDOS publishes is a **constructor
that makes conforming easy — never a gate that prevents non-conforming.**

The ledger revalidates everything, every time, **as though the producer were
hostile**, because eventually one will be. Every check the library performs is a
convenience; every check that matters is repeated at admission, on the ledger's
side of the boundary, assuming nothing about what the caller ran.

**The conformance kit tests the producer, not the ledger.** If passing it ever
becomes the evidence that the ledger will accept a fact, the truth boundary has
left this repository — through the door marked "developer experience", which is
the only door it was ever likely to leave by.

### Delivery is a library, never a service FDOS operates

**A service re-fails `E9`.** If ingesting requires calling something running on
FDOS infrastructure, the open core is not usable alone — it is usable for as
long as FDOS keeps the lights on. That is the dependency the private repository
created, wearing friendlier clothes, and an institutional adopter finds it in
the first due-diligence pass.

The cheaper arguments — no transport decided, D2 open, no persistence, no
deployment — are true and are **not** the reason. Cost arguments lose to anyone
willing to pay; this one does not.

This draws the open-core line exactly: **anyone can ingest; doing it against a
dozen authenticated institutions is what you pay for.**

## Consequences

### Positive

- `E9` acquires a path to being met, rather than an aspiration. The public
  surface stops being one an external producer cannot call.
- The rule survives contact with a hostile producer, because it never assumed a
  cooperative one.
- Identity resolution stays where ADR-0007 put it. A third party never mints,
  never guesses, and never needs to.

### Negative

- **It obliges FDOS to revalidate on every admission**, including work a
  cooperative producer already did. That duplication is the price of not
  trusting linked code, and it is deliberate.
- **A library is a support surface.** Publishing one means owning its
  ergonomics, its versioning and its breakage — for producers this repository
  will never meet.
- **`E9` is still unmet after this decision.** Nothing is built here. A decision
  that a path *should* exist does not make the open core usable alone, and E9
  stays at rung 6 until the path ships.
- **"Anyone can ingest" invites volume this repository has no answer for.**
  Rate, abuse and identity of submitters are D2's, and D2 is open. The library
  shape defers that question rather than answering it.

### Enforcement

**No mechanism is added by this decision**, so its rung is the rung of what it
describes:

| Property | Rung today |
|---|---|
| The public surface accepts a claim | **none — the entry point does not exist** |
| The ledger revalidates as if the producer were hostile | **none — there is no admission path to revalidate in** |
| A producer can check its own conformance | **none — no kit exists** |

**Execution-context question.** This ADR adds no check, and that is the honest
answer rather than an omission. **If every property above were violated
tomorrow, nothing in this repository would report it, because there is nothing
to violate them in.**

The one thing worth saying about a check that does not yet exist: when the
conformance kit arrives it will run in CI as an example
(`examples/README.md` requires every example to compile and run), and it will be
observing **a producer**, not the ledger. A kit that went green while the ledger
rejected the same fact would be a kit testing the wrong subject — that is the
failure to design against, and it is the reason the two are separated in the
decision above rather than in the implementation.

## Alternatives considered

**A service.** Rejected: it recreates the dependency `E9` exists to eliminate.

**Expose `ObserveHolding` and document how to obtain an `EntityId`.** Rejected:
no such procedure exists that does not amount to minting an identity outside
FDOS, which ADR-0007 forbids for reasons that survive any convenience.

**Publish an `AcquisitionEnvelope` / `ProviderObservation` pair for third-party
producers.** Out of scope, and probably not what `E9` needs. A producer that
already holds data does not model "artifact plus observation" — that is a
pipeline shape belonging to a producer that acquires. Whether an acquisition
contract is separately worth publishing is a different question with its own
RFC.

**Let the library enforce admissibility and have the ledger trust it.**
Rejected: a linked library is the producer's code. Treating it as a gate moves
the truth boundary outside this repository.

## Notes

**The roadmap is not edited by this ADR**, as RFC-0010 §7 promised. The
redefinition is real — if the ingress is an entry point plus a kit, M8 *is* the
public ingress rather than a milestone it follows, and M8's deliverable 2 is the
same call as this decision's §1. That edit lands in its own change citing this
one.

The rename obligation on `fdos.kernel.v2` recorded in
[`../ecosystem/roadmap.md`](../ecosystem/roadmap.md) **binds this work**: the
public ingress must not publish while `SourceRef` is still named `value`,
because after that third-party producers depend on `GetValue()`.

D2 — who may send a fact — remains open and is the question this decision
defers rather than answers.
