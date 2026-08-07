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

If this changes behaviour an existing test guards:

- [ ] That test was **mutation-checked**: the new behaviour was reverted or
      disabled, the guarding test went red *for the stated reason*, and the
      change was restored and verified

> A check that has never gone red is unverified — and one that goes red for a
> different reason than you think is worse, because it will keep passing when
> the thing it names actually breaks.

<!--
Both of the above want the same evidence and neither is satisfied by "tests
pass". Paste the failure message. In M9 a mutation showed `MintIdentity`
returning `nil` and recording a duplicate mint; in M10 one showed a store
answering `ErrStaleRead` and appending anyway. Neither was visible from a green
run.
-->

## If a session opened this pull request

<!--
`gh pr create --body …` replaces this template rather than appending to it, so
a PR opened that way never sees the checklists above. Four PRs in M10 were
opened that way and none carried them.
-->

- [ ] Opened by a person, from this template
- [ ] Opened by a session — the checklists above were read and carried into the
      body deliberately, not skipped by the tool

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
Blocked work is registered as GitHub issues under the `status/blocked` label
(ADR-0032); docs/blocked.md is the frozen index of B-001 … B-009.
-->
