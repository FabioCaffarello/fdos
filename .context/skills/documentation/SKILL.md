---
type: skill
name: Documentation
description: Write and update FDOS documentation — docs/, directory contracts and .context/. Use when documenting a decision, updating a directory contract, marking provisional design intent, or keeping .context/ in step with docs/
skillSlug: documentation
phases: [P, C]
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

## Workflow

1. Identify the layer: Constitution, ADR/RFC, directory contract, `.context/`,
   or root README. Authority runs in that order.
2. Check whether the content already exists higher up. If so, reference it —
   never copy it.
3. Write it, marking anything not yet decided as provisional.
4. State gaps explicitly, including which milestone closes them.
5. Update `.context/` if `docs/` changed. The derivation is manual today.
6. Run `make verify` — directory contracts and the §15 table are checked.

## Examples

**Marking provisional content:**
```markdown
> **Nothing described here is implemented.** There is no connector, parser,
> ledger or projection in this repository. This documents the flow the
> architecture is designed to permit. The binding version is the M1.5 RFC set.
```

Without that marker, intent gets implemented as though it were decided.

**Directory contract front matter:**
```yaml
---
directory: libs          # must match the actual path
purpose: Reusable FDOS libraries. Each subdirectory is an independent Go module.
owner: "@FabioCaffarello" # must agree with CODEOWNERS
allowed:
  - Independent Go modules, one per subdirectory (ADR-0004)
forbidden:
  - Executable entry points (main packages belong in apps/)
---
```

`make contracts-check` rejects an empty `allowed` or `forbidden`: a contract
permitting or forbidding nothing is not a contract.

**Stating a gap rather than softening it:**
```markdown
| Gap | Closes at |
|-----|-----------|
| No secret scanning | M3 |
| No enforcement of the LLM boundary | M4 |

At M1 the security posture rests almost entirely on review.
```

## Quality Bar

- Never duplicate a decision. A copy drifts from the ADR, and the copy is what
  people read.
- Mark provisional content explicitly. Unmarked intent becomes accidental
  specification.
- State what does not exist. FDOS is mostly empty by design; silence reads as
  coverage.
- Update in the same change. Documentation is production code (§14), not
  follow-up work.
- Prefer a table to a paragraph when the content is a mapping.
- State costs beside benefits.

## Resource Strategy

- No `references/`: `docs/` is authoritative and copying it here guarantees
  drift. Link instead.
- `scripts/` only if a documentation invariant becomes mechanically checkable —
  in which case it belongs in the repository's `scripts/`, wired into
  `make verify`, not inside this skill.
