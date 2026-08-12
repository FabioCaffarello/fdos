---
title: Engineering Principles — operating additions proposed by the audit
status: Provisional — proposal from the 2026-08-07 architectural audit
date: 2026-08-07
---

> **Provisional.** This document is a proposal produced by the 2026-08-07
> architectural audit. It is not accepted. Nothing may be implemented against
> it until an RFC and ADR accept the relevant part (ADR-0000,
> [AGENTS.md](../../AGENTS.md)). Where this document conflicts with an
> accepted ADR, the ADR governs until superseded.
>
> **This document does not amend the Constitution.**
> [`docs/constitution.md`](../constitution.md) remains the ratified authority;
> any change to a principle goes through its amendment procedure (RFC, version
> bump, ADR). What follows proposes additions to *practice* — the way work is
> done under the existing principles.

# Engineering Principles

Eight operating principles the audit recommends adopting. Each exists because
the audit found the failure it prevents already present in the repository.

## 1. Adversarial negative testing at mechanism boundaries

A check that has never gone red is unverified — the repository already says
this. The audit's finding is one step deeper: **a check that has only gone
red against the direct form of a violation is unverified at its boundary.**
The purity analysers fail on `var x float64` and pass on a named float type
from a sibling package; the wire conformance suites only ever see messages
the codec itself produced; the fold-order property test bounds its generator
to 20 digits and certifies a property that breaks at 96. Every rung-1 and
rung-2 claim must carry a negative test of the *indirect* form: the named
type, the function-value indirection, the message this build did not write,
the input at the cliff. Fixtures that certify exactly the cases a mechanism
already handles prove nothing about the boundary.

## 2. Soak time

**No contract element is consumed externally in the milestone it is minted.**
The audit found `RoundingContext` in a pinned external consumer while its
semantics (significant digits, not decimal places) had never been exercised
by a single calculation — and were wrong for their stated purpose. Decision
velocity is an asset only while it stays ahead of the blast radius. One
milestone of internal use between minting and external consumption converts
"we believe this shape is right" into "something has used this shape."

## 3. Decision tiering

Two classes of decision, explicitly. **Class A** (RFC-gated, full ceremony):
domain semantics, canonical types, published contracts, Tier-0 ecosystem
rules — anything the ledger's meaning or an external consumer depends on.
**Class B** (one-page record): operational choices — a toolchain patch
policy, an editor configuration, an issue-tracking convention. The corpus
currently has one tier; ADR-0038 spent 231 lines on a Go patch-line policy,
and the ceremony gradient is a measured cause of the 1-domain-in-9 ADR
allocation and a hard barrier to any second contributor. The append-only
rule (ADR-0000) applies to both classes; only the required depth differs.

## 4. Every milestone answers a question

Each milestone's exit criterion is phrased as **a question a user can ask
the system that they could not ask before**. Sixteen milestones produced
zero askable questions — no read surface exists, and six of seven ledger use
cases have no production caller. Infrastructure milestones remain legitimate,
but they earn their place by naming the question they unblock. The proposed
continuation in [Roadmap.md](Roadmap.md) applies this rule to every row.

## 5. Decided-but-unshipped is a marked state

An accepted ADR whose named code change does not exist is a standing hazard:
ADR-0028 states a field rename as decided that is absent from the tree, and
ADR-0011's upcasters — load-bearing for storage — were never built. Because
accepted ADRs are immutable, the corpus needs either an
`Accepted (unimplemented)` status variant assigned at acceptance time, or a
mechanical check that an ADR's enforcement-table rows cite mechanisms that
exist. Silence between "decided" and "shipped" is where the audit found the
most dangerous gaps.

## 6. Enforcement checks validate claims, not shape

The constitution-coverage check verifies the §15 table lists every principle
— not that any cited mechanism holds its claimed rung; at least four rows
overstate. The directory-contract check verifies front matter is present —
not that a module's real imports match its declared contract, while the
script's own header claims they generate the import boundary. **Extend every
shape check one level: a §15 row citing a target must find that target a
prerequisite of the verify gate; a claim of generation must find a
generator.** Where the claim cannot be validated, the claim is rewritten to
what the mechanism actually does — the Constitution explicitly sanctions
downward correction, and the honest register already exists in the proto
comments ("an affordance for honesty, not a control").

## 7. Allocation: at most one meta milestone per three domain milestones

65% of the ADR corpus is meta-engineering; of the last nine ADRs, one
concerns domain semantics; `Portfolio` and `Market Data` exist only as
analyser test fixtures. The governance apparatus is excellent and is not the
product. The ratio is a budget, reviewed at each milestone boundary, and the
burden of proof sits with the meta milestone — it must name the domain work
it unblocks.

## 8. Preserve the honesty culture — it is the moat

Measurement before decision (ADR-0036 opens with a reproducible benchmark);
recorded self-correction ("measured, after asserting the opposite twice");
declined-rather-than-dismissed alternatives (ADR-0036's alternative E is
written up as the probably-correct long-term answer, with its adoption
trigger); costs stated inside the decision that accepts them. None of the
audit's findings would have been visible in an ordinary repository, because
an ordinary repository would never have written down enough to be checked.
Every principle above tightens the gap between claim and mechanism; none of
them is worth adopting in a way that makes people stop writing down what is
actually true.
