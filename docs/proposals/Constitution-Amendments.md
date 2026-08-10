---
title: Candidate constitutional amendments
status: Provisional — proposal from the 2026-08-07 architectural audit
date: 2026-08-07
---

# Candidate constitutional amendments

> **Status: provisional.** [`docs/constitution.md`](../constitution.md) v1.0.0
> is the ratified authority and changes only through its own amendment
> procedure: RFC, version bump, ADR. This document nominates candidates for
> that procedure. It amends nothing by itself.

The audit found the Constitution's fourteen principles sound and does not
propose weakening any of them. All three candidates strengthen the §15
machinery that makes the principles enforceable — because the audit's central
mechanical finding was that §15's *claims* are validated only in *shape*.

## A1 — Rung claims must be claim-validated, not shape-validated (minor)

§15 requires the table to list every principle; nothing requires a cited
mechanism to hold its claimed rung. The audit found four rows overstating
(rows 2, 3 and 10 rest on syntactic, bypassable analysers; row 13 cites a
check that deliberately does not run per pull request). Proposed addition to
§15: *a mechanism cited in the table must be a prerequisite of the verify
gate, and every rung-1/rung-2 claim must carry a negative test exercising an
indirect form of the violation, not only the direct form.* The existing
"downward correction is sanctioned" clause already permits the honest
alternative — downgrading the row — so this amendment forces a choice rather
than an outcome.

## B2 — Decided-but-unshipped is a visible state (minor)

ADR-0028 records a contract rename as decided; the tree does not contain it.
ADR-0011 decides upcast-on-read; no upcaster exists. Append-only immutability
(correct, keep it) means neither ADR can be annotated in place, so the corpus
silently asserts a repository that does not exist. Proposed: a recognised
status marking — `Accepted (unimplemented)` in front matter, or an ADR-side
register — plus the obligation that acceptance of an implementing change
clears it. This is an amendment to §14/§15 practice, not to any principle.

## C3 — The read side gets E9's standing (ecosystem invariant, Tier-0 procedure)

Not strictly a constitutional amendment — invariants live in
[`docs/ecosystem/invariants.md`](../ecosystem/invariants.md) and amend by the
Tier-0 procedure — but nominated here because it is the same class of
self-honesty: E9 declares the open core "a demonstration rather than a
platform" if data cannot get in without the private repository. There is no
counterpart for getting an answer out, and after sixteen milestones there is
no read surface. Proposed E-series invariant: *the open core must be able to
answer a question — a value derivable from admitted facts, at an explicit
as-of coordinate, with its derivation — using only public code.* By the same
standard E9 sets, until this holds the platform is a demonstration.

## Explicitly not proposed

- No new principle for the knowledge graph, AI, or MCP: §2, §8 and §12 already
  cover them; the gaps there are sequencing, not principle.
- No relaxation of §7's universal bitemporality for batches. The batch
  knowledge-time question (one statement is one epistemic event) is real, but
  it is an RFC about the *meaning of knowledge time under bulk arrival* —
  ADR-0036's declined alternative E is the starting point — not a
  constitutional exception.
