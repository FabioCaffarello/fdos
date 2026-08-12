---
id: RFC-0016
title: Every encoding that outlives a process is injective, ordered and versioned
status: Draft
date: 2026-08-11
authors:
  - "@FabioCaffarello"
---

# RFC-0016 — Every encoding that outlives a process is injective, ordered and versioned

## Summary

Nine measured defects share one root: an encoding that leaves a process — a
UUIDv5 seed, a content-address pre-image, a timestamp column, a sequence number
— was built by string concatenation or by a convenient in-memory comparison,
and nothing anywhere states that such an encoding must be **injective** (two
different inputs never produce one output) or **order-preserving** (byte order
matches the order the domain means).

This needs an RFC rather than an ADR for one reason: the fixes are individually
obvious and their *sequencing against [ADR-0024](../adr/0024-contract-lifecycle-and-versioning.md)
is not*. Two of them change values that a published wire format already carries
without changing the schema, which is a class ADR-0024 does not name and
`buf breaking` cannot see. One of them looks like it forces `fdos.kernel.v2`.
Getting that judgement wrong costs either an unnecessary major version — which
drags a recorded `content_hash` obligation along with it — or a silent
meaning-change to a field an external consumer already pins.

**The proposal's central finding: the set does not force `fdos.kernel.v2`, and
the one item that would has an additive route that is also the better design.**

## Motivation

### What breaks

Re-verified by execution at `bf371b7` on 2026-08-10; nine of these were
reproduced by running code, not by reading it.

| # | Defect | Consequence |
|---|---|---|
| 1 | `libs/kernel/identity/claim.go:85` joins `scheme + ":" + value`; `identity.go:106` joins `kind + ":" + canonical`. `NewClaim` permits `:` in both fields | `claim("ticker","x:y")` and `claim("ticker:x","y")` derive **one** `EntityId`. Two different claims mint one identity — the silent merge ADR-0007 exists to forbid |
| 2 | `identity.go:86` — the namespace constant is `6ba7b810-9dad-11d1-80b4-00c04fd430c8`, the RFC 4122/9562 **DNS** namespace, commented as "the FDOS root, itself a UUIDv5 over a fixed string" | Two defects, not one. The *value* is a registered namespace, so every FDOS identity shares a namespace with every DNS-named UUIDv5 in existence. And the *method the comment describes* is one RFC 9562 §6.6 forbids: custom namespaces "MUST NOT use the logic above; instead, generating a UUIDv4 or UUIDv7 Namespace ID value is RECOMMENDED" |
| 3 | `provenance.go:500-523` builds the derivation pre-image with unescaped `"\ninput="` / `"\nparam="` separators | A parameter value containing `\nparam=` forges another parameter. Measured: two structurally different derivations share address `881ab834…`, and the inputs/parameters boundary is crossable |
| 4 | `explained.go:209` consumes a `Fold` seed that `:222` never records | Two folds with different seeds and different answers produce one identical trace |
| 5 | `libs/ledger-sqlite/store.go:200` compares knowledge times as RFC3339 strings; `temporal.go:78` formats with `RFC3339Nano`, which omits trailing zeros | `.` (0x2E) sorts below `Z` (0x5A), so any sub-second instant compares as *earlier* than the whole second it follows. The memory store (`memory/store.go:85`) uses `!incoming.After(last)` and disagrees |
| 6 | `store.go:188` derives the next sequence from `COUNT(*)`; `store.go:138` discards the stored `sequence` on load | After one deleted row: refs silently re-point to different content, **and** the stream becomes permanently unappendable (`UNIQUE constraint failed`). Both measured |
| 7 | `money.go:246-250` traps neither `apd.Inexact` nor `apd.Rounded`, and `BaseContext` rounds half-up | The context named `exact` silently discards a unit at 96 significant digits. Measured: `10^96 + 1 == 10^96`, error `nil`. **This contradicts ADR-0008's decision text twice over** — see below |
| 8 | `rounding.go:103` — `precision` is significant digits, and there is no `Quantize` anywhere in `libs/kernel/money` | "Round to the cent" is inexpressible. One context yields three different decimal places for `1/3`, `1000/3`, `0.001/3` |
| 9 | Zero `reserved` field-number declarations exist anywhere in `libs/contracts/proto/` | A field deletion plus number reuse is undetectable by `buf breaking` |

**Defect 7 is not a design improvement; it is an accepted decision not being
honoured.** ADR-0008's Decision says, in two separate sentences:

> "**There is no default rounding context** and no privileged mode — the correct
> choice is jurisdiction- and instrument-specific, and a default is exactly how
> that decision goes unexamined. **Precision loss is signalled and recorded in the
> computation trace.**"

`exactContext()` violates both. It inherits `apd.BaseContext`'s rounding, which
is half-up — a privileged default mode, chosen by omission, in the one context
whose name promises no rounding at all. And because neither `apd.Inexact` nor
`apd.Rounded` is trapped, the loss is **neither signalled nor recorded**. Worse,
`Add`, `Sub` and `Mul` all route through this context (`money.go:164`, `:176`,
`:207`; `quantity.go:70`, `:82`) — the three operations ADR-0008 and
`money.proto:51-52` both describe as *exact*.

So this item is a correctness repair against an accepted ADR, not a proposal, and
it should not be sequenced behind the design questions in the rest of this RFC.

Constitution principles at stake, read against the honest §15 table:

- **§7 Temporal Modeling** sits at rung 1 for *requiring both axes*. Nothing
  claims the axes *compare* correctly, and defect 5 is that gap.
- **§6 Provenance** sits at rung 1 for `NewEnvelope` refusing incompleteness.
  Nothing claims a derivation address *identifies* one derivation, and defect 3
  is that gap.
- **§4 Immutable Ledger** sits at rung 1 because there is no whole-stream write
  to refuse. Defect 6 mutates history through the read path instead, which the
  row does not cover.

None of these is a rung regression to admit. They are properties no current
mechanism ever claimed — which is exactly why they were never caught.

### Is this retrofittable?

**Partly, and the boundary matters more than the fixes.**

- Defects 4, 7, 8, 9 are retrofittable at any time. They change what future
  computations produce and nothing that already exists.
- Defects 1 and 2 change **derived identifiers**, and — measured — break less
  than they appear to: resolution recomputes both sides from persisted claims, so
  no stored value becomes unreadable. What they create is a *silent divergence*
  between facts naming a previously-collided identifier and newly-minted separate
  ones, which only a recorded `EntitiesIdentified` judgement can settle.
- Defects 3 and 4 change **derivation addresses**, and these are the genuinely
  unmigratable ones: no derivation record is persisted anywhere, so an old
  address has no recoverable pre-image and cannot be translated. **Safe today
  only because nothing has been persisted; not a property that survives one
  adopter.**
