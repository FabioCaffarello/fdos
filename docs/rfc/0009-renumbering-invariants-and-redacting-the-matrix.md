---
id: RFC-0009
title: Renumbering the ecosystem invariants and redacting the boundary matrix
status: Accepted
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0009 — Renumbering the ecosystem invariants and redacting the boundary matrix

> **Accepted**, recorded by
> [ADR-0027](../adr/0027-invariant-renumbering-and-matrix-redaction.md).

## Why one RFC for four changes

Each of these is a Tier-0 edit, and Tier 0 is amended only by an RFC here plus
an ADR in both repositories. Every amendment bills the consuming repository a
reviewed re-sync of the vendored corpus.

Two amendments had already shipped in a single day, and that cadence was
recorded at the time as close to its limit. Shipping four separately would bill
four re-syncs for changes that share one publication event. **They travel
together for that reason and no other** — none of them depends on the others.

## Change 1 — Redact the provider names

The matrix named two financial institutions and one central bank as examples of
what the private side integrates, and gave a naming convention for private
plugin repositories. The public/private boundary forbids naming a provider in
this repository at all.

The examples carried no argumentative weight: the row is about *who owns
provider plugins*, and the answer does not depend on which providers exist. The
redaction to provider-agnostic wording is what the second boundary test asks for
anyway — an abstraction shaped by today's providers is provider-shaped.

Recorded in [`../disclosure.md`](../disclosure.md), including what is now
permanent: the tags, the history, and the downstream vendored copy until it
re-syncs.

## Change 2 — Remove private module identifiers from current documents

Module and proto-package names belonging to the private repository appeared in
the blocked register, the roadmap and the boundary document.

They are removed there and **retained** in `fdos:RFC-0008` and `fdos:ADR-0026`,
where those identifiers are the subject of the decision rather than incidental
to it. A decision whose subject has been redacted out of it is not a preserved
decision.

## Change 3 — Renumber `I1`–`I8` to `E1`–`E8`

The downstream Integration Charter carries its own `I-1`…`I-10`. Two unrelated
rule sets sharing one prefix across two repositories is a misreading waiting to
happen, and this program has already been caught twice by facts that existed in
only one place.

The mapping is the identity. The cost is that **the old numbers survive in three
immutable ADRs** — `fdos:ADR-0023`, `fdos:ADR-0024`, `fdos:ADR-0026` — which are
superseded rather than rewritten. Each takes a banner pointing at the mapping.
Renumbering after vendoring and citation is a migration, not an edit, and the
banners are what makes it one rather than a silent break.

## Change 4 — Add `E9`, the open core must be usable alone

> **E9 — The open core must be usable alone.** `fdos` must build, test, run and
> deliver value with the private repository absent, unlicensed, and unbuildable.

The offline test already required `fdos` to be *developable* without the private
side. E9 raises that to *useful*, which is a different claim and a much harder
one: if the only path for data into FDOS is a private repository, the open core
is a demonstration rather than a platform, and an adopter discovers that within
an afternoon.

E9 is deliberately admitted while unmet. `fdos` has no public ingestion path
today, so E9 sits at **rung 6** from the moment it is written. Stating an
invariant the repository does not yet satisfy is the cost; the alternative is
discovering the requirement after building an ingestion path that assumes a
private producer.

## Consequences

- The corpus becomes `ecosystem/v0.3.0`. The consumer re-pins and re-syncs once.
- Reading a bare `I2` in an FDOS decision predating `v0.3.0` means `E2`; the same
  token in the Charter means something else entirely. That ambiguity is removed
  going forward and cannot be removed going backward.
- E9 makes the public ingestion path a milestone obligation rather than a
  README section, and **invalidates an argument already recorded in
  `roadmap.md`** — see below.

## What this obliges next, and does not do here

The rejection of the acquisition-contract promotion recorded in
[`../ecosystem/roadmap.md`](../ecosystem/roadmap.md) argued that
`AcquisitionEnvelope` and `ProviderObservation` would arrive with no producer
and no consumer, because the private side has a host-plugin contract of its own.

**E9 supplies the producer that argument assumed did not exist: any third
party.** The rejection was sound on the evidence available and is wrong on this
evidence.

Reversing it needs its own RFC, and it is not reversed here. This RFC marks the
argument as superseded-in-premise so that nobody implements against it in the
meantime.

## Alternatives considered

**Ship the four changes as four amendments.** Rejected: four reviewed re-syncs
for one publication event, against a cadence already flagged as near its limit.

**Rewrite the historical documents to remove every identifier.** Rejected: it
destroys the record of the reasoning and violates `fdos:ADR-0000`. Banners
preserve both the decision and the correction.

**Redact and rewrite git history.** Rejected: the consumer has pinned the
existing tags and byte-compares them, the tag ruleset refuses deletion, and
forks and caches would retain the objects anyway. It trades a structural
disclosure for a broken vendoring contract.

**Leave the provider names**, on the argument that a national exchange and a
central bank are public infrastructure. Rejected: the rule exists to be
followed before someone judges each case, and a rule visibly disobeyed in
writing corrodes the next one. The proportionality argument is recorded in the
disclosure register instead, where it belongs.

**Defer E9 until a public ingestion path exists.** Rejected: the invariant is
what makes the path a requirement. Writing it after building would let the
build decide it.
