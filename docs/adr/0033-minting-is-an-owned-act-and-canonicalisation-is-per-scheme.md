---
id: ADR-0033
title: Minting is an owned act, and canonicalisation is per scheme
status: Accepted
date: 2026-08-06
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0033 — Minting is an owned act, and canonicalisation is per scheme

## Context

Records what [RFC-0013](../rfc/0013-minting-is-an-owned-act-and-canonicalisation-is-per-scheme.md)
settled, closing the first two open items of
[issue #57](https://github.com/FabioCaffarello/fdos/issues/57).

[ADR-0022](0022-minting-an-identity-is-a-fact.md) decided that minting an
identity **is** a fact and that a connector emits a claim. It did not decide who
performs the mint, when, or what a claim's value is canonicalised to first.

The consequence was total. `Resolve`, `MintFor` and `DeriveHoldingObserved` all
exist and **nothing called them**. A claim admitted through `AcceptHoldingClaim`
could not become an observation, because no act brought an identity into
existence. FDOS could receive financial data and do nothing with it.

Two constraints made this a decision rather than an obvious choice.

**A wrong per-scheme rule merges two real instruments.** That is the failure
ADR-0007 exists to prevent, arriving through the fix rather than through the gap.
It is the same class as `contract/breaking`: a human decision.

**The problem was smaller than the register claimed.** `canonicaliseSeed`
already folds case and collapses whitespace runs before hashing, so `"PETR4"`
and `"PETR4 "` derive the *same* identity — the earlier record that they mint two
entities silently was wrong and is corrected in `docs/blocked.md`. What actually
happened was two `EntityMinted` facts carrying one identity: recorded
duplication, not silent corruption. The surviving gap is variation the generic
floor cannot fold — internal spacing, punctuation, suffixes — and it is a
question of **semantics per scheme**, not of shape.

Constitution §1 (Financial Truth) and §6 (Provenance) are what is at stake: an
identity with no owner and no recorded origin is a truth claim nobody made.

## Decision

### Minting is an explicit application act

FDOS adds one use case, `Ledger.MintIdentity`. It is the only thing in the
repository that appends an `EntityMinted` fact.

It is **not admission**: `AcceptHoldingClaim` mints nothing and continues to
mint nothing. An identity that came into existence because a stranger submitted
a claim is an identity nobody chose.

It is **not inspection**: `UnresolvedClaims` mints nothing and continues to mint
nothing. Minting on inspection would make the act of looking change the ledger.

**It resolves first and refuses.** If the claim already resolves at the
command's coordinate, `MintIdentity` returns `ErrAlreadyMinted` naming the
existing identity. It does not silently return that identity: a second mint that
appears to succeed is how a caller learns to stop checking.

The mint records its own derivation — method `ledger.MintFor`, the claim fact it
answers when there is one, and the canonicalisation ruleset version as a
parameter — so its provenance is `Derived` rather than `Observed`. An identity is
computed from a claim by a versioned method, which is what `Derived` means.

### When: on demand, and the ledger answers afterwards

The operational loop is `UnresolvedClaims` → decide → `MintIdentity`. Each mint
is one fact with one envelope, so *when* each identity came into existence is a
query rather than a log line.

There is deliberately **no batch use case**: one provenance record covering many
independent decisions is the shape an audit cannot take apart. There is
deliberately **no automatic resolver** minting on a timer: that is minting
without an owner, and unlike the other rejected options it cannot be un-shipped
once a ledger contains its output.

### On whose authority: recorded, never checked

**This question has no technical answer today, and FDOS does not invent one.**

At rung 1, a mint carries a full envelope, so it cannot exist without naming a
`Source` and an `Interpreter`. Who minted is always recorded.

FDOS **cannot** verify that the named authority is who it says, or that it was
entitled to mint. There is no actor model, no permission model and no signature
check. Building one is adjacent to **D2** and is not decided here.

So the honest statement, recorded as such: **the authority boundary today is the
process boundary.** `MintIdentity` is a Go method; whoever can call it can mint.
`apps/` is empty by design, so there is currently no caller. A composition root
exposing `MintIdentity` on an unauthenticated endpoint would satisfy every
mechanism in this decision. That is rung 6 and is recorded as rung 6.

### Canonicalisation is per scheme, versioned and closed

The governing rule:

> **A canonicalisation rule may exist only for a scheme whose issuing standard
> defines a canonical form, and a rule may never alter a value already in that
> form.**

This is per scheme by construction — a standard belongs to a scheme, and a
provider's padding habit is not a standard, so a provider-keyed rule cannot be
written. It also makes the safety argument provable: a fold that is the identity
on every canonical value cannot merge two valid distinct values. Only invalid
renderings can collapse, which is the intent.

The accepted rule set:

| Scheme | Standard | Canonical form | Rule |
|---|---|---|---|
| `isin` | ISO 6166 | 12 contiguous alphanumerics | strip all whitespace |
| `cusip` | CUSIP Global Services | 9 contiguous alphanumerics | strip all whitespace |
| `sedol` | LSE | 7 contiguous alphanumerics | strip all whitespace |
| `figi` | OMG FIGI | 12 contiguous alphanumerics | strip all whitespace |
| everything else | *none* | — | **none — the generic floor only** |

Uppercasing is absent because `canonicaliseSeed` already does it. These rules add
exactly one thing to the generic floor: removing whitespace the standard says
cannot be part of the value. That is deliberately thin — it is the part that can
be defended.

**`ticker` gets no rule, and neither does `symbol` or `account_number`.**
`"PETR4"` versus `"PETR4.SA"` is a venue distinction; `"BRK.B"` versus `"BRK B"`
is a guess. There is no standards body to appeal to, so any such rule is a
judgement about whether two things are the same thing — which is a **merge**, and
ADR-0007 already decided merges are recorded as `EntitiesIdentified` and never
performed.

### Canonicalisation is applied before `Derive`, never inside it

```
claim.Value() ──► per-scheme fold ──► identity.Derive ──► canonicaliseSeed ──► UUIDv5
```

`canonicaliseSeed` stays the generic floor, unexported and scheme-blind. A
scheme-aware `canonicaliseSeed` would make every caller of `Derive` — including
`ledger_stream`, which has no schemes at all — depend on the identifier
vocabulary, and would attach the "versioned by behaviour" property to a table
that changes.

### Resolution folds both sides

`Resolve` matches `rules.Fold(minted.BornFrom)` against `rules.Fold(claim)`
rather than comparing bytes. The invariant:

> **Two claims resolve to the same identity if and only if minting them would
> derive the same identity.**

Two things do not change. **`Claim.Equal` stays byte equality** — it is a
value-type operator, and two claims differing by a byte are two claims. What
gains a stated, versioned rule is *resolution*, which `resolve.go` already said
is where sameness is decided. **`EntityMinted.BornFrom` stays verbatim**: it is
the birth certificate, and the folded form seeds `Derive` while the raw form is
what is recorded.

This narrows what `resolve.go` documented as byte-exact matching. It amends an
open item rather than reversing an accepted decision — ADR-0022 recorded
canonicalisation as unresolved, and B-007 has carried it as open since.

### The ruleset version is not put on the wire

No field is added to `ledger.payload.v1.EntityMinted`. There is no
`contracts@v0.4.0` and no downstream pin moves.

Two reasons, the first a measurement. `kernel.v1.DerivationRef` carries only
`content_hash`, so a derivation's parameters do not survive the wire — recording
the ruleset version there would make it visible in-process and invisible after a
decode, which looks recorded without being. And it is not needed: the question a
rule change raises is *"which existing mints now collide?"*, which is computable
from `BornFrom` alone, and `BornFrom` is on the wire.

## Consequences

### Positive

- **A claim can become an observation.** The acquisition path works end to end
  for the first time.
- Two renderings of one ISIN mint one identity, and the second attempt is
  refused rather than recorded as a duplicate.
- *Who minted this, and when* is answerable from the ledger, with a derivation
  naming the method and the claim fact.
- A wrong rule is confined to a shape that cannot merge two valid values, and a
  provider-shaped rule is unwritable rather than merely discouraged.

### Negative

- **Changing a shipped rule changes what resolves, retroactively**, because
  resolution folds at read time. That is a `contract/breaking`-class act needing
  its own ADR, and where it merges two previously distinct identities it needs
  recorded `EntitiesIdentified` facts — a path that is unused, untested and has
  no codec. Proposing it as the remedy means proposing to build it.
- **Every caller of `Resolve`, `Unresolved`, `MintFor` and
  `DeriveHoldingObserved` now passes a ruleset, and `NewLedger` requires one.**
  Deliberate: the deployed ruleset determines what merges, so it is a
  composition-root decision rather than a default. It is also a breaking change
  to `libs/ledger` and `libs/kernel` Go APIs.
- **The authority gap is not closed.** Anyone who can call `MintIdentity` can
  mint. Recorded at rung 6 above.
- **The rule set is thin, and the gaps are the common cases.** `ticker` and
  `account_number` are what producers actually emit, and both get nothing. The
  ISIN case this fixes may be rarer in practice than the ticker case it declines.
- **Nothing yet notices that claims are accumulating unresolved.** ADR-0022 named
  this gap; `UnresolvedClaims` makes it askable and nothing asks.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| A rule exists only for a scheme with an issuing standard | 1 | `NewRule` rejects an empty standard or a non-positive canonical length |
| A rule cannot encode a provider's habit | 1 | `Fold` is a closed set; a rule composes folds and carries no function |
| A fold cannot alter an already-canonical value | 1 | `NewRule` rejects a fold whose altered character class meets the canonical class |
| Canonicalisation happens before `Derive` | 2 | `canonicaliseSeed` stays unexported and scheme-blind |
| Resolution agrees with minting | 3 | test: claims resolve equal ⟺ `MintFor` derives equal |
| A second mint for a resolved claim is refused | 3 | test on `MintIdentity` returning `ErrAlreadyMinted` |
| Admission and inspection cannot mint | 1 | neither takes a `Kind`, and neither reaches anything that mints |
| The content of a rule is correct | **6** | human decision, per ADR |
| The named minting authority is entitled to mint | **6** | nothing checks it |

The last two are the honest weak points. The first of them is why RFC-0013
existed: whether stripping whitespace from an ISIN is right is a judgement about
financial truth, and the mechanisms above constrain its shape without making it.

## Alternatives considered

Recorded in full in RFC-0013. The three that were closest:

**Leave resolution byte-exact and let variants mint duplicates.** Today's
behaviour, and **not broken** — two `EntityMinted` facts, one identity,
deterministic reads. This was the strongest alternative. Rejected because every
vendor rendering variant then needs its own human minting act, so the operator's
workload scales with vendor sloppiness while the truth gained is zero, and the
ledger accumulates mint facts recording only that a provider typed a space.

An earlier draft of the RFC called this option a deadlock. That was wrong and was
corrected before the decision: it is coherent, provided `MintIdentity` refuses on
whether the claim resolves rather than on whether the identity exists. Only that
combination — the stronger refusal with byte-exact matching — is incoherent.

**Express rules as functions in a registry.** Smallest type, most flexible.
Rejected because a `func(string) string` can express anything, including a
provider's padding habit, so the boundary test that matters most would be
enforced only by review. The closed `Fold` set exists to make the wrong rule
unwritable.

**Rules in configuration rather than in code.** A new institution would need no
release. Rejected outright: it puts a decision that can merge two real
instruments in a file with no review, no ADR and no test, inverting the
enforcement ladder for the highest-risk rule in the repository.

Also rejected: a scheme-aware `canonicaliseSeed`; an automatic minting resolver;
and recording the ruleset version on `EntityMinted`.

## Notes

Additive to no contract. `contracts` is untouched, so the pinned downstream
consumer is unaffected.

Still open on [issue #57](https://github.com/FabioCaffarello/fdos/issues/57)
after this decision:

- vocabulary governance — `"ticker"` versus `"symbol"` for one concept, which is
  the same shape one level up and which no type solves;
- `HoldingObserved` provenance must be `Derived` — rung 6, as ADR-0022 recorded;
- no `IdentifierAssertion` codec, because nothing produces one yet;
- the `EntitiesIdentified` path, now load-bearing as the remedy for a rule
  change, and still unused, untested and without a codec;
- who notices that claims are accumulating unresolved.