- Defects 5 and 6 change **persisted storage** and are the easy half — both
  columns are redundant projections of the encoded blob, so a rebuild recovers
  everything. But §Migration shows that without a version marker a store written
  before the fix keeps producing wrong answers *after* it, silently, forever.

So the retrofittable claim in the audit's Phase-0 framing ("cheap now, permanent
later") is sound in direction and needs one correction: it is not the *fix* that
becomes impossible, it is the *silent* fix. After real data exists, every one of
these remains fixable — as a migration with a recorded lineage, which costs more
and is visible to a consumer.

### One assumption, checked rather than repeated

Every "cheap now" argument rests on **"no production data exists"**, which is a
claim about the world, not about the tree. What is checkable:

- No `.db` or `.sqlite` file is tracked or present in either repository's tree.
- **`apps/submitd` has no tag at all**, and `-store` has an empty default with no
  in-memory fallback (`main.go:47`, `:53-55`) — so a store exists only where an
  operator explicitly pointed one, at an arbitrary path with nothing conventional
  to sweep.
- **`libs/ledger-sqlite/v0.1.0` is a published tag.** That is the real exposure:
  anyone can `go get` the durable store and write facts directly, without
  touching `submitd`.

The assumption is therefore **better and worse** than "no data exists".

*Better:* `submitd`'s handler decodes one message type and calls
`AcceptHoldingClaim` (`server.go:99-115`), which appends `domain.HoldingClaimed`
— `identity.Claim` values, verbatim, **with no derived identifiers at all**
(`usecases.go:300-305`). A store written only by `submitd` contains nothing that
needs recovering, which makes the identity fixes fully migratable against it.
`ObserveHolding`, the path that persists caller-supplied identifiers, is
unreachable from any binary in `apps/`.

*Worse:* a store written through the published library can contain exactly the
unrecoverable cases, and **with no version marker there is no way to interrogate
a file to find out what encoding it holds.** So the honest position is not "no
data exists" but *"no data this repository can see, and no means to ask."* The
proposal is designed not to need the assumption: every step is safe whether or
not an adopter's store exists, which costs one PRAGMA and buys the difference
between a migration and a corruption.

## Design

### The four classes, because ADR-0024 names only three

ADR-0024 governs the *contract*: a new message or field is a minor bump, and
"nothing that changes the meaning of an existing field is ever a minor bump."
It does not name the class this change set is mostly made of.

| Class | Definition | ADR-0024 route | Detected by |
|---|---|---|---|
| **Internal** | No published artifact changes | none | — |
| **Additive** | New message, field or operation | minor | `buf breaking` passes ✅ |
| **Meaning-changing** | An existing published field means something else | **major** — new package path, N-1 for a milestone, consumer issue before merge | **nothing** — measured below |
| **Value-changing** | Schema byte-identical; every value derived under it differs | **unnamed** | nothing |
| **Domain-narrowing** | Schema and meaning unchanged; the *accepted input set* shrinks | **unnamed** | nothing |

The last three are the classes this change set is mostly made of, and naming
them is half of what this RFC is for.

**Measured: `buf breaking` cannot see the one act ADR-0024 forbids.** Verified
with the pinned `buf` 1.68.4 against isolated copies of the proto tree, with an
identical-copy control passing:

| Candidate edit to `RoundingContext` | `buf breaking` |
|---|---|
| Redefine `precision`'s meaning in comments only | **PASS** |
| Add `sint32 scale = 3` | PASS |
| `precision` `uint32` → `int32` | BREAKING (type change) |
| Rename `precision` → `significant_digits` | BREAKING (name, and `json_name` under the JSON ruleset) |
| `reserved 1` replacing `precision` | BREAKING — **two** findings: the deletion *and* the name not being reserved |
| `SourceRef.value` → `content_hash` | BREAKING — confirms the obligation already recorded at `provenance.proto:52-59` |

So ADR-0024's central sentence — *"Nothing that changes the meaning of an
existing field is ever a minor bump"* — has **no enforcement mechanism
whatsoever**. It is a rung-6 rule guarding the most expensive mistake in the
document, and row 1 above is the proof.

`buf breaking` compares *schemas*. An `EntityId` is the same field before and
after defect 1 is fixed; the schema is identical and every value in it changes. A
gate that reports success on the change that invalidates stored data is worse
than no gate.

**And one published *guarantee* is broken even though no meaning is.**
`identity.proto:8-13` documents `EntityId` as "derived deterministically at first
observation, so replaying the same input yields the same identifier". Fixes 1–3
make that false across the fix boundary. Under ADR-0024's wording the field's
meaning is unchanged, so this stays a value-change — which is precisely why the
class needs a governance home rather than a case-by-case judgement.

### Classification of the change set

| # | Fix | Published artifact | Class | Note |
|---|---|---|---|---|
| 1 | Injective identity seed | `fdos.kernel.v1.EntityId` (shape unchanged) | **Value-changing** | Fix is a length-prefixed or escaped join; no schema edit |
| 2 | A real FDOS namespace constant | same | **Value-changing** | Same UUIDv5 shape, different namespace argument |
| 3 | Injective derivation pre-image | `fdos.kernel.v1.DerivationRef.content_hash` (shape unchanged) | **Value-changing** | Fix is length-prefixed framing of the pre-image |
| 4 | Record the `Fold` seed | none | **Internal** | Additive to `provenance.NewDerivation`'s inputs at the call site |
| 5 | Orderable temporal storage | none — `libs/ledger-sqlite` publishes no proto | **Internal + storage** | The wire is already correct: `temporal.proto:14-32` carries `google.protobuf.Timestamp`, an integer seconds-and-nanos pair that is orderable by construction. The store **re-renders** it as `RFC3339Nano` text and loses a property the contract already had. The fix restores the representation the contract chose, one layer down. See Migration |
| 6 | `MAX(sequence)+1`, preserve stored sequence on load | none | **Internal** | Implements ADR-0034 correctly; see below |
| 7 | Trap `Inexact`/`Rounded` in the exact context | none | **Internal** | Behaviour behind an unchanged signature |
| 8 | Express decimal places | **`fdos.kernel.v1.RoundingContext`** (`money.proto:59-62`) | **Additive *or* meaning-changing — this is the decision** | See below |
| 9 | A `reserved` policy | all of `libs/contracts/proto` | **Additive** | Adoptable inside `v1` at zero cost: measured, `reserved` on unused numbers passes `buf`. It must reserve **names alongside numbers** — reserving an in-use number produces *two* `buf` findings, the deletion and the un-reserved name |

