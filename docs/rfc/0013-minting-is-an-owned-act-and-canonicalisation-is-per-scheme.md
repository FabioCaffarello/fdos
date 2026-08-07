---
id: RFC-0013
title: Minting is an owned act, and canonicalisation is per scheme
status: Accepted
date: 2026-08-06
authors:
  - "@FabioCaffarello"
---

# RFC-0013 — Minting is an owned act, and canonicalisation is per scheme

> **Accepted**, recorded by
> [ADR-0033](../adr/0033-minting-is-an-owned-act-and-canonicalisation-is-per-scheme.md),
> which also settles the four open questions below. The rule set accepted is the
> one proposed in §4 — four standard schemes — and resolution folds both sides
> (§5).

## Summary

[ADR-0022](../adr/0022-minting-an-identity-is-a-fact.md) decided that minting an
identity **is** a fact and that a connector emits a claim. It did not decide who
performs the mint, when, or what a claim's value is canonicalised to first. Those
three are the first two items of
[issue #57](https://github.com/FabioCaffarello/fdos/issues/57), and they are
**one decision**: the act is what applies the canonicalisation, and the
canonicalisation is what lets the act refuse a second mint.

The consequence of leaving them open is total. `Resolve`, `MintFor` and
`DeriveHoldingObserved` all exist and **nothing calls them**. A claim admitted
through `AcceptHoldingClaim` cannot become an observation, because there is no
act that brings an identity into existence. FDOS can currently receive financial
data and do nothing with it.

This cannot be settled by an ADR alone because the third question has a failure
mode worth more than the feature: **a wrong per-scheme rule merges two real
instruments**, which is precisely the failure ADR-0007 exists to prevent,
arriving through the fix rather than the gap.

## Motivation

### What is at stake

Constitution §1 (Financial Truth) and §6 (Provenance). An identity that comes
into existence with no owner and no recorded origin is a truth claim nobody
made. An identity that comes into existence *twice* for one instrument, or once
for *two* instruments, is worse: it is a wrong answer that looks like an answer.

### A correction that changes the shape of the problem

Three earlier records — the M8 gate, `B-007`, and a comment to the downstream
consumer on #28 — stated that a producer rendering `"PETR4"` and then `"PETR4 "`
mints **two entities for one instrument, silently**. That is false, and the
correction is already committed to the register.

Measured rather than re-read:

| Pair | Same derived identity? |
|---|---|
| `ticker:PETR4` vs `ticker:PETR4 ` | **yes** |
| `ticker:PETR4` vs `ticker:petr4` | **yes** |
| `ticker:PETR4` vs `ticker: PETR4` | no |
| `ticker:PETR4` vs `ticker:PETR 4` | no |
| `ticker:PETR4` vs `ticker:PETR4.` | no |

`MintFor` derives through `identity.Derive`, whose `canonicaliseSeed` collapses
whitespace runs and folds case before hashing. The example used to argue the
point was the one case that already worked.

What actually happens with `"PETR4 "` today: the claim does not **resolve**
against the existing mint, because `Claim.Equal` is byte equality by design — so
a second mint is performed, and it derives **the same identity**. The result is
two `EntityMinted` facts carrying one identity: *a defect the ledger records
rather than hides*, with `Resolve` deterministic on the first visible mint.

**Recorded duplication, not silent corruption.** That distinction is why this RFC
is about semantics rather than about a bug.

### The gap that survives, and it is sharper

`canonicaliseSeed` is **generic and knows nothing about schemes**. Variation it
cannot fold derives genuinely different identities: internal spacing,
punctuation, suffixes. A vendor rendering an ISIN as `"BR PETR ACNPR0"` and
another rendering it as `"BRPETRACNPR0"` produce two identities for one
instrument, and no amount of generic folding fixes that without also being
willing to fold `"PETR 4"` into `"PETR4"`, which is a different and much worse
claim.

That asymmetry is the whole finding: **the correct fold depends on the scheme**,
and a generic function cannot know it.

### Is this retrofittable?

Partly, and the honest answer matters.

`canonicaliseSeed`'s own comment says it is *"versioned by behaviour rather than
by a version field… existing identifiers are unaffected — they were assigned
once and are never re-derived."* So changing canonicalisation does **not**
re-partition history. What it does is allow one real-world instrument to hold
two identities across the change, reconcilable only by a recorded merge.
`kernel.v1.EntitiesIdentified` exists for exactly that and is unused, untested
and has no codec.

So: the *mechanism* is retrofittable at a stated cost. The *rules* get more
expensive the longer they are deferred, because every mint made under a missing
rule is a mint that may later need merging.

## Design

### 1. Who mints: an explicit application act

FDOS adds one use case, `Ledger.MintIdentity`. It is the only thing in the
repository that appends an `EntityMinted` fact.

```go
type MintIdentityCommand struct {
    Stream string
    Kind   identity.Kind
    Claim  identity.Claim

    // Answers is the claim fact this mint answers, when there is one. It is
    // optional because RFC-0007 established that an account is often minted
    // from operator configuration before any observation arrives.
    Answers   domain.Ref
    Effective temporal.Interval

    Source      provenance.Source
    CollectedAt temporal.Instant
    Interpreter provenance.Interpreter
    Confidence  provenance.Confidence
}
```

Three properties, each defending something already decided:

- **It is not admission.** `AcceptHoldingClaim` mints nothing and keeps minting
  nothing. An identity that came into existence because a stranger submitted a
  claim is an identity nobody chose.
- **It is not inspection.** `UnresolvedClaims` mints nothing and keeps minting
  nothing. Minting on inspection would make the act of looking change the
  ledger, and an identity would exist because somebody ran a report.
- **It resolves first, and refuses.** If the claim already resolves at the
  command's coordinate, `MintIdentity` returns `ErrAlreadyMinted` naming the
  existing identity. It does **not** silently return the existing identity: a
  second mint that looks like it succeeded is how a caller learns to stop
  checking.

### 2. When: on demand, and the ledger answers afterwards

`UnresolvedClaims` already lists what is waiting. The operational loop is: look
at the list, decide, mint. Each mint is one fact with one envelope, so *when*
each identity came into existence is a query, not a log line.

There is deliberately **no batch use case**. A batch is a loop in the caller, and
making it a single call would tempt a single provenance record to cover many
independent decisions — which is the shape that makes an audit unable to
distinguish them.

There is deliberately **no automatic resolver** minting for unresolved claims on
a timer. It would reintroduce minting-without-an-owner through the back door,
and it is the one option that cannot be un-shipped once a ledger contains its
output.

### 3. On whose authority: recorded, never checked

**This question has no technical answer today, and this RFC does not invent
one.**

What FDOS *can* do, at rung 1: a mint carries a full envelope, so it cannot
exist without naming a `Source` (content-addressed) and an `Interpreter`. Who
minted is therefore always recorded.

What FDOS **cannot** do: verify that the named authority is who it says, or that
it was entitled to mint. There is no actor model, no permission model, and no
signature check. Building one is adjacent to **D2** and is not this slice.

So the honest statement is: **the authority boundary today is the process
boundary.** `MintIdentity` is a Go method; whoever can call it can mint.
`apps/` is empty by design, so there is currently no caller at all — which means
the boundary is real but is enforced by deployment rather than by FDOS.

This is rung 6, and it should be recorded as rung 6 rather than dressed as
something stronger. A composition root that exposes `MintIdentity` on an
unauthenticated endpoint would satisfy every mechanism in this design.

### 4. What a value is canonicalised to: per scheme, versioned, closed

#### The governing rule

> **A canonicalisation rule may exist only for a scheme whose issuing standard
> defines a canonical form, and a rule may never alter a value already in that
> form.**

That single sentence does most of the work:

- It is **per scheme by construction**. A standard belongs to a scheme. A
  provider's padding habit is not a standard, so a rule cannot be written for
  it — which is the second-provider boundary test, satisfied structurally rather
  than by review.
- It makes the safety argument **provable rather than asserted**. If a fold is
  the identity on every value already in canonical form, then it cannot merge
  two *valid distinct* values. Only invalid renderings can collapse, and
  collapsing two invalid renderings of the same thing is the entire point.
- It explains, without special pleading, why `ticker` gets **nothing**.

#### What that yields

| Scheme | Standard | Canonical form | Rule |
|---|---|---|---|
| `isin` | ISO 6166 | 12 contiguous alphanumerics | strip all whitespace |
| `cusip` | CUSIP Global Services | 9 contiguous alphanumerics | strip all whitespace |
| `sedol` | LSE | 7 contiguous alphanumerics | strip all whitespace |
| `figi` | OMG FIGI | 12 contiguous alphanumerics | strip all whitespace |
| `ticker` | *none* | — | **none** |
| `symbol` | *none* | — | **none** |
| `account_number` | *none* (per-institution) | — | **none** |
| anything else | *none* | — | **none** |

Uppercasing is not listed because `canonicaliseSeed` already does it. **These
rules add exactly one thing to the generic floor: removing whitespace the
standard says cannot be part of the value.**

That is a thin result and it is deliberately thin. It is the part that can be
defended.

#### Why `ticker` gets nothing, stated plainly

`"PETR4"`, `"PETR4.SA"`, `"PETR4:BZ"` — the suffix names a venue. Stripping it
merges listings. `"BRK.B"`, `"BRK B"`, `"BRK-B"` are probably one instrument;
`"PETR 4"` and `"PETR4"` probably are too — but *probably* is the word that has
no place in a function that decides identity. There is no standards body to
appeal to, so any ticker rule is a judgement, and a judgement about whether two
things are the same thing is a **merge**. ADR-0007 already decided how merges
go: recorded as `EntitiesIdentified`, never performed.

The `"ticker"`-versus-`"symbol"` item on issue #57 is the same shape one level
up — cross-scheme rather than intra-scheme — and this RFC does not close it.

#### Applied *before* `Derive`, never inside it

```
claim.Value()  ──►  per-scheme fold  ──►  identity.Derive  ──►  canonicaliseSeed  ──►  UUIDv5
                    (this RFC)                                  (generic floor, unchanged)
```

`canonicaliseSeed` stays the generic floor. It gains nothing, loses nothing, and
keeps its property of being versioned by behaviour. A scheme-aware
`canonicaliseSeed` would make every caller of `Derive` — including kinds that
have no schemes at all, like `ledger_stream` — depend on the identifier
vocabulary.

#### Shape of the type

In `libs/kernel/identity`, because both the ledger's minting and any future
resolver need it, and because it is about identity seeds.

```go
// A Fold is one declared transform. The set is closed: a rule composes folds,
// it does not carry a func.
type Fold uint8
const (
    FoldUnspecified Fold = iota
    StripWhitespace
)

type Rule struct {
    scheme   string
    standard string // "ISO 6166" — the issuing standard whose form this recovers
    length   int    // canonical length; the canonical set is uppercase
                    // alphanumerics of exactly this length
    folds    []Fold
}

type Ruleset struct {
    version string
    rules   map[string]Rule
}

func (r Ruleset) Fold(c Claim) Claim   // scheme preserved, value folded
func (r Ruleset) Version() string
func Canonicalisation() Ruleset        // the ruleset this build ships
```

The closed `Fold` set is the mechanism that makes a provider-shaped rule
unwritable. Today it has exactly one member, which is honest: exactly one
transform is currently defensible. Adding a member is an ADR.

`NewRule` rejects, at construction:

- an empty `standard` — no standard, no rule;
- a non-positive `length` — without a canonical set there is nothing to check a
  fold against;
- a fold whose altered character class intersects the canonical set.
  `StripWhitespace` alters only values containing whitespace; the canonical set
  is alphanumeric; disjoint, so it is admitted. A hypothetical suffix-stripping
  fold would be rejected for `isin` — and could not be reached for `ticker` at
  all, because `ticker` cannot carry a rule.

#### The predicate is a guard on rules, never a filter on claims

Stated because somebody will otherwise wire it up. A claim whose value is not
well-formed for its scheme — a nine-character `isin` — is still **admitted,
still folded, still mintable**. RFC-0007 decided that claims reach the ledger
and that resolution is not a precondition of appending. The `length` predicate
exists only to prove a fold is safe. Using it to reject a claim would move a
truth decision to the door, which is what §6 exists to prevent.

### 5. Resolution folds both sides

This is the part that changes recorded behaviour, so it is stated on its own.

`Resolve` currently matches on `minted.BornFrom.Equal(claim)` — byte equality.
Under this proposal it matches on `rules.Fold(minted.BornFrom)` against
`rules.Fold(claim)`.

The invariant this buys:

> **Two claims resolve to the same identity if and only if minting them would
> derive the same identity.**

The refusal in §1 is what forces the choice, and it is worth being exact about
why, because an earlier draft of this section overstated it. `MintIdentity`
could refuse on either of two predicates:

| Refuse when… | With byte-exact resolution | With folding resolution |
|---|---|---|
| …the claim **resolves** | coherent. `"PETR4 "` does not resolve, so it mints again: two facts, one identity, and afterwards it resolves. Today's behaviour, plus folded seeds. | coherent, and one fact per identity |
| …minting would derive an **already-minted identity** | **incoherent** — `"PETR4 "` can never be minted and can never resolve | coherent |

So byte-exact resolution is *not* a deadlock; it is the recorded-duplication
behaviour that already exists, carried forward. What it cannot do is combine
with the stronger refusal. The argument for folding is therefore about cost
rather than about correctness: without it, every vendor rendering variant needs
its own `EntityMinted` fact and its own human minting act, forever.

Two things that do **not** change:

- **`Claim.Equal` stays byte equality.** It is a value-type operator and two
  claims differing by a byte are two claims. What changes is what *resolution*
  means by sameness — and `resolve.go` already said resolution is where that is
  decided: *"deciding they are the same thing is resolution."* This gives that
  sentence a stated, versioned rule instead of a byte comparison.
- **`EntityMinted.BornFrom` stays verbatim.** It is the birth certificate: what
  the provider actually said. The folded form is what seeds `Derive`; the raw
  form is what is recorded.

### 6. Why the ruleset version is not on the wire

The obvious move is to record, on each mint, which ruleset minted it. This RFC
proposes **not** to, and the reason is worth stating because the alternative
costs a contract release.

Two findings:

1. **`kernel.v1.DerivationRef` carries only `content_hash`.** A derivation's
   method and parameters do not survive the wire. So recording the ruleset
   version as a derivation parameter would make it visible in-process and
   invisible after a decode — the worst of both, since it would look recorded.
2. **It is not needed.** The question a rule change actually raises is *"which
   existing mints now collide?"*, and that is computable from `BornFrom` alone:
   fold every recorded `BornFrom` under the new ruleset and look for collisions.
   `BornFrom` is on the wire. Knowing which version minted each identity answers
   a different, less useful question.

So no field is added to `ledger.payload.v1.EntityMinted`, there is no
`contracts@v0.4.0`, and no downstream pin moves. The ruleset version is still
recorded as a derivation parameter on the mint, for the in-process trace and its
content address; this RFC simply does not claim it is durable.

### 7. What a ruleset change costs

Adding a rule for a **new** scheme cannot change existing resolution for any
other scheme. Rules are keyed by scheme and unregistered schemes get the floor.

**Changing an existing rule can change what resolves**, including retroactively,
because resolution folds at read time. That is a `contract/breaking`-class act:
it needs its own ADR, and where the change merges two previously distinct
identities it needs recorded `EntitiesIdentified` facts — which means building
that path, since it is currently unused, untested and has no codec.

This is the honest cost and it is larger than "history is not re-partitioned"
suggested. History is not re-partitioned; *reads* are.

### Not covered

Transport, persistence, `apps/`, D2 (authorisation), the validator binary, the
`ticker`-versus-`symbol` vocabulary question, the `EntitiesIdentified` codec,
and `HoldingObserved`'s rung-6 obligation to carry `Derived` provenance. All
five of the last remain open on issue #57.

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| A rule exists only for a scheme with an issuing standard | 1 | `NewRule` rejects an empty `standard` or a non-positive `length` |
| A rule cannot encode a provider's habit | 1 | `Fold` is a closed set; a rule composes folds and carries no func |
| A fold cannot alter an already-canonical value | 1 | `NewRule` rejects a fold whose altered class meets the canonical class |
| Canonicalisation happens before `Derive` | 2 | `canonicaliseSeed` stays unexported and scheme-blind; the fold is applied by the caller |
| Resolution agrees with minting | 3 | property test: claims resolve equal ⟺ `MintFor` derives equal |
| A second mint for a resolved claim is refused | 3 | test on `MintIdentity` returning `ErrAlreadyMinted` |
| Admission and inspection cannot mint | 1 | they take no `Kind` and call nothing that mints; existing tests stay |
| The content of a rule is correct | **6** | human decision, per ADR |
| The named minting authority is entitled to mint | **6** | nothing checks it; see §3 |

The last two are the honest weak points. The first of them is why this is an RFC
rather than a commit: *whether stripping whitespace from an ISIN is right* is a
judgement about financial truth, and the mechanisms above constrain its shape
without making it.

## Alternatives

**Leave resolution byte-exact and let variants mint duplicates.** This is
today's behaviour and it is **not broken** — two `EntityMinted` facts, one
identity, deterministic reads — so it is the strongest alternative here and is
a genuine option rather than a formality. It pairs only with the weaker refusal
(see §5). Rejected because every vendor rendering variant then needs its own
human minting act, so the operator's workload scales with vendor sloppiness
while the truth gained is zero, and because the ledger accumulates mint facts
that record nothing except that a provider typed a space.

**Make `canonicaliseSeed` scheme-aware.** One function, no new type. Rejected:
it puts the identifier vocabulary inside the generic identity primitive, so
`ledger_stream` identifiers — which have no scheme — would depend on it, and the
"versioned by behaviour" property that makes `canonicaliseSeed` safe to reason
about would now cover a table that changes.

**Express rules as functions in a registry.** Most flexible, smallest type.
Rejected because a `func(string) string` can express *anything*, including a
provider's padding habit, and the boundary test that matters most would then be
enforced only by review. The closed `Fold` set exists to make the wrong rule
unwritable rather than merely discouraged.

**Rules in configuration rather than in code.** Attractive operationally: a new
institution would not need a release. Rejected outright — it puts a decision
that can merge two real instruments in a file with no review, no ADR and no test,
which inverts the whole enforcement ladder for the highest-risk rule in the
repository.

**A resolver that mints automatically for unresolved claims.** Closes the loop
with no human in it. Rejected: it is minting without an owner, which ADR-0022
and M8 both refused, and unlike the others it cannot be reversed once a ledger
contains its output.

**Record the ruleset version on `EntityMinted`.** Rejected in §6, for a reason
that is a measurement rather than a preference: `DerivationRef` carries only a
hash, so the durable version of this costs a proto field and a contract release
to answer a question that `BornFrom` already answers.

## Prior art

Securities master systems converge on the same split as ADR-0007 already
found — a landing record of what the source said, then an auditable
match-and-merge — and the ones that fail are the ones where the match is
performed rather than recorded. The specific lesson borrowed here is narrower:
the systems that survive vendor churn are the ones that normalise **only** where
an issuing standard defines the target form, and treat everything else as a
merge candidate. Vendors themselves distinguish the two: ISIN and CUSIP
normalisation is boilerplate, and symbology mapping is a paid product precisely
because it is judgement.

The failure this design is shaped to avoid is the common one of a "smart"
symbol normaliser that strips exchange suffixes: it works until two venues list
instruments that are not fungible, and by then the identifiers are in years of
records.

## Open questions

- **Is `strip all whitespace` the right and only fold for the four standard
  schemes?** Proposed, not decided. The author's position is that it is the
  largest fold that is provably safe, and that anything larger is a merge.
  Resolved by whoever accepts this RFC.
- **Should `MintIdentity` refuse, or be idempotent?** Proposed as refuse. The
  cost is that an operator scripting over `UnresolvedClaims` must handle
  `ErrAlreadyMinted` rather than ignore it. That is the intent, but it is a
  usability judgement.
- **Who is entitled to mint?** Unanswered by design. §3 records the boundary as
  the process boundary and marks it rung 6. It becomes answerable when D2 does.
- **What notices that claims are accumulating unresolved?** ADR-0022 named this
  as a real operational gap and did not close it. `UnresolvedClaims` makes it
  *askable*; nothing yet *asks*.

## Consequences

**Easier.** A claim can become an observation, which is the first time the
acquisition path works end to end. Two renderings of one ISIN mint one identity.
"Who minted this and when" is answerable from the ledger. A wrong rule is
constrained to a shape that cannot merge two valid values.

**Harder.** Every caller of `Resolve`, `Unresolved`, `MintFor` and
`DeriveHoldingObserved` now passes a ruleset, and `NewLedger` requires one.
That is deliberate: the deployed ruleset determines what merges, so it is a
composition-root decision rather than a default. Changing a shipped rule becomes
an ADR-class act with an `EntitiesIdentified` obligation attached.

**Impossible.** Minting by admission or by inspection. Writing a canonicalisation
rule for a scheme with no issuing standard. Writing a rule that alters a value
already in canonical form. Minting a second identity for a claim that already
resolves.
