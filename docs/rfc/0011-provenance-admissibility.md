---
id: RFC-0011
title: Provenance admissibility — a specified form, an unspecified referent, and an honest interpreter
status: Accepted
date: 2026-08-06
authors:
  - "@FabioCaffarello"
---

# RFC-0011 — Provenance admissibility

> **Accepted**, recorded by
> [ADR-0028](../adr/0028-provenance-admissibility.md).

Settles **D4**. Also settles the `InterpreterRef` finding raised by
[RFC-0010](0010-the-public-surface-receives-a-claim.md), because it is the same
message and deciding provenance twice would be worse than deciding it once.

## 1. What was open

[ADR-0010](../adr/0010-provenance-envelope-reference-versioning.md), Notes:

> Open, deliberately: whether `SourceRef` is an opaque reference resolved
> privately (Open Core, Constitution §13).

It has since been answered by construction on the producing side, while
remaining unanswered here. `E4` makes provenance an *admission criterion* for
the ledger, so leaving it open means the ledger's admission rule is authored by
whoever produces facts for it.

## 2. Opaque and unconstrained are different properties

This is the distinction ADR-0010 conflated, understandably: at the time there
was no producer to constrain.

- **Opaque** is about *resolution*. FDOS never dereferences a `SourceRef`, never
  learns what it addresses, and gains no dependency on any store. Constitution
  §13 and the offline test both hold.
- **Unconstrained** is about *form*. Nothing ever required it.

Specifying the form costs none of the opacity.

## 3. The form

A `SourceRef` is an **algorithm-prefixed content hash**: `sha256:` followed by
64 lowercase hexadecimal characters.

It carries three assertions, made by the producer:

1. The addressed record is **immutable**.
2. The producer asserts it is **replayable** — the record can be produced again.
3. The reference identifies the **acquisition**, not any single artifact within
   it. One acquisition may yield a capture and a snapshot; "the artifact digest"
   would be ambiguous.

### The argument is replay, not intent

`DerivationRef` and `SourceRef` are the same construct: an immutable record too
large to inline, addressed by hash so lineage stays a DAG and identical records
deduplicate. `RFC-0004` makes that argument explicitly for derivations — *"a
value derived from a thousand facts would otherwise carry a thousand entries,
making a fold quadratic"* — and the acquisition side inherits it unchanged,
because one acquisition yields many claims.

The only real difference is **who stores the record**. FDOS stores derivation
records because FDOS derives; producers store acquisition records because
producers acquire. That difference justifies FDOS not *resolving* a `SourceRef`.
It never justified FDOS not knowing what shape one is.

The FKOS vision does list Checksum as a first-class provenance field. That is
**corroboration and it is intent, not decision.** The argument above stands
without it, which is the test a founding document has to pass before it can be
cited.

## 4. The field is renamed to `content_hash`

`SourceRef.value` becomes `SourceRef.content_hash`. Wire-compatible — same tag,
same type — and **source-breaking**: generated Go moves from `GetValue()` to
`GetContentHash()`.

### Why now, when the cheaper option is to document the grammar and stop

**The name is doing most of the enforcement.** A form check catches accident,
not intent: a well-formed digest of nothing passes, and nothing here detects it.
That is stated plainly below and in the ADR. But it has a consequence — if the
mechanism is low-rung by construction, then the *identifier* is the strongest
remaining defence, and `value` on a provenance reference is precisely the name
that invites a URL, an account id, or a file path. `content_hash` makes the
mistake visible at the call site, which is the only place anyone looks.

Deferring would have left the strongest available defence switched off in order
to save one migration for one consumer.

**The cost is monotonically increasing and there is a step change in sight.**
Today there is one consumer, inside this programme, whose corresponding work is
still `Draft` and therefore has not hardened. After the public ingestion library
lands (RFC-0010), producers exist who never read a proto comment — they read
`GetValue()`. Renaming then is not an `E7` migration; it is renaming a method in
other people's code, which means never.