**Item 6 is not a new decision.** ADR-0034 already decided that the store
assigns the sequence, and records it in its own enforcement table as *rung 1 —
"the sequence is assigned by the store, not by a pure value."* `COUNT(*)`
assigns it by *position*, and the load path re-derives refs through
`domain.Stream.Append`. Fixing this implements an accepted decision that the
implementation does not currently honour. It still needs an ADR — the `app.Store`
port's documented semantics change — but it needs no design exploration and
should not wait for the rest of this RFC.

**It does need one thing the audit's version of the fix omits: a reader rule.**
Prior art is emphatic that a writer rule alone is insufficient — every mature
sequence is monotonic but **not contiguous**. PostgreSQL states it outright:
"sequence objects cannot be used if 'gapless' assignment of sequence numbers is
needed." Kafka guarantees offsets are permanent and ordered while transaction
markers leave visible holes. EventStoreDB's truncation makes a stream's first
visible revision non-zero. And in a global sequence a hole is *ambiguous*
between "rolled back forever" and "committing right now", which is why Marten
needs a high-water mark and why reading past a hole is silent data loss.

FDOS is in the easy case and should exploit it. A single-writer, append-only
per-stream sequence assigned by the store has **no legitimate source of gaps**:
`Append` never rolls back a reserved number, because it does not reserve one.
Therefore a gap is not a condition to tolerate — it is **evidence of corruption
or of out-of-band deletion**, and the correct reader rule is to *refuse the
stream and say so*, not to renumber it into consistency. Renumbering is what the
code does today, and it is why one deleted row silently re-points every
subsequent ref.

This is a stricter rule than "preserve the stored sequence", and it is the one
the append-only guarantee actually licenses.

### The encoding rule itself

Prior art is unanimous that "canonical" must name a profile rather than describe
an intention, so the proposal is specific. **One profile, no parameters.**

Every value that will be hashed or ordered is serialised by a single function
obeying four rules:

1. **Domain separation by type tag.** Each structured value is prefixed by a
   constant tag naming what it is (`entity-seed`, `derivation`, `parameter`).
   Git's `"blob 16\0"` is the model: two distinct kinds of thing can never
   produce one byte string.
2. **Length-prefix every variable-length component**, as a fixed-width
   big-endian count. This is what makes the encoding injective, and it is what
   `":"` and `"\nparam="` fail to do — no separator is safe when the payload may
   contain it, and escaping merely moves the problem to the escape character.
3. **One ordering, stated.** Where a collection has no inherent order
   (derivation parameters), sort by the **UTF-8 byte encoding** of the key.
   Bytewise is chosen over UTF-16 code units deliberately: the specs disagree
   (see Prior art), Go strings are already UTF-8, and picking the basis that
   needs no transcoding removes a whole class of divergence.
4. **Minimum-length integers, no alternative renderings.** DER clause 10's
   discipline, enumerated rather than assumed.

**And the decode side is not optional.** BIP 66 and DAG-CBOR together show that
encoder rules without decoder enforcement decay into per-implementation
dialects. So:

- Where a non-canonical rendering can arrive from outside — a submission's
  `stream` name, a claim's scheme — it is **rejected**, never normalised into
  canonical form. Normalising is how two distinct inputs become one value, which
  is the defect class this RFC exists to close.

  **This is not new policy; it is `claim.go`'s existing reasoning generalised.**
  `NewClaim` already refuses a non-canonical scheme rather than folding it,
  and says why at `claim.go:28-31`: *"A non-canonical scheme is rejected rather
  than normalised, because silently folding `\"Ticker\"` into `\"ticker\"` hides
  that a connector is emitting something the vocabulary does not contain."* The
  proposal is that every encoding boundary inherit that stance.

  It also locates defect 1 precisely. `NewClaim` checks the scheme's **shape**
  (lowercase, unpadded) and not its **membership** in the vocabulary, which is
  deliberately open. So `"ticker:x"` is a well-formed scheme by every rule the
  constructor enforces, and `"x:y"` is a well-formed value because values are
  verbatim by decision. The collision is reachable through the public
  constructor without violating anything — which is why the fix belongs in the
  encoding and not in a stricter claim.
- The rejection predicate is a single small function per encoding, testable in
  isolation against a corpus of non-canonical byte strings, in the shape of
  BIP 66's `IsValidSignatureEncoding`.

**Scalar domains are named before the bytes**, per RFC 8785's own limit: an
instant is an integer count of nanoseconds from a stated epoch; a decimal is a
coefficient-and-exponent pair, never a float; a string is UTF-8 bytes with no
normalisation applied. This is why item 8 sits inside this RFC rather than beside
it — a rounding context cannot be encoded into a trace before the domain has
words for both of its concepts.

### The one item that decides whether this is `fdos.kernel.v2`

`money.proto:59-62` publishes:

```protobuf
message RoundingContext {
  uint32 precision = 1;
  RoundingMode mode = 2;
}
```

`precision` carries **no field comment**, and the eight-line message comment
above it (`money.proto:51-58`) never says significant digits or decimal places
either. Nor does **ADR-0008**, which created the type: it specifies the shape as
`{precision, mode}` (`:46`) and says nothing about the unit. Only Go commits —
`rounding.go:103` documents significant digits and `rounding.go:124` passes it to
`apd.WithPrecision`.

So the *published* meaning is **absent, not wrong**, in the contract *and* in the
decision record. Adding a scale field contradicts nothing any consumer or any ADR
was ever told; redefining `precision` changes something only Go and its callers
ever knew, while still paying the full price of a major version.

**And the additive route answers a question ADR-0008 deliberately left open.**
That ADR's Notes read: *"Open, deliberately: **per-currency scale constraints at
construction**; whether a distinct `Rate` type earns its keep; maximum supported
precision…"* — and RFC-0002's open questions name the exact example this RFC's
prior art rediscovered independently: *"Should `Money` carry a scale constraint
per currency (**JPY 0 decimals, USD 2**), enforced at construction? It catches
real errors but conflicts with intermediate values that legitimately need more
precision."*

So Route B is not a new concept bolted on. It is the recorded open item, and the
tension RFC-0002 anticipated is exactly the precision-versus-scale interaction
the decimal specification resolves: **context precision governs intermediates,
scale governs the result.** Both are needed, which is why RFC-0002 could not
choose between them.

**One thing Route B must not be mistaken for.** ADR-0008 *rejected* integer minor
units as the **representation** — "the scale is implicit and therefore easy to get
wrong across boundaries", plus overflow on high-notional low-denomination amounts
(`:85-86`, RFC-0002 `:142-145`). Adding an explicit scale *constraint* to an
arbitrary-precision decimal is the opposite move: it makes scale explicit rather
than implicit, and changes no representation. The accepting ADR should say so, so
that nothing here reads as reopening a rejected alternative.

Two routes, and they cost an order of magnitude apart:

