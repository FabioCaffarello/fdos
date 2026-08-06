---
id: ADR-0023
title: The ecosystem boundary is written down, and contracts flow one way
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0023 — The ecosystem boundary is written down, and contracts flow one way

## Context

FDOS is built by two repositories developed in parallel by two agents that
cannot see each other: `fdos`, public and Apache-2.0, and `fdos-connectors`,
private. Constitution §13 establishes Open Core and says private implementations
must depend only on published contract versions. [ADR-0002](0002-license.md),
[ADR-0004](0004-module-granularity.md) and
[ADR-0020](0020-open-core-boundary-and-pull-request-workflow.md) each settle a
piece of the mechanism.

None of them says **which repository owns which concern**. That was carried in
the briefing prompt each session receives, and nowhere else.

The cost of that has now been observed rather than predicted. `fdos-connectors`
independently built a vendoring manifest, a platform drift check, and a
host↔plugin wire contract, checking them against a boundary that existed only
inside a prompt. It also, in `fdos.kernel.v1.SourceRef`, answered a question
[ADR-0010](0010-provenance-envelope-reference-versioning.md) had explicitly left
open — by construction, without an ADR on either side, and without anything in
`fdos` being able to notice.

The results happen to be compatible and careful. That is the uncomfortable part:
the boundary held on judgement, and judgement is not a mechanism. The next
divergence has no reason to be as lucky, and the failure mode is silent —
nothing in either build fails when the two repositories disagree about who owns
a concern.

This is Constitution §14 applied to the ecosystem rather than to one repository:
a decision that lives only in a conversation is not a decision.

## Decision

**FDOS publishes an ecosystem governance corpus at
[`docs/ecosystem/`](../ecosystem/), and it is authoritative for both
repositories.**

- [`invariants.md`](../ecosystem/invariants.md) — I1–I8, the rules that hold
  across every repository.
- [`boundary.md`](../ecosystem/boundary.md) — the responsibility matrix, the
  four boundary tests, and the register of disputed items.
- [`contracts.md`](../ecosystem/contracts.md) — the registry of what is
  published, at what version, consumed by whom.
- [`dependencies.yaml`](../ecosystem/dependencies.yaml) — known cross-repository
  edges.
- [`roadmap.md`](../ecosystem/roadmap.md) — milestone alignment by dependency.

**Contracts flow one way.** `fdos` defines canonical contracts;
`fdos-connectors` consumes them at a pinned version. There is no reverse edge in
types. A need originating downstream travels upstream as an *issue*, becomes an
RFC in `fdos`, and returns as a published contract — never as a type defined
downstream.

**The invariants and the responsibility matrix are Tier 0.** Authored here,
vendored verbatim, never edited downstream, amended only by an RFC in `fdos`
followed by an ADR in both repositories. They are delimited by explicit markers
in the files so that "verbatim" is checkable rather than aspirational.

**A disputed item is not settled by whoever writes code first.** Five are open
and recorded (D1–D5). Implementing against one settles it by accident, which is
the specific failure this ADR exists to prevent.

**What this ADR deliberately does not do:** it does not resolve D1–D5. In
particular it does not decide what a `SourceRef` must resolve to. Recording that
the question is open, load-bearing, and already answered on one side is the
whole of the contribution here.

## Consequences

### Positive

- The other repository can discover the boundary by reading a file instead of
  by asking, which is what I3 requires and what has not been true until now.
- Disagreements become locatable. Two divergences were found while writing the
  corpus — the "Python toolchain" row is counterfactual, and the "contracts"
  row is broader than practice — and both are now written down instead of being
  absorbed by whichever session read the charter last.
- The `SourceRef` question is visible before `fdos` has an ingestion path. It
  costs an ADR now; it would cost a migration of a live ledger later.

### Negative

- **`fdos` becomes a bottleneck by construction.** Every downstream need routes
  through an RFC here. That is the intended trade — one definition of money —
  but it is paid by the other repository in latency, and this ADR does nothing
  to reduce it.
- **The matrix is inherited from a charter that contains at least one
  counterfactual row.** Ratifying it means ratifying its errors until they are
  amended. They are listed rather than fixed, because Tier 0 must stay
  byte-identical to the copy the other repository vendors, and unilaterally
  "improving" the canonical copy is exactly the drift the tier exists to stop.
- **Nothing verifies that the corpus was vendored, or vendored unmodified.** The
  consumer checks the Constitution byte-for-byte against its own copy; `fdos`
  cannot check anything, because the consumer is private and depending on it
  would be the reverse edge. Verification is structurally one-directional.
- **Five open disputes now have a home, which makes it easier to leave them
  open.** A register can become a place where questions go to be filed rather
  than answered. D4 has a stated trigger for that reason.

### Enforcement

**Rung 5 — documentation — for the boundary itself, and honestly so.**

The Tier-0 markers make a byte comparison *possible*, and the consumer already
performs that comparison for the Constitution using the same technique. But
`fdos` cannot run it: it cannot read a private repository, and building a
mechanism that did would violate the boundary being enforced. From this side the
rung is 6, and no amount of work here changes that.

What *is* enforced, and was before this ADR: `make proto-check` runs `buf
breaking` on every pull request, so the contract surface cannot break silently
(rung 3). `make consumer-check` proves the module resolves with `GOWORK=off`
and no `replace` (rung 3). Those cover the mechanical half of I2. The
responsibility matrix is the half that no check can reach.

To climb: publish the corpus at a tag, have the consumer vendor it pinned with a
drift check on its side, and treat a version bump as a migration. That makes the
consumer's build the enforcement point — the only place it can live.

## Alternatives considered

**Leave the boundary in the prompts.** Rejected: it has already failed once, in
a way that produced a silent unilateral answer to an open ADR question. Prompts
are not versioned, not diffable, and not readable by the other party.

**Write the boundary but keep it advisory.** Rejected as a distinction without a
difference. An advisory boundary and a prompt-only boundary have the same
enforcement (none) and the same failure mode, but the advisory one additionally
implies coverage.

**Resolve D1–D5 in this ADR while the corpus is being written.** Rejected, and
it was tempting: the boundary tests give a defensible answer for D3 and D5
today. But D4 changes what provenance means, the charter reserves disputed items
for both repositories to ratify, and settling five questions in an ADR whose
subject is *governance structure* would be the same accident this ADR names —
committed by the document that forbids it.

**Define `fdos.acquisition.v1` now, per the brief's promotion ruling.**
Rejected; the argument is in [`roadmap.md`](../ecosystem/roadmap.md). In short:
its sequencing premise expired when the contract surface shipped at M4, the
starvation it predicted did not occur, and the types would arrive with a
producer and a consumer that both already exist elsewhere.

## Notes

Open and deliberately unresolved here: D1 browser runtime provenance, D2 the
authentication split, D3 where normalisation stops, D4 what a `SourceRef` must
resolve to, D5 which contracts constitute the contract surface. All five in
[`boundary.md`](../ecosystem/boundary.md).

D4 is the one with a deadline attached: it is free while `fdos` has no ingestion
path, and stops being free at the first externally-produced fact.

The mirror issue announcing this corpus to `fdos-connectors` is not opened by
this change, because it must reference the tag that publishes the corpus and no
such tag exists until this merges. Recorded as B-009 in
[`../blocked.md`](../blocked.md).
