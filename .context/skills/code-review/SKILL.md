---
type: skill
name: Code Review
description: Review changes against the FDOS Constitution, accepted ADRs and directory contracts. Use when reviewing any change to this repository, checking whether a change violates an architectural decision, or deciding whether a convention should become an enforced mechanism
skillSlug: code-review
phases: [R, V]
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

## Workflow

1. Read `docs/constitution.md`, the ADRs touching the changed paths, and the
   `README.md` front matter of every directory in the diff.
2. Check for Constitution violations. Blocking; name the principle.
3. Check for contradiction of an accepted ADR. Blocking unless the change
   carries the superseding ADR.
4. Check each directory's `forbidden` list against what the change actually adds.
5. Ask whether the change pre-judges an open M1.5 RFC question.
6. Ask whether anything here could be enforced one rung higher.
7. Run `make verify`. If you cannot, say so rather than implying you did.

## Examples

**Blocking — Constitution §2, determinism:**
```
libs/domain/pricing/accrual.go:42 uses float64 for an interest amount.

Binary floating point is not reproducible and silently corrupts money.
Constitution §2 requires every calculation to be reproducible; §9 requires
a 2031 regeneration to be byte-identical.

Use a decimal type or integer minor units. This will become a build error
when the M2 analyser lands — the rule exists, the enforcement does not yet.
```

**Blocking — pre-judges an open RFC:**
```
This adds libs/domain/go.mod, which fixes the layer structure.

Layering (domain / app / adapters) is an output of the M1.5 RFCs and is
explicitly recorded as a hypothesis in .context/docs/architecture.md.
Creating the module decides it by accident.

Hold until the RFC lands, or write the RFC.
```

**Ladder observation, non-blocking:**
```
The PR description says "remember not to import adapters from domain".

That is rung 6. depguard can express this today from the libs/ README
`forbidden` list. Worth an issue for M2 rather than a comment on this PR.
```

## Quality Bar

- Findings ordered by ladder rung, not by file order. A principle violation
  outranks every style comment below it.
- Every claimed defect states the concrete failure: input, state, wrong result.
  A defect you cannot make fail is a hypothesis — label it as one.
- Distinguish "violates a decision", "is a defect", and "I would have done this
  differently". The third carries no authority; mark it as preference.
- New enforcement mechanism with no negative test is a blocking finding. A check
  that has never gone red is unverified.
- Documentation left stale by the same change is not follow-up work; it is an
  incomplete change.

## Resource Strategy

- `scripts/` only when a review step is mechanical enough to be a fitness
  function — at which point it belongs in `scripts/` at the repository root
  instead, wired into `make verify`.
- `references/` not needed; the authoritative material is `docs/`, and copying
  it here would create a second version that drifts.