**Route A — redefine `precision` to mean decimal places.** This is
*meaning-changing*: ADR-0024's seven-step process, `fdos.kernel.v2` published
alongside `v1`, N-1 held for a full milestone, and an issue in
`fdos-connectors` before merge. And because a major boundary is the only vehicle
the `SourceRef.value` → `content_hash` rename can ride —
`docs/ecosystem/roadmap.md` records it as *"a blocking obligation on whatever
eventually warrants `fdos.kernel.v2`"* — Route A drags that rename along with
it. One line of desired semantics buys a major version and an unrelated
obligation.

**Route B — add a scale concept beside `precision`, and leave `precision`
alone.** *Additive*: a minor bump, `buf breaking` passes, no consumer
migration, no N-1 window, and the `content_hash` obligation stays parked where
its own roadmap section put it.

**This RFC proposes Route B, and not as a compromise.** Significant digits and
decimal places are two genuinely different concepts, and every mature decimal
system carries both rather than choosing: IEEE 754-2008's decimal arithmetic has
precision *and* `quantize`; Java's `BigDecimal` has `precision()` *and*
`scale()`; Python's `decimal` has context precision *and* `quantize()`. Route A
would not correct a mistake — it would delete one of the two concepts the domain
needs. The measured defect is an **absence**, and the fix for an absence is
additive by nature.

There is a stronger argument than symmetry, and it is not aesthetic:

- **Money's rounding target is an absolute scale that is published per
  currency.** ISO 4217's official list carries `CcyMnrUnts` per entry. In the
  current list the distribution is 2 → 224 entries, **0 → 31** (JPY), **3 → 7**
  (KWD), **4 → 2** (CLF), and **N.A. → 13** (XAU and other metals). A fixed
  significant-digit budget cannot express any of them, and a hardcoded `2` is
  wrong for 51 of 277 entries.
- **In at least one jurisdiction it is a legal requirement.** Council Regulation
  (EC) No 1103/97 Article 5: monetary amounts "shall be rounded up or down to
  the nearest cent", or "to the nearest sub-unit or in the absence of a sub-unit
  to the nearest unit", and "if the application of the conversion rate gives a
  result which is exactly half-way, the sum shall be rounded up."

So rounding money to significant digits is not merely awkward, it is **wrong by
construction** against the standard that defines currencies and against statute.
Route A would take a major contract version in order to delete the concept that
is legally mandated and keep the one that cannot express it.

**One design obligation Route B carries, from the same prior art.** `quantize`
is total on scale and *partial on precision*: the decimal arithmetic
specification raises Invalid Operation when the quantized coefficient would
exceed the context's precision — `quantize('+35236450.6', '1e-2')` is `NaN` at
precision 9. So adding a scale concept does not retire `precision`; the two
interact, and the accepting ADR must state that precision is sized from the
largest representable amount times the currency's minor units, and that an
Invalid Operation surfaces as a **domain error**, never as a NaN a caller can
propagate. This is the same trap as defect 7 one level up: an arithmetic
condition that is not trapped becomes a wrong number instead of a refusal.

**Therefore: the P0 set does not force `fdos.kernel.v2`, and no
`fdos-connectors` migration issue is owed under ADR-0024 step 2.** Two
independent reasons, both worth stating because either alone would be easy to
lose:

1. **Scope, and the scoping word is in Tier 0.** ADR-0024 introduces its
   seven-step process as an application of E7, and E7 reads: *"Every breaking
   **contract** change carries an RFC, a deprecation window, an N-1
   compatibility period, and a tracked migration issue in every consuming
   repository"* (`invariants.md:48-51`). ADR-0025 then settles what is and is not
   a contract: `libs/contracts` "is the only module FDOS offers to code outside
   this repository", while `libs/kernel`, `libs/ledger`, `libs/kernel-wire` and
   `libs/ledger-wire` "carry **no compatibility promise across versions**". It
   even names these change classes — "a rename, a constructor signature, an
   unexported field — none of which changes any contract" — and draws the
   consequence: "FDOS can refactor the kernel and ledger freely." A Go-API change
   in the kernel is outside the process **by decision, not by omission**.
2. **Fact.** `fdos-connectors` imports `libs/contracts` and nothing else — no
   `libs/kernel`, no `libs/ledger`, no `-wire` module — so every Go-API change
   in this set is invisible to it at compile time.

ADR-0025 also records the limit of reason 2 honestly, and it is the same limit as
the store data: *"imports `libs/kernel` tomorrow, nothing in this repository will
report it."* The claim is true and unverifiable, which is an argument for
sequencing now rather than for relaxing anything.

### Migration, and the finding that makes it necessary

`libs/ledger-sqlite/store.go:42-53` declares the schema with
`CREATE TABLE IF NOT EXISTS … STRICT` and applies it on every `Open`. There is
**no `PRAGMA user_version`, no migration table, no format marker of any kind**.

Measured consequences of changing the temporal columns without one:

1. `CREATE TABLE IF NOT EXISTS` is a **no-op against an existing file**. Its
   columns stay `TEXT` forever; the schema constant in the binary and the schema
   in the file silently disagree.
2. `STRICT` does **not** save this. A `STRICT` `TEXT` column accepts an integer
   and converts it losslessly to text — verified: inserting
   `1782000000000000000` into a `STRICT TEXT` column stores
   `typeof() = 'text'` with no error.
3. The resulting order is wrong in a new way. Verified:
   `ORDER BY` returns `1782000000000000000` **before** `2026-07-01T00:00:00Z`,
   because `'1' < '2'` lexicographically.

So a store written before the fix keeps ordering facts wrongly *after* the fix,
silently, with no way to tell which encoding a given row uses. **The encoding
change is not safe without a durable version marker, and is cheap with one.**
`PRAGMA user_version` exists, defaults to `0`, and costs nothing.

Proposed: `user_version = 1` denotes the RFC3339-text encoding, `2` the
orderable encoding. A store at `1` is either migrated by table rebuild or
refused with a message naming the migration. **Refusing to open is the correct
default** — a store the binary cannot order correctly is one it must not answer
an as-of query from.

#### Storage migrates cleanly, because the columns are derived state

Fixes 5 and 6 are the easy half, and measurably so. Both the temporal columns and
the `sequence` column are **redundant projections of the encoded blob**:

- The true sequence is stored twice — in the `sequence` column and inside the
  blob's own `FactRef` (`ledger-wire/codec.go:57-58`, `ledger/v1/fact.proto:26`)
  — and they agree. Verified: after deleting an interior row, the surviving row's
  column reads `3` and the ref inside its blob reads `h#3`. `Load` does not need
  new information; it needs to **stop discarding** what it already has
  (`stream.go:52` recomputes `len+1`).
- The temporal columns are likewise recomputable from the blob, verified equal.

So once `user_version` exists to say which encoding a file holds, fix 5 is a
table rebuild from data already present. No information is lost and none has to
be invented.

