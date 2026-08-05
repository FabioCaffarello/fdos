<!--
Delete any section that does not apply. An empty section is noise; a missing
one that mattered is a review failure.

`make verify` is the whole gate and CI runs exactly it (ADR-0014). This
template covers what the gate cannot check.
-->

## What this changes, and why

<!-- The diff says what. Say why, and what it costs. -->

## Governance

<!--
An ADR is required for any change to: repository structure or directory
contracts, module boundaries or the public contract surface, the toolchain or
pinned versions, enforcement mechanisms, or the Constitution itself.

Tick what applies, or state plainly that none does.
-->

- [ ] No ADR needed — this changes none of the above
- [ ] ADR included: <!-- ADR-NNNN -->
- [ ] Reverses an earlier decision, by **supersession** — the superseded ADR
      keeps its text, gains `status: Superseded` and `superseded_by`
- [ ] RFC included, because the design needed exploration first

## Enforcement ladder

<!--
Every principle is enforced at the highest feasible mechanism (ADR-0005):
type system > static analysis > CI > automated review > documentation >
human discipline.
-->

- [ ] Does not change where any principle sits
- [ ] Moves a principle up a rung — Constitution §15 updated to say so
- [ ] Moves a principle **down**, or shows an earlier target was not
      achievable — §15 corrected downward, with the reason

If this adds an enforcement mechanism:

- [ ] It was **negative-tested**: the invariant was broken deliberately, the
      check failed with a useful message, and the working tree was restored and
      verified

> A check that has never gone red is unverified.

## Contract surface

- [ ] Does not touch `libs/contracts/proto/`
- [ ] `make proto-check` passes — no wire- or JSON-incompatible change
- [ ] Generated code regenerated with `make proto-gen`, not hand-edited

## Verification

```
make verify
```

- [ ] Passes locally
- [ ] Could not run it — stated here, plainly, rather than implied

<!--
State what you actually ran. "Should be fine" is not a verification, and a
claimed check that was not run is worse than an unchecked change.
-->

## Documentation

- [ ] Updated **in this change**, not deferred
- [ ] `.context/` still describes the repository that exists
- [ ] No paragraph left stale by this change

<!--
`make context-check` catches a name that no longer exists. It cannot catch a
paragraph that is merely wrong — three such statements survived every check
until M3.5 audited them by hand.
-->

## Blocked or deferred

<!--
Anything this change leaves incomplete, and why. Silence reads as coverage.
See docs/blocked.md for work blocked on something outside this repository.
-->
