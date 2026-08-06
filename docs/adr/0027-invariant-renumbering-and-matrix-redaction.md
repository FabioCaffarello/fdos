---
id: ADR-0027
title: Invariants are E1-E9, the matrix names no provider, and the open core must stand alone
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0027 — Invariants are E1-E9, the matrix names no provider, and the open core must stand alone

## Context

Records what [RFC-0009](../rfc/0009-renumbering-invariants-and-redacting-the-matrix.md)
settled.

The Tier-0 corpus shipped at `ecosystem/v0.1.0` naming two financial
institutions and one central bank in the responsibility matrix. The
public/private boundary forbids naming a provider in this repository at all, and
the text arrived through the one channel that could not be corrected on the way
in: a Tier-0 block reproduced verbatim by instruction.

Separately, the ecosystem invariants were numbered `I1`–`I8` while the
downstream Integration Charter carries its own `I-1`…`I-10`. And the open-core
position now carries a product requirement the architecture tests did not: the
public repository must be *useful* alone, not merely developable alone.

Four Tier-0 edits. Two amendments had already shipped in one day, and that
cadence was recorded as close to its limit.

## Decision

**One amendment, published as `ecosystem/v0.3.0`, carrying all four.** They
share a publication event and nothing else; travelling together bills the
consuming repository one reviewed re-sync instead of four.

1. **The matrix names no provider.** Provider-agnostic wording throughout. What
   was disclosed, and what of it is now permanent, is recorded in
   [`../disclosure.md`](../disclosure.md).
2. **Private module identifiers leave the current documents** and stay in the
   historical ones where they are the subject of a decision rather than
   incidental to it.
3. **`I1`–`I8` become `E1`–`E8`.** The mapping is the identity. Three immutable
   ADRs keep the old numbers and take banners pointing at the mapping.
4. **`E9` is added:** the open core must build, test, run and deliver value with
   the private repository absent, unlicensed and unbuildable.

## Consequences

### Positive

- The rule the boundary states is now the rule the corpus follows. A rule
  visibly disobeyed in writing corrodes the next one.
- `E2` in an FDOS document and `I-2` in the Charter can no longer be confused,
  which matters most to whoever reads them six weeks from now with neither
  session available to ask.
- E9 turns the public ingestion path from an assumption into an obligation
  *before* an ingestion path is built. Built first, it would have assumed a
  private producer and the requirement would have been discovered by an adopter.
- The redaction is what the second boundary test asks for independently: wording
  shaped by which providers exist today is provider-shaped.

### Negative

- **The redaction does not un-publish anything.** Tags `ecosystem/v0.1.0` and
  `v0.2.0` carry the names permanently, the ruleset refuses tag deletion, git
  history is public, and the downstream vendored copy holds the old text until
  it re-syncs. This decision changes what is *said*, not what was *seen*.
- **E9 is admitted unmet.** There is no public ingestion path, so E9 is at
  rung 6 from the moment it is written, and it will stay there for at least one
  milestone. An invariant the repository does not satisfy is a real cost, paid
  deliberately.
- **The renumbering strands the old numbers in three immutable decisions.**
  Banners make that navigable, not invisible. Anyone reading `fdos:ADR-0023`
  alone still sees `I2`.
- **Four changes in one amendment is worse for review than four amendments.**
  A reviewer who disagrees with the redaction must accept or reject the
  renumbering with it. That is the price of not billing four re-syncs, and it is
  the more defensible trade only because none of the four is contentious in
  isolation.

### Enforcement

**Rung 6 throughout, and the first item cannot climb.**

A mechanical check for "no provider is named here" requires a list of provider
names committed to this repository to match against. That list *is* the
disclosure, in a more complete and more machine-readable form than the leak it
would prevent. The mechanism costs more than the defect, so it is not built, and
the register says so rather than implying coverage.

The narrower half is checkable — that no private module path appears in the
current tree — and would not have caught what was actually disclosed. It is not
built either, for that reason.

**Execution-context question**, for the mechanisms that do exist here: the
banners are inert text, checked by nothing. `make adr-check` verifies ADR
front matter and `make adr-immutability-check` verifies that no line was
*removed* — both run in CI and on pre-commit, and both would pass if every
banner were deleted tomorrow. What they observe is structure, not meaning.
**If the subject of this decision were simply absent — if the banners were never
added — every check in this repository would stay green.** That silence is the
answer, and it is why this is rung 6.

## Alternatives considered

**Four separate amendments.** Rejected: four reviewed re-syncs for one
publication event, against a cadence already flagged.

**Rewrite the historical documents to remove every identifier.** Rejected:
destroys the reasoning and violates `fdos:ADR-0000`.

**Rewrite git history and recreate the tags.** Rejected: the consumer pins and
byte-compares those tags, the ruleset refuses deletion, and forks and caches
retain the objects. It trades a structural disclosure for a broken vendoring
contract.

**Leave the provider names** on proportionality — a national exchange and a
central bank are public infrastructure, and nothing technical was revealed.
Rejected as a rule question rather than a severity question. The proportionality
argument is true and is recorded in the register, where it informs judgement
without licensing the next exception.

**Defer E9.** Rejected: the invariant is what makes the ingestion path a
requirement, and writing it after the build would let the build decide it.

## Notes

Consumers re-pin to `ecosystem/v0.3.0` and re-sync. Retiring anything on their
side is theirs.

**This ADR invalidates a premise, not a decision.** The rejection of the
acquisition-contract promotion in
[`../ecosystem/roadmap.md`](../ecosystem/roadmap.md) rested on there being no
producer for `AcquisitionEnvelope` and `ProviderObservation`. E9 supplies one:
any third party. Reversing that rejection needs its own RFC and is not done
here; it is marked so nobody implements against a premise already known to be
false.

D1, D2 and D3 remain open. D4 remains M8's gating deliverable and is untouched.