#### Identities: nothing becomes unreadable, and the damage is a different shape

This is the part the audit's framing got wrong, and it is worth stating
precisely.

**Resolution does not use the stored identifier.** `resolve.go:50-52` compares
`CanonicalSeed(minted.BornFrom) == CanonicalSeed(claim)` — it recomputes *both*
sides from persisted claims and never re-derives or inspects `minted.Entity`.
`identity.Restore` validates shape only. So after fixes 1 and 2, **an existing
mint still answers its own claim**, and no stored value becomes unreadable or
wrong.

What actually changes is narrower and sharper. The `":"` join is present in
`CanonicalSeed` too, so two claims that previously collided *also* previously
co-resolved. After the fix they do not: the minted one keeps its stored
identifier and resolves itself, the other resolves to nothing, `ErrAlreadyMinted`
stops firing, and a **second identity is minted while every already-persisted
fact still names the collided one**. That is a silent inconsistency between
stored and newly-derived values, and no mechanical migration can settle it —
deciding whether the two claims were one entity is exactly the
`EntitiesIdentified` judgement, which ADR-0007 reserves for a recorded decision
rather than an inference.

**The pre-image is durable, for minted identities only.** `EntityMinted`
persists `Entity` (value and kind) plus `BornFrom` as a verbatim
`{scheme, value}` pair (`domain/claimed.go:42-45`,
`payload/v1/identity.proto:22-29`, round-tripped at `ledger-wire/codec.go:230-234`
and `:286-295`). So every *minted* identity is recomputable. `ObserveHolding`
(`usecases.go:109-114`) persists caller-supplied `identity.ID` values with no
mint at all, and for those the pre-image is nowhere in the ledger and is
unrecoverable — currently unreachable from any binary in `apps/`, but reachable
by anyone importing the published store.

**The linkage mechanism has two missing prerequisites, not one.** ADR-0007's
`EntitiesIdentified` is the right vehicle for recording an old identity and its
successor as one entity. But:

1. It has **no Go domain payload type and no codec case**. It exists in the
   contract (`kernel/v1/identity.proto:90`) and in generated code, and nothing
   in `libs/ledger/domain` or `libs/ledger-wire` can produce or read it. The
   merge decision the fix forces on an operator **cannot currently be recorded
   as a fact at all.**
2. `ProjectPosition` does not traverse it — zero occurrences in
   `libs/ledger/domain/position.go`. Even once recordable, positions would split
   across the old and new identity.

Both are prerequisites of the identity fixes, not follow-ups. Stating them as
blockers is the difference between a migration and a data-loss event.

#### Derivation addresses: migration is impossible, not unnecessary

This is the one class with **no path back**, and it is the strongest argument for
sequencing this work now.

Only `DerivationRef { string content_hash }` is persisted
(`kernel/v1/provenance.proto:109-111`). The `DerivationRecord` message exists in
the contract (`kernel/v1/derivation.proto:24`) and **no derivation record is
persisted anywhere** — the SQLite schema has exactly one table. ADR-0034
recorded the same fact from the other direction: making a derivation's method and
parameters durable "is a `contracts` release, not a storage change."

So an old derivation address is an opaque value with no recoverable pre-image.
After fixing the pre-image encoding it neither matches a recomputation nor can be
translated into the new address. The same holds for the `Fold` seed fix, which
moves every fold trace including the `seed == 0` case (verified:
`1b8704c2…` → `218a2ea9…`).

**These two fixes are safe only while no derivation address has been persisted or
published.** That is true today and is not a property that survives a single
adopter. Unlike identities, there is no birth certificate to replay from — the
only route back would be re-deriving from source facts, which requires the
derivation store that does not exist.

One consequence worth following: correcting the rounding semantics also moves
derivation hashes, because `RoundingContext.String()` renders
`precision=N,mode=X` directly into derivation parameters. Item 8 is therefore not
independent of fix 3 — they change the same addresses and should land together.

### What this design does not cover

- **Per-scheme canonicalisation.** Already decided by ADR-0033 and implemented
  at `canonical.go:335-338`. The scheme-blind `canonicaliseSeed` and its
  uppercase-and-collapse floor are **deliberate**, and a scheme-aware
  `canonicaliseSeed` is a *rejected alternative* in that ADR (`:257`). Nothing
  here touches it. ADR-0033's safety argument — a fold that is the identity on
  every canonical value cannot merge two valid distinct values — is sound; what
  defeats it is the seed *concatenation* underneath, which is defect 1.
- **Upcasters.** ADR-0011 mandates upcast-on-read and ADR-0034 records that this
  became "load-bearing for storage". Zero occurrences of `upcast` exist in any
  `.go` file, and `codec.go:173` hard-fails any version mismatch. That is a real
  and separate hole; it is a payload-evolution decision, not an encoding one.
- **Stream-name validation** and **the idempotency natural key**. Both were on
  the audit's P0 list and neither belongs here. Validation refuses submissions
  the published admission path accepts today, which is an admission-contract
  question. The natural key is worse than out of scope: as sketched it composes
  producer-supplied `content_hash` and `collected_at`, so a producer varying
  either on retry defeats the unique index and a hostile one mints arbitrarily
  many "distinct" duplicates — against ADR-0037 §2's requirement to revalidate
  "as though the producer were hostile". It needs its own threat model and its
  own RFC, and it is unusable by a retrying producer until the missing
  submission *response* contract exists.
- **The `Any` payload.** `fact.proto:60` is `google.protobuf.Any payload = 5`
  with `Fact.type` a runtime string (`:54`), so `buf breaking` is structurally
  blind to *all* payload compatibility — the payload's identity is resolved at
  runtime, and `verify-proto.sh` scans only `fdos/ledger/v1` for the envelope
  rule, leaving payload packages exempt by construction. A `reserved` policy
  (fix 9) does not reach it. Named here so it is not assumed covered.

  One useful corollary: `Fact.type_version` (`:58`) gives payloads their **own
  major-version axis**, so a future meaning change *inside a payload* never needs
  a package bump. That does not rescue item 8 — `RoundingContext` lives in
  `fdos.kernel.v1`, not in a payload — but it means payload-level encoding fixes
  are structurally cheaper than kernel-level ones, which is worth knowing before
  the next one is scoped.

## Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A seed or pre-image encoding is injective | **3 — CI** | Property test: no two distinct structured inputs share an encoding, generated across the separator characters (`:`, `\n`, `=`) the current code is defeated by |
| Byte order equals chronological order | **3 — CI** | Property test over instants that differ only in sub-second precision — the region `storetest` currently cannot reach |
| The two stores agree | **3 — CI** | `storetest` gains fractional-second knowledge times. Today it builds every one as `epoch.Add(N * time.Hour)`, fixed-width and fraction-free, which is the single region where lexicographic and chronological order coincide — the suite is structurally blind to defect 5 and both stores pass it |
| A store's encoding version is known | **1 — type** | `Open` reads `user_version` and returns an error naming the migration; an unreadable store cannot be queried because there is no value to query it with |
| Refs survive a load, and a gap is refused | **3 — CI** | Test: delete an interior row; assert `Load` **fails naming the gap** rather than renumbering, and that an intact stream round-trips its stored sequences. An append-only single-writer sequence has no legitimate gap, so tolerating one is how corruption becomes an answer |
| The exact context is exact | **3 — CI** | Test: `10^96 + 1 != 10^96`, and the trap fires rather than rounding |
| Non-canonical input is refused, not normalised | **3 — CI** | A corpus of non-canonical byte strings, each asserted **rejected** by a single small predicate per encoding — BIP 66's `IsValidSignatureEncoding` shape. Encoder rules without this row are what DAG-CBOR's own spec admits decays into dialects |
| Field numbers are never reused | **6 — discipline** | A `reserved` policy is a convention `buf` cannot check for fields that were never declared. Stated as rung 6 rather than implied to be higher |

Two of these rows are the point. The encodings are being fixed *and* the
instruments that failed to detect them are being fixed in the same change —
because a green suite over a blind generator is the mechanism that let nine
defects reach a published tag.

**A rung claim this RFC deliberately does not make.** Constitution §15 row 2
cites the `nondet` and `nofloat` analysers at rung 2. Measured against the real
`fdoslint` binary, a package doing float arithmetic through a named type
(`type Rate float64`), two indirect clock reads and order-dependent map
iteration produces **zero diagnostics** — exit 0. The analysers catch direct
syntactic forms only. That is not this RFC's subject and must not be fixed
quietly inside it; it is noted so that nothing here is read as evidence the
determinism mechanisms are stronger than they are.

### The gate cannot see a multi-module change, which is what this one is

Measured, and it changes how the work must be **sequenced** rather than what it
is. `FOR_EACH_MODULE` (`Makefile:30-36`) runs every module with `GOWORK=off`, so
siblings resolve through **published versions from the module cache**:

```
cd libs/ledger && GOWORK=off go list -f '{{.Dir}}' .../libs/kernel/identity
→ /…/go/pkg/mod/…/fdos/libs/kernel@v0.7.0/identity
```

So while `libs/kernel` is being edited, `make verify` compiles `libs/ledger`,
`libs/kernel-wire`, `libs/ledger-wire`, `libs/ledger-sqlite` and `apps/submitd`
against **kernel v0.7.0 from the cache**. A kernel signature change is
compile-checked only inside `libs/kernel` itself, and its blast radius surfaces
one tag at a time: kernel → {kernel-wire, ledger} → {ledger-wire, ledger-sqlite}
→ submitd. `libs/ledger-sqlite` is absent from `go.work` entirely, so even a
workspace build never sees its source against local siblings.

This is deliberate — `GOWORK=off` is ADR-0004's discipline and it is what proves
each module resolves standalone. But it means **a change of exactly this shape is
the one the gate is least able to evaluate.** The accepting ADR must therefore
state a release order, and the implementation should carry a root-level
`go build ./...` — which *does* use the workspace — as a pre-flight no current
target runs.

`explained.Fold` is the only guaranteed compile break in the set: the seed is
`B any` at `explained.go:197-204` and never reaches the pre-image at `:222-224`,
so no fix for defect 4 can hide behind the current signature.

### Two live defects this RFC surfaces and does not own

Both were found while establishing the above, and both are the same class as the
blind conformance generator — an enforcement claim that is **false right now**.

1. **`examples/ingest` does not compile.** Verified:
   `./conform.go:56:50: not enough arguments in call to app.NewLedger — have
   (*memory.Store, *clock.Sequence), want (app.Store, app.Clock,
   identity.Ruleset)`. The break came from ADR-0033's added `identity.Ruleset`
   parameter, not from this work. It is invisible because
   `scripts/list-modules.sh` is `find libs apps -name go.mod`, excluding
   `examples/` by construction — so ADR-0037's enforcement row claiming *"A
   producer can check its own conformance | rung 3 | `examples/ingest` — the kit
   runs in CI and restates no rules, calling the real admission path"* is not
   true today. It is also one of only two production call sites of
   `AcceptHoldingClaim`.
2. **The contract registry is stale, and ADR-0024 calls it "part of the
   interface", not documentation about it.** `contracts.md:34-36` lists the
   `fdos.*.v1` packages at `v0.3.0` while `apps/submitd/go.mod:6` pins
   `libs/contracts v0.5.0`; `contracts.md:80` lists `libs/kernel` at `v0.5.0`
   while `submitd` pins `v0.7.0`; and `libs/ledger-sqlite` — published, imported
   and tagged — is not in the registry at all.

Neither is caused by this RFC and neither should be repaired silently inside it.
Both are named because M12a's sequencing depends on the first and its release
step depends on the second.

## Alternatives

**Fix nothing until a real adopter appears.** Rejected on the measured
asymmetry: nothing here is expensive today and every item becomes a migration
with a recorded lineage once facts persist. But the honest version of this
alternative has already partly won — `libs/ledger-sqlite/v0.1.0`,
`libs/ledger/v0.5.0` and `libs/contracts/v0.5.0` were published on 2026-08-06
and 2026-08-07 *with these defects inside*, so the defective versions are the
only published ones. The rule "P0 before any release" can only bind future
tags. Superseding releases with migration notes are part of this work, not a
consequence of it.

**One big encoding module — a single `libs/encoding` owning every canonical
form.** Rejected: it would put identity seeds, derivation pre-images and storage
columns behind one import, and ADR-0013's layer topology exists to stop exactly
that. The three encodings share a *property*, not a dependency. A shared
property is a shared test, and this RFC proposes the tests rather than the
module.

**Route A for item 8** (redefine `precision`), taking `fdos.kernel.v2` and
folding in the `content_hash` rename while the window is open. Rejected as
designed, but recorded as the strongest alternative because it has one real
argument: a major boundary migrates every consumer by construction, so doing
several breaking things at once is cheaper than doing them separately. It loses
on the premise — Route B is not a smaller version of Route A, it is the correct
model, and buying a major version to delete one of two needed concepts is
paying for a regression.

**Encode the pre-image as protobuf and hash that.** The obvious move in a
repository whose contract surface is already protobuf, and rejected on the
publisher's own documented anti-guarantee: *"protobuf serialization is not (and
cannot be) canonical […] hashes of serialized protos are fragile and not stable
across time or space."* Field order is intentionally left undefined; the Go,
Java and C++ APIs each instruct users needing fingerprinting to "define their own
canonicalization specification". Taking this route would replace a **visible**
collision with an **invisible** instability — addresses that drift with a
library upgrade — which is strictly worse, because the current defect at least
reproduces.

