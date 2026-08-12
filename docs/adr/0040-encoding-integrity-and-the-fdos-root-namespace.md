---
id: ADR-0040
title: Encodings that outlive a process are framed, and the FDOS root namespace is a constant
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0040 — Encodings that outlive a process are framed, and the FDOS root namespace is a constant

## Context

Records what [RFC-0016](../rfc/0016-encoding-integrity.md) settled.

Nine measured defects shared one root: an encoding that leaves a process was
built by string concatenation, and nothing stated that such an encoding must be
**injective** or **order-preserving**. Two of the nine have already been repaired
because they implemented decisions this repository had already accepted — the
store's sequence (ADR-0034, [#80](https://github.com/FabioCaffarello/fdos/pull/80))
and the exact money context (ADR-0008,
[#81](https://github.com/FabioCaffarello/fdos/pull/81)). This decision covers the
rest, which needed a design and a human before they could be written.

Two forces make this a decision rather than an obvious choice.

**The cost was ambiguous by an order of magnitude.** One item — expressing
decimal places — touches `fdos.kernel.v1.RoundingContext`, a published message.
Redefining `precision` would be a *meaning* change, which
[ADR-0024](0024-contract-lifecycle-and-versioning.md) says "is never a minor
bump": a new package path `fdos.kernel.v2` published alongside `v1`, N-1 held for
a milestone, and a migration issue in every consuming repository before merge.
And because a major boundary is the only vehicle the `SourceRef.value` →
`content_hash` rename can ride, taking `v2` would drag that recorded obligation
along with it.

**`buf breaking` cannot see the class most of this work belongs to.** Measured
with the pinned `buf` 1.68.4: a comment-only redefinition of `precision`'s
meaning **passes**. The identity and derivation repairs change every *value* a
byte-identical schema carries. So the gate that protects the contract reports
success on precisely the changes that invalidate stored data.

Serves Constitution §6 (provenance), §7 (temporal modeling) and §9
(reproducibility). None of those is a rung regression to admit: injectivity and
byte-order-equals-domain-order are properties no existing mechanism ever claimed,
which is why nothing caught their absence.

## Decision

**FDOS adopts RFC-0016.** Five points are decided here; the first two were
routed to a human by the RFC and answered by one, and the remaining three adopt
the RFC's own proposals rather than choosing between open forks.

### 1. Decimal places are expressed additively — Route B

`precision` keeps the meaning it has: **significant digits**, governing
intermediates. A **scale** concept is added beside it, governing the result:
`optional sint32 scale = 3` on `fdos.kernel.v1.RoundingContext`, signed because
rounding to tens is as legitimate as rounding to cents, and with explicit
presence so "no scale constraint" stays distinguishable from "scale 0". The
money kernel gains `Quantize` as the operation that applies it.

**This is a minor bump. `fdos.kernel.v2` is not opened, and the `content_hash`
rename stays parked** where `docs/ecosystem/roadmap.md` put it.

Route B is not the cheaper half of a trade. Significant digits and decimal
places are two different concepts, and money's rounding target is the second:
ISO 4217 publishes minor units per currency — 0 for JPY, 3 for KWD, 4 for CLF,
undefined for metals — and Council Regulation (EC) No 1103/97 Article 5
*requires* rounding to the sub-unit. A fixed significant-digit budget cannot
express any of them. Route A would have bought a major version in order to
delete the concept that is legally mandated and keep the one that cannot say it.

It also answers an item ADR-0008 left open in as many words — "per-currency scale
constraints at construction" — and names the example RFC-0002 named: *"JPY 0
decimals, USD 2"*. The tension RFC-0002 recorded between a per-currency
constraint and intermediates needing more precision is exactly the split above:
precision for intermediates, scale for results.

**Two obligations travel with it.** `Quantize` is total on scale and *partial on
precision* — the decimal specification raises Invalid Operation when the
quantized coefficient would exceed precision — so precision is sized from the
largest representable amount times the currency's minor units, and an Invalid
Operation surfaces as a **domain error**, never as a NaN a caller can propagate.
And this does not reopen ADR-0008's rejection of integer minor units: that
rejected scale as the *representation*, because it would be implicit. An explicit
scale constraint on an arbitrary-precision decimal is the opposite move.

### 2. The FDOS root namespace is `2b0f57e7-1fb1-4b00-811c-8dd92cc8170b`

A UUIDv4, generated from a CSPRNG on 2026-08-12, recorded here as a constant and
never to be regenerated.

```
2b0f57e7-1fb1-4b00-811c-8dd92cc8170b
```

`libs/kernel/identity` currently uses `6ba7b810-9dad-11d1-80b4-00c04fd430c8` —
the **registered DNS namespace** — behind a comment claiming it is "the FDOS
root, itself a UUIDv5 over a fixed string". Both halves were wrong. The value is
a namespace every DNS-named UUIDv5 in existence shares, and the method the
comment describes is one RFC 9562 §6.6 forbids for a custom namespace:
*"These custom Namespace ID values MUST NOT use the logic above; instead,
generating a UUIDv4 or UUIDv7 Namespace ID value is RECOMMENDED."*

v4 rather than v7 deliberately: a namespace constant gains nothing from being
time-ordered, and v7 would embed the moment it was minted for no benefit.
Verified at generation — version nibble 4, variant bits `10`, 122 bits of
entropy, no collision with any registered namespace, and not the UUIDv3 or
UUIDv5 of any obvious name over any registered namespace.

### 3. One encoding profile, with no negotiable parameters

Every value that will be hashed or ordered is serialised by a single function:

1. **Domain separation by type tag** — a constant prefix naming what the value
   is, so two different kinds of thing cannot produce one byte string.
2. **Length-prefix every variable-length component**, fixed-width big-endian.
   This is what makes the encoding injective, and it is what `":"` and
   `"\nparam="` fail to do: no separator is safe when the payload may contain it,
   and escaping moves the problem to the escape character.
3. **One stated ordering** — where a collection has no inherent order, sort by
   the **UTF-8 byte encoding** of the key. Bytewise over UTF-16 code units
   because the determinism specifications disagree on this exact point, Go
   strings are already UTF-8, and the basis that needs no transcoding removes a
   class of divergence.
4. **Minimum-length integers, no alternative renderings.**

**Scalar domains are named before the bytes**: an instant is an integer count of
nanoseconds from a stated epoch; a decimal is a coefficient-and-exponent pair,
never a float; a string is UTF-8 bytes with no normalisation applied.

**Non-canonical input is refused, never normalised.** Normalising is how two
distinct inputs become one value, which is the defect class this decision closes.
The rejection predicate is one small function per encoding, testable against a
corpus of non-canonical byte strings. This is not new policy — it is
`identity.NewClaim`'s existing reasoning generalised: it already refuses a
non-canonical scheme rather than folding it, *"because silently folding
`\"Ticker\"` into `\"ticker\"` hides that a connector is emitting something the
vocabulary does not contain."*

**Protobuf is not the encoding**, and its publisher is the authority for that:
*"protobuf serialization is not (and cannot be) canonical […] hashes of
serialized protos are fragile and not stable across time or space."* The Go, Java
and C++ APIs each instruct users needing fingerprinting to define their own
canonicalization specification. Defining one here follows that instruction rather
than inventing a burden.

### 4. A store whose encoding version is unknown is refused, not guessed

`libs/ledger-sqlite` records its format in `PRAGMA user_version`: `1` denotes the
RFC3339-text encoding, `2` the orderable integer encoding. A store at `1` is
refused with a message naming the migration.

Refusing is the correct default because a store the binary cannot order correctly
is one it must not answer an as-of query from. And the marker is not optional
decoration — measured, without it the change is unsafe in a way no error reports:
`CREATE TABLE IF NOT EXISTS` is a **no-op against an existing file**, a `STRICT`
`TEXT` column **silently coerces an integer to text**, and the resulting mixed
store orders `1782000000000000000` before `2026-07-01T00:00:00Z`.

The migration itself is a table rebuild from data already present: both the
temporal columns and the sequence are redundant projections of the encoded blob.

### 5. A value change under an unchanged schema is a recorded consequence, not a silent one

ADR-0024 has no row for "the schema is byte-identical and every value derived
under it differs", and `buf breaking` reports success on it. **This decision does
not amend ADR-0024**; it states the narrow rule and leaves the general case to
whoever needs it:

> A change that alters the values a published field carries, without altering the
> field, is recorded in the ADR that makes it and in
> `docs/ecosystem/contracts.md`. It is not a breaking *contract* change under E7,
> because the contract is unchanged — and it is not invisible either.

`identity.proto` documents `EntityId` as *"derived deterministically at first
observation, so replaying the same input yields the same identifier"*. Point 2
makes that false across the fix boundary. That is a published promise being
broken while no schema changes, and it is exactly why the class needs naming.

## Consequences

### Positive

- Two different claims can no longer become one identity by accident.
  `claim("ticker", "x:y")` and `claim("ticker:x", "y")` currently derive **one**
  `EntityId`; the framing in point 3 makes that unrepresentable.
- An as-of read means what it says in both stores, at sub-second precision.
- "Round to the cent" becomes expressible, and the money kernel gets its first
  reason to have a caller.
- A store that cannot be ordered correctly says so at `Open` instead of
  answering wrongly.
- The cheapest possible contract outcome: a minor bump, no consumer migration,
  and an obligation that stays parked instead of being dragged forward.

### Negative

- **The identity repairs need `EntitiesIdentified` to become real first.** It
  exists in the contract and has **no Go domain payload type, no codec case and
  no projection traversal** — so the merge decision a migration forces on an
  operator cannot currently be recorded as a fact at all. Three known gaps become
  blockers rather than improvements.
- **Derivation addresses cannot be migrated, only re-derived.** No
  `DerivationRecord` is persisted anywhere, so an old address has no recoverable
  pre-image. This is safe today because none exists, and it is not a property
  that survives one adopter — which is the whole sequencing argument.
- **`RoundingContext` now carries two concepts**, and a caller must know which it
  means, including that the two can conflict into a domain error. That is the
  cost of the domain genuinely having two, and the alternative is worse.
- **Three superseding releases are owed**, because `libs/ledger-sqlite/v0.1.0`,
  `libs/ledger/v0.5.0` and `libs/contracts/v0.5.0` were published *with these
  defects inside*. "P0 before any release" can only bind future tags.
- **The registry must be corrected as part of the release**, and it is currently
  wrong: `docs/ecosystem/contracts.md` lists the `fdos.*.v1` packages at `v0.3.0`
  against a pinned `v0.5.0`, `libs/kernel` at `v0.5.0` against a pinned `v0.7.0`,
  and omits `libs/ledger-sqlite` entirely. ADR-0024 calls the registry "part of
  the interface", not documentation about it.
- **The gate cannot evaluate a change of this shape.** `FOR_EACH_MODULE` runs
  `GOWORK=off`, so while `libs/kernel` is edited, `make verify` compiles the five
  modules that depend on it against **kernel v0.7.0 from the module cache**. The
  blast radius surfaces one tag at a time, which is why the release order below
  is part of this decision rather than an implementation detail.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A seed or pre-image encoding is injective | **3 — CI** | Property test over the separator characters the current code is defeated by (`:`, `\n`, `=`) |
| Byte order equals chronological order | **3 — CI** | Property test over instants differing only in sub-second precision — the region `storetest` cannot currently reach |
| The two stores agree | **3 — CI** | `storetest` gains fractional-second knowledge times. It builds every one as `epoch.Add(N * time.Hour)` today: fixed-width and fraction-free, the one region where lexicographic and chronological order coincide, which is why both stores pass it while disagreeing |
| Non-canonical input is refused, not normalised | **3 — CI** | A corpus of non-canonical byte strings, each asserted rejected by one small predicate per encoding |
| A store's encoding version is known | **1 — type** | `Open` reads `user_version` and errors; an unreadable store cannot be queried because there is no value to query it with |
| The namespace constant is never regenerated | **6 — discipline** | Nothing can detect a future edit to a random constant. Stated as rung 6 rather than implied to be higher — the mitigation is that it is recorded here, and ADRs are immutable |
| Field numbers are never reused | **6 — discipline** | A `reserved` policy is a convention `buf` cannot check for fields never declared. It must reserve **names alongside numbers**: reserving an in-use number produces *two* `buf` findings, the deletion and the un-reserved name |

**A rung claim this decision deliberately does not make.** Constitution §15 row 2
cites the `nondet` and `nofloat` analysers at rung 2. Measured against the real
`fdoslint`, a package doing float arithmetic through a named type, two indirect
clock reads and order-dependent map iteration produces **zero diagnostics**. That
is not this decision's subject and must not be repaired quietly inside it.

**Release order**, because the gate cannot derive it: `libs/kernel` →
{`libs/kernel-wire`, `libs/ledger`} → {`libs/ledger-wire`, `libs/ledger-sqlite`}
→ `apps/submitd`, with `libs/contracts` cut before anything that consumes the new
field. A root-level `go build ./...` — which does use the workspace — is the
cheap pre-flight no current target runs.

## Alternatives considered

**Route A: redefine `precision` to mean decimal places.** The strongest
alternative, and it had one real argument — a major boundary migrates every
consumer by construction, so doing several breaking things at once is cheaper
than doing them separately, and the `content_hash` rename is already assigned
there. It lost on the premise: Route B is not a smaller Route A, it is the
correct model. Buying a major version to delete one of two needed concepts is
paying for a regression.

**Derive the namespace by hashing `"fdos"` or the module path.** Rejected by
RFC 9562 §6.6 in as many words, and it is what the current comment claims was
done. A derived namespace is guessable and shares a preimage space with every
other implementer who had the same idea.

**UUIDv7 for the namespace.** Rejected: time-ordering is worthless for a
constant, and it would embed the minting moment for no benefit.

**Encode the pre-image as protobuf and hash that.** The obvious move in a
repository whose contract surface is protobuf, and rejected on the publisher's
own anti-guarantee. It would replace a *visible* collision with an *invisible*
instability — addresses that drift with a library upgrade — which is strictly
worse, because the current defect at least reproduces.

**One `libs/encoding` module owning every canonical form.** Rejected: it would
put identity seeds, derivation pre-images and storage columns behind one import,
and ADR-0013's layer topology exists to prevent that. The three encodings share a
*property*, not a dependency. A shared property is a shared test, which is what
the enforcement table proposes.

**Migrate a pre-fix store silently rather than refusing it.** Rejected on the
measurement in point 4: there is no way to distinguish the two encodings inside a
file without a marker, so a silent migration is a guess about data the ledger
exists to be certain about.

## Notes

**Item 9 travels with this decision.** The `reserved` policy is additive,
adoptable inside `v1` at measured zero cost, and depends on none of the design
above. RFC-0016's fifth open question asked whether it should split out; it does
not need to.

**What this decision does not settle**, carried forward from the RFC:

- **Per-scheme canonicalisation is untouched.** ADR-0033 decided it, implemented
  it, and lists a scheme-aware `canonicaliseSeed` as a *rejected* alternative.
  ADR-0033's safety argument is sound; what defeated it was the seed
  concatenation underneath, which point 3 fixes without reopening it.
- **Upcasters remain absent.** ADR-0011 mandates upcast-on-read and ADR-0034
  records that this became load-bearing for storage. Zero occurrences of `upcast`
  exist in any `.go` file. That is a payload-evolution decision, not an encoding
  one.
- **The `Fold` seed** (RFC-0016 item 4) is the one remaining item whose mechanism
  is genuinely open. ADR-0012 requires the trace to record inputs and its stated
  rationale is that "a trace cannot be dropped by manual threading" — which the
  seed currently is. The only caller, `domain.ProjectPosition`, compensates by
  hand. `explained.Fold` is the only guaranteed compile break in the set, so no
  fix hides behind its signature.
- **Stream-name validation and the idempotency natural key** are admission
  questions, not encoding ones. The natural key as sketched composes
  producer-supplied `content_hash` and `collected_at`, so it needs its own threat
  model against ADR-0037 §2, and it is unusable by a retrying producer until the
  missing submission *response* contract exists.
- **The `Any` payload.** `fact.proto:60` is `google.protobuf.Any`, so `buf
  breaking` is structurally blind to all payload compatibility. A `reserved`
  policy does not reach it. `Fact.type_version` does give payloads their own
  major axis, which makes future payload-level encoding fixes cheaper than
  kernel-level ones.
