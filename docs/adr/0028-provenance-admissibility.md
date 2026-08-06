---
id: ADR-0028
title: A SourceRef is a content hash with an unspecified referent, and an interpreter is always named
status: Accepted
date: 2026-08-06
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0028 — A SourceRef is a content hash with an unspecified referent, and an interpreter is always named

## Context

Records what [RFC-0011](../rfc/0011-provenance-admissibility.md) settled.

[ADR-0010](0010-provenance-envelope-reference-versioning.md) left open, in as
many words, whether `SourceRef` is an opaque reference resolved privately. `E4`
makes provenance an **admission criterion** for the ledger, so an unsettled
`SourceRef` means the ledger's admission rule is authored by whoever produces
facts for it rather than by the ledger.

The question stopped being theoretical when a producer began populating the
field, and became urgent when `E9` obliged this repository to accept facts from
producers it has never met.

## Decision

### A `SourceRef` is a content hash

`sha256:` followed by 64 lowercase hexadecimal characters, carrying three
producer assertions: the addressed record is immutable, it is replayable, and it
identifies the **acquisition** rather than any single artifact within it.

**FDOS never dereferences it.** Opacity is about resolution and is preserved
entirely; what changes is that the *form* is now known, which it never needed to
stop being in order for the referent to stay private (Constitution §13).

The justification is replay symmetry with `DerivationRef`, not the founding
document. Both are immutable records too large to inline, addressed by hash so
lineage stays a DAG and identical records deduplicate; the only difference is
who stores them, which justifies not resolving and never justified not knowing
the shape.

### The field is renamed `value` → `content_hash`

Wire-compatible, source-breaking. Generated Go moves from `GetValue()` to
`GetContentHash()`.

The form check catches accident, not intent — a well-formed digest of nothing
passes — which means the **identifier carries most of the remaining enforcement
weight**. `value` on a provenance reference is the name that invites a URL, an
account id or a path; `content_hash` makes that mistake visible at the call
site, which is the only place anyone looks.

It happens now because the cost only rises: today one consumer, inside this
programme, whose corresponding work is still `Draft`. After a public ingestion
library exists, producers read `GetValue()` and never read a proto comment, and
the rename stops being an `E7` migration and becomes a rename in other people's
code.

### An interpreter is always named, and `unmediated` is a value

`InterpreterRef` stays **required**. A producer that interpreted nothing
programmatically uses the reserved name `unmediated`, versioned by this
convention rather than by code that does not exist.

There is deliberately **no value meaning "I do not know"**. `unmediated` is an
assertion — *no code read this; it was transcribed as stated* — and a producer
that cannot make it must name its pipeline or cannot submit. A sentinel that
absorbs both "no code read this" and "nobody filled this in" makes the field
optional in practice while claiming to be required.

### `collected_at` is documented wrongly and is corrected

It says "When FDOS fetched it". FDOS fetches nothing, and for an
externally-produced fact never will.

## Consequences

### Positive

- `E4` becomes enforceable. A grammar check at admission needs no knowledge of
  the referent, and turns an invariant that has been prose since it was written
  into something a machine can refuse.
- RFC-0010's conformance kit acquires a subject: "admissible" is now a statement
  rather than folklore.
- A producer with a file gains something honest to say. The provenance shape
  stops being fitted to producers that run code.
- One consumer migration window covers the grammar, the comment corrections and
  the rename, rather than two.

### Negative

- **It catches accident, not intent.** A well-formed digest of nothing passes,
  and a producer determined to lie is unaffected. Anyone reading a parse check
  as provenance verification has misread it, and this ADR says so rather than
  letting the check imply more than it does.
- **`unmediated` is rung 6.** Nothing detects a producer using it while having
  interpreted programmatically. It is an affordance for honesty, not a control.
- **It is a source-breaking contract change**, paid by a consumer for a gain
  that is real but not urgent to them. The mitigation is timing, not avoidance.
