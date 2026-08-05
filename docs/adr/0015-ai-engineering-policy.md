---
id: ADR-0015
title: AI-assisted work is bounded by enforced gates and a checkable knowledge base
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0015 — AI-assisted work is bounded by enforced gates and a checkable knowledge base

## Context

FDOS is developed with substantial AI assistance. That is a decision about how
the work gets done, and like every other such decision it needs a mechanism
rather than an intention.

The roadmap placed this milestone **after** M3 deliberately. At M2 the
determinism rules existed as `make` targets that depended on someone
remembering to run them. Enabling agents to generate code under that regime
would have meant accepting automated contribution under the weakest mechanism in
the repository. Since M3 the gate runs on every pull request without anyone
remembering, which is the precondition this milestone was waiting for.

Two failure modes remain, and neither is hypothetical.

**Knowledge drift.** `.context/` is derived from `docs/`. Nothing checked the
derivation. A playbook naming a `make` target that was renamed, or an ADR whose
status changed, is worse than no playbook: an agent acts on it confidently and
nothing reports that it was wrong. `.context/README.md` has recorded this as
rung 6 since M1.

**Playbooks without subjects.** The M1 scaffold generated fifteen agents,
including `database-specialist` for a system whose domain must not depend on a
database. Ten were removed. The pressure to add them back exists at every
milestone, and "we might need it later" is exactly how documentation stops
describing the repository.

## Decision

### AI output is bounded by the same gates as any other contribution

An agent's work is a contribution, not an authority. It passes `make verify` or
it does not land. No agent output is exempt, and no reviewer treats "an agent
wrote it" as evidence either for or against.

This is already true mechanically — the gate does not know who authored a diff —
and stating it prevents the drift towards a lighter path for automated changes.

### `.context/` is derived, and the derivation is checked

`make context-check` verifies that the knowledge base describes the repository
that exists:

- every relative link resolves
- every `make` target it names exists in the Makefile
- every `scripts/*.sh` it names exists
- every ADR and RFC it cites exists, and any status it claims matches
- no scaffold file remains `status: unfilled`
- the agent and skill rosters match what is on disk, in both directions

Drift becomes a failing build rather than a latent wrong answer.

### Playbooks declare a prompt contract

Every agent playbook declares, in front matter:

| Field | Meaning |
|-------|---------|
| `must_read` | Paths the agent reads before acting. Checked to exist |
| `must_not` | Actions that are out of bounds for this role |
| `evidence` | What the agent produces to show the work is done |

`make agent-contract-check` fails on a playbook missing any of these, or naming
a path that does not exist.

This is what "prompt contract" means here: not a template for phrasing, but the
subset of an agent's obligations that can be stated as data and verified.

### A playbook must have a subject

A role is added when the repository contains something for it to act on, and
removed when that thing goes away. `.context/README.md` records both directions
with reasons.

At M2.5 this admits `test-writer` (there are tests, with a specific discipline)
and `devops-specialist` (there are workflows, hooks and a supply chain). It
continues to exclude `api-design` until M4 produces contracts, and
`database-specialist` permanently, because a first-class database role invites
the coupling Constitution §10 forbids.

### The truth boundary is unchanged

Constitution §2 is not relaxed for engineering work. An agent may write code,
documentation and analysis. No agent output becomes a financial fact, and the
M4 type boundary applies to model output regardless of which model produced it
or why.

## Consequences

### Positive

- Knowledge drift becomes detectable. The most likely failure mode of an
  AI-assisted repository — confidently acting on stale context — now fails a
  build.
- Prompt contracts make an agent's obligations reviewable as data rather than
  buried in prose that nobody diffs.
- The roster stays honest, and the reason for each absence is recorded rather
  than rediscovered.

### Negative

- **The contract checks only what is checkable.** That an agent actually reads
  `must_read`, or honours `must_not`, cannot be verified from the outside. The
  fields make the obligation explicit and reviewable; they do not enforce it.
  Treating a green `agent-contract-check` as evidence of agent compliance would
  be exactly the false confidence this repository keeps trying to avoid.
- Two more checks in `make verify`, and every documentation change now risks
  failing on a reference that used to be prose.
- The staleness check is textual. It catches a renamed target and a deleted
  file; it cannot catch a paragraph that is merely wrong.
- Maintaining playbooks is ongoing work with no user-visible output, and it is
  the first thing that will be skipped under pressure.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| Knowledge base references resolve | 3 | `make context-check` |
| Playbooks declare a prompt contract | 3 | `make agent-contract-check` |
| Rosters match disk | 3 | `make agent-contract-check` |
| AI output passes the same gate | 3 | `make verify` — authorship-blind by construction |
| Agents honour their contracts | 6 | review |

The last row is the honest one. It is rung 6 and there is no credible path
above it, because the obligation is about behaviour rather than artifacts.

## Alternatives considered

**Generate `.context/` from `docs/` mechanically.** The stated M1 intent, and it
would make drift impossible rather than detectable. Rejected for now: the two
have genuinely different audiences — `docs/` argues a decision, `.context/`
tells an agent what to do about it — and a generator would either produce
unusable prose or require so much annotation in `docs/` that the annotation
becomes the second copy. Worth revisiting if the checks start failing often, as
that would be evidence the derivation is mechanical after all.

**No `.context/` at all; point agents at `docs/`.** Simpler, one source, no
drift by construction. Rejected: `docs/` is written to justify decisions to
humans and is the wrong shape for instruction. Agents given only it reliably
re-derive the wrong operational conclusions from correct principles.

**Restrict agents to non-architectural work.** Tempting, and it would cap the
blast radius. Rejected as unenforceable — the boundary between "implement this"
and "decide this" is exactly where the interesting failures live, and a rule
that cannot be checked is rung 6 wearing a policy costume. The ADR/RFC process
already governs decisions regardless of who proposes them.

**Require a human to author every ADR.** Rejected for the same reason: the
review gate is what matters, not the keyboard. An ADR is judged on whether its
negative-consequences section is honest, which is reviewable either way.

## Notes

Open, deliberately:

- Whether `.context/` should eventually be generated (see Alternatives). The
  trigger to revisit is the staleness check failing routinely.
- Whether prompt contracts should carry a `phases` obligation tied to the PREVC
  workflow, or whether that is ceremony.
- Nothing verifies that an agent's `must_read` list is *sufficient* — only that
  the paths exist. A playbook omitting `docs/constitution.md` would pass.
