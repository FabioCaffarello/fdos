---
type: skill
name: Feature Breakdown
description: Break FDOS work into ADR- and RFC-shaped units within the milestone roadmap. Use when planning a milestone, decomposing a large change, or deciding whether work needs an RFC before it can start
skillSlug: feature-breakdown
phases: [P]
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

## Workflow

1. Locate the work on the roadmap. If it belongs to a later milestone, say so
   and stop — sequencing is a decision, not an obstacle.
2. Ask what is **undecided**. Undecided design becomes an RFC before any task
   list exists.
3. Ask what the work would **foreclose**. Anything that silently settles an open
   RFC question is blocked, however small.
4. Decompose into units, each producing an ADR, an RFC, or a mechanism.
5. State dependencies, and for each unit its position on the enforcement ladder.
6. Give each unit acceptance criteria that can fail.
7. Name the risks, especially the ones that make the whole approach wrong.

## Examples

**Correct shape — design before tasks:**
```
## M1.5: Reference Data & Temporal Reproducibility

Undecided, therefore RFC first. No tasks until it lands.

RFC-0003 must settle:
  - Does every canonical event carry a reference-dataset version?
  - Universal, or scoped per aggregate type?
  - How is a dataset version identified and stored?

Risk, and the reason this is M1.5 and not M4: not retrofittable. If the
model does not carry a reference-data version from the first event,
historical reproducibility is permanently lost.

Acceptance: RFC accepted, and ADRs recording what it settled.
```

**Correct refusal:**
```
Request: "add a Position struct to libs/domain so we can start."

Blocked. This decides aggregate boundaries, identifier strategy and
bitemporal scope by accident — all open M1.5 RFC questions.

The layering in .context/docs/architecture.md is explicitly marked a
hypothesis. Writing the struct converts it into a decision without an ADR.

Do the RFC. It is the shorter path.
```

**Unit with a real acceptance criterion:**
```
### Unit: ban binary floating point in domain packages

Mechanism: go/analysis pass; ladder rung 2 (was 6).
Depends on: M1.5 layering RFC (needs to know which packages are "domain").

Acceptance: a PR adding `var x float64` to a domain package fails CI,
citing the rule by name. Verified by negative test, not by inspection.
```

## Quality Bar

- Every unit produces an ADR, an RFC, or a mechanism. A unit producing only code
  in a repository with no decided model is a warning sign.
- Acceptance criteria must be falsifiable. "Documentation improved" is not one;
  "a PR doing X fails CI with message Y" is.
- Each unit names its ladder rung, current and target.
- Dependencies stated explicitly, including on unaccepted RFCs.
- Risks include what would make the whole approach wrong, not only what could
  slow it down.
- Work belonging to a later milestone is named and deferred, not quietly pulled
  forward.

## Resource Strategy

- None. The roadmap lives in `README.md` and `.context/docs/project-overview.md`;
  a copy here would drift from both.