## Prior art

Financial and cryptographic infrastructure has solved canonical encoding at
least four times and broken it at least as often. Every quote below was matched
against the cited source.

### Protobuf must not be the encoding, and its maintainers say so

The strongest single citation, because it removes an option that would otherwise
look obvious. Protobuf publishes a page whose only purpose is this claim
([Proto Serialization Is Not Canonical](https://protobuf.dev/programming-guides/serialization-not-canonical/)):

> "Unfortunately, protobuf serialization is not (and cannot be) canonical. […]
> This means that hashes of serialized protos are fragile and not stable across
> time or space."

The [encoding guide](https://protobuf.dev/programming-guides/encoding/) states
that even the reflexive case may fail — `Hash(foo.SerializeAsString()) ==
Hash(foo.SerializeAsString())` is listed among "checks [that] may fail" — and
gives the reason: "we intentionally leave serialization order undefined to allow
for more optimization opportunities."

And all three reference implementations independently instruct the reader to do
what this RFC does. Go's
[`proto.MarshalOptions`](https://pkg.go.dev/google.golang.org/protobuf/proto#MarshalOptions):

> "Users who need canonical serialization (e.g., persistent storage in a
> canonical form, fingerprinting, etc.) must define their own canonicalization
> specification and implement their own serializer rather than relying on this
> API."

Java and C++ carry the same sentence; Go adds "It is not guaranteed to remain
stable over time." A protobuf maintainer states the empirical form on
[issue #3417](https://github.com/protocolbuffers/protobuf/issues/3417#issuecomment-317816097):
"Do not use serialized proto as key. It's a bad practice and has been reported
repeatedly as the source of problem."

**Lesson: defining an explicit encoding here is following the upstream
instruction, not inventing a burden.**

### Frame with a type tag and a length; never join

Git's object address is not a hash of content —
[Pro Git §10.2](https://git-scm.com/book/en/v2/Git-Internals-Git-Objects) shows
the construction: `header = "blob #{content.bytesize}\0"`, then
`store = header + content`, then SHA-1 of that. Type tag, space, length, NUL,
payload. Two distinct `(type, content)` pairs cannot collide, and the book
proves the construction by reproducing `git hash-object`'s digest exactly.

ASN.1 DER ([ITU-T X.690](https://www.itu.int/rec/T-REC-X.690/en)) exists because
BER admits several encodings of one value; DER's scope is stated as rules that
"restrict the encoding of values to just one of the alternatives provided by the
basic encoding rules", and clause 10 is a *short enumerable list* — minimum-octet
definite lengths, no constructed string forms, one ordering for set components.

**Lesson: enumerate the injectivity restrictions explicitly, as DER does, and
prove the encoder against a reference digest, as Pro Git does. FDOS currently
joins with `":"` and `"\nparam="`.**

### Reject non-canonical input; do not normalize it

Bitcoin's [BIP 66](https://github.com/bitcoin/bips/blob/master/bip-0066.mediawiki)
is the case study in encoding laxity as a correctness bug. OpenSSL accepted
DER deviations; when it stopped, "it made some nodes reject the chain." The fix
was strictness at the boundary with an auditable predicate: "if the signature
does not pass the `IsValidSignatureEncoding` check below, the entire script
evaluates to false immediately."

[DAG-CBOR](https://ipld.io/specs/codecs/dag-cbor/spec/) shows what happens
without that. It requires "a single, canonical way of encoding any given set of
data" and then concedes: "decoders may relax strictness requirements by
default […] map entries may be accepted in any order". Its own listed
implementations diverge.

**Lesson: determinism written as encoder rules and unenforced on decode decays
into dialects. This is why §Enforcement specifies a corpus of *rejected* byte
strings rather than only encoder property tests.**

### One profile, no parameters, whole object

RFC 8785 (JCS) opens by naming the history: "Seasoned XML developers may recall
difficulties getting XML signatures to validate. This was usually due to
different interpretations of the quite intricate XML canonicalization rules."
W3C's own [XML Security 2.0 requirements](https://www.w3.org/TR/xmlsec-reqs2/)
diagnose it: "Support for arbitrary canonicalization algorithms, and the
complexity of the existing algorithms […] is also a source of problems."

**Lesson: permit exactly one algorithm with no negotiable parameters.**

### "Canonical" is not an adjective; it is a named profile

Three contradictions between specs that all claim determinism:

| Question | RFC 8785 (JCS) | RFC 8949 / DAG-CBOR |
|---|---|---|
| Sort map keys by | UTF-16 code units | bytewise encoding, length first |
| Float width | — | **shortest** (8949 §4.2.1) vs **always 64-bit** (DAG-CBOR) |

And protobuf's Java documentation admits its own map ordering "may be different
from the deterministic serialization in other languages". Two determinism specs
layered on each other disagree on a rule that changes the hash.

**Lesson: the accepting ADR must pin a profile and ship a shared test corpus,
never assert canonicality in prose.**

### State the scalar domain before the byte encoding

RFC 8785 Appendix D, on its own limits, is directly on point for a financial
ledger: "monetary data like `payMeThis` would presumably not rely on
floating-point data types due to rounding issues with respect to decimal
arithmetic", so "numbers that do not have a natural place in the current JSON
ecosystem MUST be wrapped using the JSON string type."

RFC 8949 §10 states the general principle this whole RFC serves:

> "Protocols should be defined in such a way that potential multiple
> interpretations are reliably reduced to a single interpretation."

**Lesson: name the value domain (decimal, scaled integer, instant) before the
bytes — which is also why item 8's two-concept answer is a prerequisite for
encoding a rounding context into a trace, not a separate concern.**

### The identity spec already requires what defect 1 breaks

RFC 9562 §6.5 states four requirements for name-based UUIDs. The fourth is the
injectivity property directly:

> "If two UUIDs that were generated from names (using the same canonical format)
> are equal, then they were generated from the same name in the same namespace
> (with very high probability)."

FDOS derives identities with UUIDv5 and violates this — not probabilistically,
but *constructively*, because `"ticker" + ":" + "x:y"` and `"ticker:x" + ":" +
"y"` are the same octet sequence. The spec also anticipates exactly this class of
error:

> "A canonical sequence of octets is one that conforms to the specification for
> that name form's canonical representation. A name can have many usual forms,
> only one of which can be canonical."

And it illustrates with DNS's three conveyance formats — `www.example.com`,
`www.example.com.`, `3www7example3com0` — one name, three renderings, one of
which must be chosen. **FDOS has never named the canonical octet form of a
claim.** That absence is defect 1, and §6.5 says naming it is the implementer's
obligation, not the library's.

### Ordering: RFC 3339's guarantee is conditional, and FDOS breaks a condition

The citation for defect 5 is RFC 3339 §5.1, which grants string-sortability
*only* under four simultaneous preconditions:

> "Assuming that the time zones of the dates and times are the same (e.g., all in
> UTC), expressed using the same string (e.g., all "Z" or all "+00:00"), and all
> times have the same number of fractional second digits, then the date and time
> strings may be sorted as strings […] The presence of optional punctuation would
> violate this characteristic."

`RFC3339Nano` omits trailing fractional zeros, so **"the same number of
fractional second digits" is exactly the precondition FDOS violates** — the
store's comparison is not merely fragile, it is outside the guarantee it relies
on. Two more of the four are reachable from ordinary input: `Z` versus `+00:00`
versus `-00:00` are three legal spellings of one instant, and `-` sorts before
`+` in ASCII.

The mechanical fix is not in dispute anywhere:

- RFC 9562 lists as a motivation for UUIDv6/v7 that "Introspection/parsing is
  required to order by time sequence, as opposed to being able to perform a
  simple byte-by-byte comparison" — the IETF treats *must-parse-to-order* as a
  defect worth a spec revision. §6.11: v6 and v7 "sort as opaque raw bytes
  without the need for parsing or introspection."
- KSUID states the two requirements plainly: "The timestamp uses big-endian
  encoding, to support lexicographic sorting", and its text form "is always 27
  characters" — fixed width and big-endian, nothing else.
- Bigtable's schema guide gives the one-line reproduction of the whole bug
  class: "lexicographically, 3 > 20 but 20 > 03."

**Lesson applied: the ordering invariant belongs on a fixed-width big-endian
integer, and the human-readable timestamp becomes a derived projection that
nothing compares.** ULID's own spec is the caution against over-claiming — even
it says "Within the same millisecond, sort order is not guaranteed" absent a
monotonic factory, so §Enforcement tests the sub-second region rather than
asserting totality.

### Sequences are monotonic, not contiguous — everywhere

PostgreSQL, `CREATE SEQUENCE` Notes:

> "Because nextval and setval calls are never rolled back, sequence objects
> cannot be used if "gapless" assignment of sequence numbers is needed. It is
> possible to build gapless assignment by using exclusive locking of a table
> containing a counter; but this solution is much more expensive than sequence
> objects."

Kafka's consumer documentation records the same shape from the reader's side:
transaction markers "are not returned to applications, yet have an offset in the
log. As a result, applications reading from topics with transactional messages
will see gaps in the consumed offsets." EventStoreDB's truncation leaves a stream
whose first visible revision is not zero.

All three treat the sequence as **assigned and stored**, never as a count of
surviving rows. `COUNT(*)` conflates *how many facts remain* with *what position
the next one holds* — and the reason gaplessness is expensive elsewhere is
exactly why FDOS gets it free: a single-writer append-only store never reserves a
number it might not use.

### Decimals: the operation names matter

Java separates the two concepts in its API surface, and the separation is the
design: `scale()` is "the number of digits to the right of the decimal point",
`precision()` is "the number of digits in the unscaled value", `setScale` moves
the point, and `round(MathContext)` truncates significant digits. Python's
`decimal` documents `quantize` as the money operation directly — "This method is
useful for monetary applications that often round results to a fixed number of
places" — while noting that the module "incorporates a notion of significant
places", which is the trap.

**Lesson applied: name the money operation after the scale concept
(`Quantize`/`SetScale`) and keep significant-digit rounding out of the ledger
path entirely.** A shared vocabulary is what stops the two knobs being swapped by
a caller who only has one word for both.

## Open questions

Each names who resolves it. None is resolved here.

1. **Route A or Route B for item 8.** This RFC proposes B with a prior-art
   argument. The *decision* is the accepting ADR's — **human**, because it is
   the one item that could commit the project to a major contract version.
2. **Migrate or refuse, for a pre-fix store.** The proposal is to refuse and
   name the migration. Whether a migration tool ships with it is scope —
   **human**, and cheap to defer only while the no-adopter assumption holds.
3. **Does the value-changing class need a governance home?** ADR-0024 has no
   row for "schema identical, every value different", and `buf breaking` reports
   success on it. Options: amend ADR-0024, add an ecosystem invariant, or state
   it in this RFC's ADR and leave the general case. **Human** — it is a
   governance-shape question, and it is the reason this set was nearly
   mis-sequenced as a major.
4. **What the FDOS root namespace UUID actually is.** The *method* is no longer
   open — RFC 9562 §6.6 requires a custom namespace to be a random UUIDv4 or
   UUIDv7 and forbids deriving one by hashing a string, which is what
   `identity.go:84` currently claims was done. What remains is the **value**:
   once chosen it is a constant forever, so it must be generated and recorded in
   the accepting ADR by a **human**, not produced by a session. It is the one
   value in this RFC that can never be changed again without repeating the whole
   migration.
5. **Which items leave this RFC as straight repairs.** Two of the nine explore no
   design at all — they implement accepted decisions the code does not honour:
   **item 6** (ADR-0034 already assigns sequence assignment to the store and
   records it at rung 1) and **item 7** (ADR-0008 already requires that precision
   loss be signalled and that no rounding mode be privileged). Both are on the
   critical path for a first queryable answer, and splitting them into their own
   ADR now would unblock the read path sooner than the design questions here can
   be settled. **Human**, on sequencing grounds.

## Consequences

### Easier

- An as-of read means what it says, in both stores, at sub-second precision.
- Two different claims can no longer become one identity by accident, which is
  the failure this system's whole identity design exists to prevent.
- "Round to the cent" becomes expressible, and the money kernel gets its first
  reason to have a caller.
- A store that cannot be ordered correctly says so at `Open` instead of
  answering wrongly.

### Harder

- **`EntitiesIdentified` has to become real before the identity fixes land** — a
  domain payload type, a codec case, and a projection traversal, none of which
  exist. This RFC turns three known gaps into blockers.
- `RoundingContext` carries two concepts, and a caller must now know which one it
  means — including that `quantize` raises a domain error when scale and precision
  conflict. That is the cost of the domain genuinely having two, and prior art
  says the alternative is worse.
- Three superseding releases with migration notes, for tags already published
  with these defects inside.

### Impossible

- **Translating a derivation address across the fix.** No derivation record is
  persisted, so there is no pre-image to replay. This is the one class with no
  route back, and it is why the sequencing argument is about these two fixes
  rather than about all nine.
- Reading a pre-fix store with a post-fix binary without an explicit migration.
  Deliberate: the alternative is reading it wrongly and silently, which is what
  happens today.
- Settling mechanically whether two previously-collided claims were one entity.
  That judgement is reserved to a recorded `EntitiesIdentified` fact by ADR-0007,
  and this RFC does not propose inferring it.