- **`unmediated` is vocabulary**, and no type governs vocabulary. It joins the
  open half already recorded for claim schemes: a second sentinel invented by a
  future producer would fragment the same way `"ticker"` and `"symbol"` do.
- **Deciding while the corresponding work is `Draft` elsewhere** means telling a
  producer the answer rather than agreeing it. That is correct — provenance is
  this repository's concern — but it obliges the consumer issue to open before
  merge, not after.

### Enforcement

The rung belongs to the **property**, not to the mechanism. A missing mechanism
and an unprotected guarantee are different facts, and it is the second that
changes what anyone should trust.

| Property | Rung **today** | Mechanism |
|---|---|---|
| Provenance carries an admissible source | **6 — nothing** | no grammar is validated anywhere; a malformed `SourceRef` would be accepted by every path that exists |
| A fact cannot exist without an interpreter | **1 — type system** | `provenance.NewProvenance` refuses an empty one |
| `unmediated` means what it asserts | **6** | nothing; an affordance for honesty, not a control |
| The referent is a real acquisition record | **none, by design** | FDOS never dereferences |

The first row is the one that matters. Writing "the check is not yet built"
would describe a mechanism; what a reader needs is that **admissibility is
unguaranteed today** — this decision states a rule that nothing enforces, and
will keep stating it until the admission path exists.

> **Banner — this clause understates the available enforcement.**
>
> The table above says no grammar is validated anywhere and that the check has
> no execution site. **A chokepoint exists and was missed:** `libs/kernel-wire`
> decodes provenance through `provenance.NewSource`, so a grammar rule in that
> constructor would apply to every value crossing the wire codec, today, without
> any admission path.
>
> The correction matters beyond accuracy. An ADR that understates available
> enforcement gets cited later as evidence that enforcement was impossible.
>
> **It is still not implemented, and the reason is stronger than scope.**
> Validating in the codec is enforcement at **decode**, not at admission, and
> those differ in what they destroy. Admission rejects bad *new* data. Decode
> makes *existing* data unreadable — if a producer's stored references are not
> `sha256:` plus hex, switching this on does not break their build, it makes
> facts they already hold unrecoverable through the normal path.
>
> That is a data-plane change wearing validation's clothes. It needs its own
> decision, with migration of existing references as the central question rather
> than a footnote.

### The dependency, named so it is not rediscovered

The grammar check has **no execution site until there is an admission point**,
and the admission point is the public ingestion path
[RFC-0010](../rfc/0010-the-public-surface-receives-a-claim.md) specifies and M8
builds.

That ordering is not incidental: `E9` obliges the ingestion path, the ingestion
path gives the check somewhere to run, and the check is what moves the first row
above from rung 6 to rung 3. Until then this ADR is a rule with a stated rung of
6, which is the honest description of a decision waiting for its enforcement
site.

**Execution-context question.** **If a malformed `SourceRef` were submitted
right now, no check in this repository would report it — because there is
nothing to submit it to.** The silence is total, and it is the answer.

## Alternatives considered

**Stay opaque and unconstrained.** Rejected: it conflates resolution with form
and leaves admission authored by producers.

**Specify the referent.** Rejected: it crosses into how a producer stores
things, which Constitution §13 puts outside this repository.

**Document the grammar and keep `value`.** Rejected: it switches off the
strongest available defence — the name — to save one migration for a consumer
that has not hardened. An earlier draft of the RFC leaned on the published
proposal not having requested the rename; that argument was **withdrawn** as
the same class of reasoning the RFC refuses when it declines to justify itself
by the founding document.

**Make `InterpreterRef` optional.** Rejected: one class of producer would cost
every other producer its replay guarantee, and an optional field becomes an
absent one.

## Notes

Implementation is a separate change (§10): the proto edit, the regenerated Go,
the kernel constructors and the version bump. **The consumer issue opens before
that merges**, per `E7` and `fdos:ADR-0024`.

D4 is closed. D1, D2 and D3 remain open. The rejection question, batch
provenance, and signed attestation over the acquisition record are named in
RFC-0011 §7 as out of scope with reasons.
