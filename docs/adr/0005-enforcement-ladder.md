---
id: ADR-0005
title: Every architectural principle is enforced at the highest feasible mechanism
status: Accepted
date: 2026-08-04
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0005 — Every architectural principle is enforced at the highest feasible mechanism

## Context

The FDOS Constitution states fourteen principles. Every one of them is currently
prose, and prose does not survive contact with a decade of maintenance, staff
turnover and delivery pressure. Principles decay silently: nothing reports the
moment "the domain must not depend on infrastructure" stops being true.

The naive remedy is to convert each principle into a CI gate. This is too
narrow. A CI gate fails late — after the code is written, pushed and waiting for
review — and it is the wrong mechanism for a constraint the type system could
have made unrepresentable. Some guarantees belong to types, some to static
analysis, some to CI, some to review, and only the residue to documentation.

## Decision

FDOS enforces every architectural principle at the **highest feasible rung** of
this ladder:

| Rung | Mechanism | Fails at |
|------|-----------|----------|
| 1 | Type system — the violation cannot be expressed | authoring time |
| 2 | Static analysis — compiler pass, custom analyser, import boundary | build |
| 3 | CI — test, fitness function, reproducibility gate | pull request |
| 4 | Automated review — agent-assisted review with defined contracts | review |
| 5 | Documentation — convention, playbook, checklist | never, automatically |
| 6 | Human discipline | — |

Three obligations follow:

1. **Human discipline is the last line of defence, never the first.** A principle
   whose only enforcement is "we will remember" is recorded as unenforced.
2. **Every principle declares its current rung.** The Constitution (§15) carries
   the table. It is a self-assessment and is expected to be unflattering.
3. **Climbing is not optional.** When a mechanism one rung higher becomes
   feasible, adopting it becomes work, not a suggestion. Every ADR states the
   rung it lands on and the rung it targets.

## Consequences

### Positive

- Architectural erosion becomes measurable. "Which principles are unenforced?"
  has an answer that can be tracked over time.
- Enforcement effort is directed by leverage rather than by habit. Reaching for
  CI when a type would do is now visibly a compromise.
- New engineers and AI agents inherit the constraints structurally rather than
  by absorbing culture.

### Negative

- Rung 1 and rung 2 mechanisms are real engineering. A custom `go/analysis` pass
  is a maintained artifact with its own bugs, and false positives breed
  `//nolint` culture that silently returns the principle to rung 6.
- The ladder can be gamed. A weak type that technically sits at rung 1 while
  permitting the violation is worse than an honest rung 5, because it reports
  safety that is not there.
- Cost is front-loaded onto a project with no users, in service of a decade-long
  horizon that may not arrive.
- Over-enforcement is a real failure mode. A principle that turns out to be
  wrong is much harder to change once it is a compiler pass. Rung placement must
  follow confidence in the principle, not enthusiasm for the mechanism.

### Enforcement

Rung 3 (CI), self-referentially: `make verify` runs the fitness functions that
exist. The completeness of the §15 table is reviewed at every milestone
boundary. Nothing currently prevents that table from going stale — a check that
every Constitution principle appears in it is a candidate for M1.

## Alternatives considered

**"Every principle becomes a CI gate."** Rejected as the framing this ADR
replaces. It is a single-mechanism answer to a problem with a hierarchy of
mechanisms, and it systematically under-uses the type system, which is the only
rung that fails before the code is even written.

**Enforce nothing; rely on review and culture.** Rejected: this is rung 6 for
everything. It works at small scale and with stable staff, and both assumptions
fail over a decade.

**Enforce everything at rung 1.** Rejected as not feasible. Some principles —
"prefer deleting complexity to adding abstraction" — are judgements, not
predicates, and pretending otherwise produces ceremony rather than safety.
