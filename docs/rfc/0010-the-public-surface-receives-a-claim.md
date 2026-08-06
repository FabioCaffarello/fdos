---
id: RFC-0010
title: The public surface receives a claim, not a resolved identity
status: Draft
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0010 — The public surface receives a claim, not a resolved identity

> **Draft.** It has a named precondition — see §6. It is written now so that
> acceptance is mechanical on the day that precondition lands, and it is not
> accepted before, because accepting against unsettled content is how a
> hypothesis becomes a decision without an ADR.

## 1. The problem

`E9` requires the open core to build, test, run and **deliver value** with the
private repository absent. It was admitted unmet at `ecosystem/v0.3.0` and sits
at rung 6.

The reason is narrower and worse than "there is no ingestion path":

```go
type ObserveHoldingCommand struct {
    Account    identity.ID
    Instrument identity.ID
```

`app.Ledger` exposes `ObserveHolding`, `CorrectFact` and `ProjectPosition`. All
three take an already-resolved `identity.ID`, and by
[ADR-0007](../adr/0007-internal-deterministic-identity.md) and
[ADR-0022](../adr/0022-minting-an-identity-is-a-fact.md) **no external producer
may mint one** — deriving an identity from a ticker makes the ticker the primary
key, and the day the world reuses one, two instruments merge silently inside an
append-only ledger.

**The only public application-layer entry point is structurally unusable from
outside.** Not undocumented, not inconvenient: unusable, by a rule this
repository is right to have.

What is *not* missing is the contract. `libs/contracts v0.3.0` already publishes
`IdentifierClaim`, `HoldingClaimed`, `Envelope` and `Provenance` — everything a
producer needs to express a claim. The gap is that nothing public accepts one.

## 2. Proposal

**The public surface receives a claim, not a resolved identity.**

An external producer asserts identifiers — `{scheme, value}`, exactly as it read
them — and FDOS resolves them internally, where ADR-0007 says resolution lives
and where the mint that results is itself a recorded fact (ADR-0022).

This is the whole subject of this RFC. Everything else here follows from it.

It is also not a new mechanism. `domain.Resolve`, `domain.MintFor` and
`domain.DeriveHoldingObserved` already exist and already do this; nothing in
`app/` calls them. The proposal is to expose the path that the domain already
implements, to the only kind of caller that can use it.

### What the producer supplies

| Supplies | Does not supply |
|---|---|
| `IdentifierClaim` per entity, verbatim | any `EntityId` |
| the observed quantity and its effective interval | knowledge time — FDOS assigns it (ADR-0009) |
| provenance of its own acquisition | the resolution, the mint, or the derivation |

### Exact-match semantics are a producer obligation

`Resolve` matches on **exact equality** of `{scheme, value}`. `identity.NewClaim`
rejects a non-canonical *scheme* and leaves the *value* verbatim and untouched,
deliberately: canonicalising `"PETR4 "` is a resolver's decision to record, not a
parser's to make silently.

The consequence belongs in the conformance kit in bold: **a producer must render
a given value identically forever.** Emit `"PETR4"` today and `"PETR4 "`
tomorrow and you mint two entities for one instrument, silently, and nothing
rejects it because both claims are well-formed.

## 3. The library builds; the ledger admits

The corollary that keeps §5 from becoming a security hole.

**Admissibility never lives in linked code.** A third party can link a modified
build and post whatever it likes. Any library FDOS publishes is therefore a
**constructor that makes conforming easy — never a gate that prevents
non-conforming.**

The ledger revalidates everything, every time, **as though the producer were
hostile**, because eventually one will be. Every check the library performs is a
convenience; every check that matters is repeated at admission, on the ledger's
side of the boundary, with no assumption that the caller ran anything.

**The conformance kit tests the producer, not the ledger.** If passing the kit
ever becomes the evidence that the ledger will accept a fact, the truth boundary
has moved outside this repository — which is the failure the open core exists to
prevent, arriving through the door marked "developer experience".

## 4. Can a human with a file produce honest provenance?

`E4` makes provenance mandatory, and it should. A third party with an exported
statement has no session, no acquired artifact and no plugin — but *"I exported
this from my bank on this date, and this is its hash"* is honest provenance and
ought to be sufficient.

So the question is not "envelope or no envelope". It is: **is there a minimal
provenance form a producer with a file can fill in without lying?**

Today, no. This is a **contract finding**, reported rather than absorbed:

```go
func NewProvenance(source Source, collectedAt Instant, interpreter Interpreter, …)
    if interpreter.name == "" {
        return Provenance{}, fmt.Errorf("%w: interpreter", ErrIncomplete)
```

`Interpreter` is **required at rung 1** — the constructor refuses without it —
and `NewInterpreter` requires both a name and a version, on the correct
reasoning that a parser version must be pinned so a report regenerates with the
interpreter of its time.

A producer that did not interpret anything programmatically has no honest value
to put there. Filling `{name: "manual", version: "1"}` is a lie-shaped field: it
asserts a versioned interpretation that does not exist and cannot be replayed.

**That is the provenance shape over-fitting its first producer.** Every producer
so far runs code, so every producer so far has an interpreter, so the field
looked universal. It is not, and this is the first concrete case.

This RFC does not settle it — the fix belongs with D4, which is about the same
message. It records the finding, and notes that the options are visibly
different in cost: an optional interpreter weakens replay for everyone, whereas
an explicit "not programmatically interpreted" value keeps the field required
and honest. The second looks right and is not decided here.

## 5. Delivery: a library, never a service

Packaging, deliberately after the decision rather than in front of it.

**A service FDOS operates re-fails `E9`.** If ingesting requires calling
something that runs on FDOS infrastructure, the open core is not usable alone —
it is usable for as long as FDOS keeps the lights on. That is the same
dependency the private repository created, wearing friendlier clothes, and an
institutional adopter finds it in the first due-diligence pass.

The cheaper arguments — no transport decided, D2 open, no persistence, no
deployment — are true and are not the reason. Cost arguments lose to anyone
willing to pay. *"It recreates the dependency `E9` exists to eliminate"* does
not.

So the public ingress is:

1. **A public entry point taking a claim** (§2), in `libs/ledger/app`.
2. **A stated definition of admissible provenance** (§4), so conformance is
   checkable rather than folklore.
3. **A conformance kit in `examples/`** — synthetic fixtures and a suite a third
   party runs against its own producer. That directory already forbids real
   institution data and requires every example to run in CI, which is the right
   contract for this and is why the kit belongs there rather than in `libs/`.

This is the open-core line drawn exactly: **anyone can ingest; doing it against
a dozen authenticated institutions is what you pay for.**

## 6. Precondition — D4, which is ours

**D4 — what a `SourceRef` must resolve to — is a precondition of accepting this
RFC**, because §4's admissibility statement cannot be written without it.

It is not an external blocker. **Provenance is `fdos`'s concern and D4 is
`fdos`'s decision.** A proposal has been published to this repository and it is
persuasive — specify the form and the assertions, specify nothing about the
referent — but adopting it is a decision this repository makes and currently
owes.

If D4 blocks `E9`, D4 rises in priority. That is a sequencing decision for this
repository, not an impediment to report.

**Acceptance is mechanical once D4 lands:** §4's finding folds into the same
decision, and nothing else in this RFC depends on it.

## 7. Consequence — M8 is redefined, not extended

Declared, not actioned. The roadmap is untouched by this RFC and changing it is
its own ADR.

M8 is currently "ingestion, stopping at the `libs/` boundary". If the public
ingress is an entry point plus a conformance kit, then **M8 *is* the public
ingress**, plus its documentation — not a milestone the ingress follows. The
difference between M8 as written and `E9` as satisfied is the kit and the
admissibility statement.

Deliverable 2 of M8 and §2 of this RFC are the same call.

## 8. Alternatives considered

**A service.** Rejected on §5: it recreates the dependency `E9` exists to
eliminate.

**Expose `ObserveHolding` publicly and document how to obtain an `EntityId`.**
Rejected: there is no such procedure that does not amount to minting one
outside, which ADR-0007 forbids for reasons that survive any convenience.

**Publish an `AcquisitionEnvelope` / `ProviderObservation` pair for third-party
producers.** Not rejected — **out of scope, and probably not what `E9` needs.**
A producer that already holds data does not model "artifact plus observation";
that is a pipeline shape belonging to a producer that acquires. A previously
recorded rejection of that contract was invalidated in its premise by `E9`
supplying a producer; this RFC narrows what that producer actually requires,
which is less. Whether an acquisition contract is separately worth publishing is
a different question and wants its own RFC.

**Let the library enforce admissibility and trust it.** Rejected on §3: a linked
library is the producer's code, and treating it as a gate moves the truth
boundary outside this repository.