"Fold it into the next genuinely breaking change" is a bet that a break arrives
before the public surface does. If the public surface arrives first, the rename
becomes permanently unaffordable — and a deferred rename waiting for a ride
rarely boards, because the ride always turns up when nobody wants extra scope.

### An argument this RFC withdrew

An earlier draft leaned on *"the published proposal explicitly did not request
the rename."* That is the same class of reasoning as *"the founding document
asked for Checksum"* — which §3 refuses as a reason. **The silence of a proposal
is not an argument in either direction**, and applying the right epistemic
standard in one section while relying on its inverse in another is the failure
worth recording rather than quietly fixing.

### One migration window, not two

Settling D4 changes this message regardless: the grammar is documented, and
`collected_at`'s comment is wrong (below). The rename ships **in the same
consumer migration window** as those changes rather than in a second one. That
is what makes this cheap in practice and not only in argument.

## 5. `collected_at` is documented wrongly

> `// When FDOS fetched it.`

FDOS fetches nothing, and for an externally-produced fact it never will. A
producer fetches; the moment a queue sits between acquisition and admission the
two times diverge and the field silently means the wrong one. Corrected in the
same change.

## 6. An honest interpreter

`NewProvenance` requires a non-empty `Interpreter` at rung 1, and
`NewInterpreter` requires both a name and a version — correctly, so that a
report regenerates with the interpreter of its time.

A producer that interpreted nothing programmatically has no honest value to put
there. `{name: "manual", version: "1"}` asserts a versioned interpretation that
does not exist and cannot be replayed. **That is the provenance shape
over-fitting its first producer:** every producer so far runs code, so the field
looked universal.

**Making it optional is rejected.** It would weaken replay for every producer to
accommodate one class, and an optional field becomes an absent field.

**A reserved value is adopted.** `InterpreterRef{name: "unmediated", version:
"<convention version>"}` asserts a positive fact: *no code interpreted this
value; it was transcribed as the source stated it.* The version is the version
of this convention, not of code that does not exist — so if FDOS later changes
what unmediated means, the two are distinguishable on replay.

### There is deliberately no value for "I do not know"

This is what stops the sentinel becoming a dumping ground.

`unmediated` is an **assertion**, not an absence. A producer that does not know
what interpreted its data may not use it — it must name its pipeline, or it
cannot submit. If "no code read this" and "nobody filled this in" collapsed into
one value, the field would be optional in practice while claiming to be
required, which is worse than being optional honestly.

### Its rung, labelled

**Rung 6.** Nothing detects a producer that used `unmediated` while having
interpreted programmatically. It is an **affordance for honesty, not a
control** — it gives a truthful producer something true to say, which is all it
does.

Recorded because a sentinel described as though it were checked is exactly the
kind of rule this repository is supposed to label.

## 7. What is not settled here

- **Whether a rejection carries a `SourceRef`.** Admitting one to the ledger
  means publishing a payload type for it, which is a payload decision riding on
  a grammar decision. Its own RFC.
- **Batch provenance.** Held answered-by-construction: claims from one
  acquisition share a `SourceRef`, and content addressing makes the repetition
  free. Per-fact provenance with an identical reference is strictly more
  flexible than a batch construct with overrides.
- **Signed attestation over the acquisition record.** It needs a trust root that
  does not exist while D2 is open. Out of scope, and named as a dependency
  rather than a limitation.

## 8. Alternatives considered

**Stay opaque and unconstrained.** Rejected: it conflates two properties, and it
leaves the ledger's admission rule authored by its producers.

**Specify the referent — say what an acquisition record must contain.**
Rejected: it crosses into how a producer stores things, which Constitution §13
puts outside this repository. Specifying form gets the admission check without
the intrusion.

**Document the grammar and keep `value`.** Rejected on §4: it switches off the
strongest available defence to save one migration for a consumer that has not
hardened.

**Make `InterpreterRef` optional.** Rejected on §6: one class of producer would
cost every other producer their replay guarantee.
