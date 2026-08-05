---
name: pr-review
description: Review a pull request end to end including its governance obligations. Use when reviewing a pull request before merge, checking whether a structural change carries the required ADR, or validating that a change meets the FDOS definition of done
---

## Workflow

1. Read the description. What decision does this change embody?
2. **Governance gate:** does it change repository structure, module boundaries,
   the public contract surface, the toolchain or an enforcement mechanism? If
   so, it requires an ADR in the same pull request.
3. Apply the [code-review](../code-review/SKILL.md) skill to the diff.
4. Check the definition of done: `make verify` passes; structural change has an
   ADR; new checks have negative tests; Constitution §15 matches reality;
   documentation updated in this change, not later.
5. Check whether the change closes a gap listed at rung 6 — if so, §15 must be
   updated to record the climb.
6. Approve, request changes, or comment.

## Examples

**Request changes — missing governance.** The real review of what became
`scripts/verify-adr-immutability.sh`:
```
This adds a check comparing each ADR against its introducing commit and
wires it into `make verify`.

Two blockers:

1. New enforcement mechanism, no ADR. Enforcement mechanisms require one
   (ADR-0000). It should state the principle enforced and the ladder rung.

2. No negative test. Break the invariant deliberately, confirm the check
   fails with a useful message, restore. Two of the four M0 checks had real
   defects that only negative testing surfaced.

Also: docs/constitution.md §15 lists ADR immutability under §14 as review
only. This change moves it to rung 3 — the table should say so.
```
Both were addressed before it landed: ADR-0014 records the decision, and the
check was exercised against a rewritten body line and a deleted section.

**Approve:**
```
Approved.

ADR-0007 states the decision, the cost, and the alternative that lost.
Negative tests cover all four failure modes. §15 updated: §6 moves from
rung 6 to rung 1.

make verify passes locally on a clean clone.
```

## Quality Bar

- The governance gate is checked before the diff. A structurally significant
  change without an ADR is blocking regardless of code quality.
- Approve only after running `make verify`, or state explicitly that you did not.
- Required changes and suggestions are visually separated. Reviewers who blur
  them get their required changes ignored.
- A pull request that reverses an earlier decision must carry the superseding
  ADR — never a silent edit to the original.
- Backward compatibility of published contract modules is checked from M4 via
  `buf breaking`; until then it is a manual review obligation, and saying so is
  better than assuming coverage.

## Resource Strategy

- No `scripts/`, `references/` or `assets/`. The authoritative checklist is the
  definition of done in `.context/docs/development-workflow.md`; duplicating it
  here would create a second copy that drifts.
