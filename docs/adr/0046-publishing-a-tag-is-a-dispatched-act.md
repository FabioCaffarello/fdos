---
id: ADR-0046
title: The release chain is planned by a command and published by a dispatched act
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes:
  - ADR-0045
superseded_by: []
---

# ADR-0046 — The release chain is planned by a command and published by a dispatched act

> **Supersedes [ADR-0045](0045-the-affected-graph-is-the-release-graph.md).**
> Not because it was wrong in substance — everything it decided is restated
> below — but because one of its rules, stated as written, made the release it
> was meant to support impossible to perform. ADR-0000 permits no edit that
> changes what a decision says, so this replaces it rather than patching it.
> [#103](https://github.com/FabioCaffarello/fdos/issues/103) said from the start
> that the plan and the publication were "two halves of one workflow"; they are
> one decision here.

## Context

### What ADR-0045 established, and which still holds

The affected graph and the release graph are the same graph: "which modules did
this change affect" and "which modules now need a tag" are one traversal asked
twice. `make release-plan` computes it, `make registry-check` holds the contract
registry to the tags, and a non-required `preflight` job runs the affected
modules for fast failure without becoming a narrower gate. None of that changes.

### The rule that did not survive contact with a release

ADR-0045 stated `registry-check` rule G1 as *"every module row names its
module's newest tag"*. Measured against the ritual it was supposed to protect,
that is unperformable.

The registry update belongs in the commit that gets tagged — so that whoever
checks out `libs/kernel/v0.10.0` reads a table describing `v0.10.0`. During that
pull request the row names a version no tag has yet, and G1 as written fails it:

```
$ ./scripts/verify-registry.sh
  libs/kernel: registry says v0.10.0, newest tag is v0.9.0
FAIL: 1 registry violation(s).
```

The alternative order — tag first, update the registry afterwards — fails on
`main` for the whole window between, which is **the exact property
[ADR-0044](0044-the-gate-compiles-the-tree-as-a-workspace.md) refused for
`pin-check` R4**: a gate that goes red the instant a module is tagged. ADR-0045
declined it there and reintroduced it here, in a different rule, four hours
later.

### The publication half was never decided

`CONTRIBUTING.md` has said since M3 that *"manual tagging is not an acceptable
fallback: cross-module version chains are too easy to get wrong by hand"*, and
manual tagging has been the process for thirty-six tags. ADR-0045 made the
*planning* mechanical and left the act itself unaddressed.

Two facts make that act worth deciding rather than leaving to habit:

- **A release tag is immutable by ruleset**
  ([ADR-0043](0043-downloaded-artifacts-are-pinned-by-digest.md)). It cannot be
  moved or deleted, so a tag on a commit that cannot release is permanent
  garbage — and that is not hypothetical. B-008 is fourteen such tags, still in
  the namespace, describing releases that were never published.
- **The tag is what `release.yml` signs and attests against.** Tagging a red
  commit produces a signed statement about code that does not pass.

## Decision

### 1. G1 applies only to modules with nothing unreleased

A module whose source matches its newest tag must have a row naming that tag. A
module with unreleased changes may name a version **above** it — declaring the
release in flight — and may never name one below.

This is the same split as `pin-check` R3/R4 and for the same reason: check what
is settled, and let a release in progress say where it is going. The comparison
is tag-to-working-tree rather than tag-to-`HEAD`, so that an author editing a
module is told the truth about it locally as well as in CI.

### 2. Publishing is a dispatched act, and the machine takes the mechanics

`make release-tag MODULE=… VERSION=…` refuses unless all six hold:

| | Precondition |
|---|---|
| 1 | the version is well formed and above the module's newest tag |
| 2 | the tag does not already exist, locally or on the remote |
| 3 | the module actually has unreleased changes |
| 4 | the registry already declares this version |
| 5 | the tree is clean, on `main`, and identical to `origin/main` |
| 6 | **`verify` is green for this exact commit** |

Rule 6 is the one B-008 would have wanted. Rules 1 to 5 are the ones a person
gets wrong at the end of a long day, and rule 5 is written down in
`CONTRIBUTING.md` precisely because it has already cost rework.

**A dry run is the default and publishing requires `PUBLISH=1`.** The dangerous
path costs a word; the safe path costs nothing.

### 3. The button is a person's, and the workflow holds no logic

`release-tag.yml` is `workflow_dispatch` only, and its `publish` input must be
typed as the string `yes` — spelled out rather than a checkbox so it cannot be
left ticked from last time. It runs `make release-tag`, which is the same
command a maintainer runs locally, so refusing there and refusing here are the
same refusal (ADR-0014).

**Why dispatched rather than automatic.** Keyless signing binds the artifact's
identity to the workflow. A tag pushed without a human choosing to publish
produces a signed statement nobody decided to make. The human keeps *whether*
and *which version* — a judgement about compatibility no script can make — and
the machine takes the chain order and the six preconditions, which is where the
errors are.

### 4. Write permission is scoped to that one job

`contents: write` appears in exactly one job in this repository, behind a
`release` environment whose deployment branch policy restricts it to `main`.

**Required reviewers are deliberately not set**, for the reason
`docs/branch-protection.md` already records for required approvals: a single
maintainer cannot approve their own dispatch, and a protection rule that must
always be bypassed trains the one person who can bypass it to reach for the
override. It rises the day there is a second maintainer.

### 5. `release-prepare` does one mechanical edit and no more

It sets the module's registry row to the version about to be released, and opens
a pull request. It does not choose the version, bump pins, or write the paragraph
saying what the release carries — pins are already held current by `pin-check`
R3 for any module with unreleased changes, and the paragraph is the part a reader
actually needs.

### 6. What this does not decide

**Applications are still not released.** `release-tag` refuses `apps/*` and
`examples/*` outright and names ADR-0039 as the decision that would change that.

**Nothing merges anything.** `main` is reached through a pull request, as
ADR-0020 requires.

**No version is chosen by a machine**, here or anywhere.

## Consequences

### Positive

- The release ritual is performable again, and `registry-check` protects it
  instead of forbidding it.
- A tag cannot be pushed onto a red commit, onto a dirty tree, onto a module
  with nothing to release, or onto a commit whose registry does not describe it.
- `CONTRIBUTING.md`'s claim about manual tagging is true for the mechanical half
  for the first time.
- The tagged commit describes itself: checking out `libs/x/vN` yields a registry
  saying `vN`.

### Negative

- **This adds a second consumer to the release path with the worst record in the
  repository.** B-008 shipped fourteen empty releases; `release-tag` does not
  touch `release.yml`, but it is now what starts it, and a bad tag is permanent.
- **`contents: write` exists in this repository now, where it did not.** It is
  one job, environment-scoped and branch-restricted, and it is still a real
  increase in what a compromised workflow could do.
- **The `release` environment is repository state nobody diffs**, the same gap
  `docs/branch-protection.md` records for rulesets. It is documented there and
  checked by nothing.
- **Rule 6 depends on `gh` and on GitHub's check-run API.** Without `gh` the
  script says so and refuses to publish rather than pretending; that is a
  refusal a maintainer offline cannot argue with, and it is deliberate.
- **Superseding a one-day-old ADR to change one rule is expensive**, and it
  restates a page of decisions that never changed. That is what ADR-0000's
  immutability costs, and paying it visibly is better than editing the record.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A settled module's registry row names its newest tag | 3 | `make registry-check` G1, negative-tested |
| A row may run ahead only while a release is in flight | 3 | same rule, negative-tested both ways |
| Every published module is described | 3 | `make registry-check` G2 |
| The corpus row names the newest `ecosystem` tag | 3 | `make registry-check` G3 |
| Every published contracts version is described | 3 | `make registry-check` G4 |
| A tag is never pushed onto a red or dirty commit | 3 | `make release-tag`, six preconditions, exercised as dry runs |
| Publishing is a human act | 3 | `workflow_dispatch` with an explicit `yes`; no other trigger writes a tag |
| Write permission is confined | 3 | one job, `release` environment, branch policy `main` |
| The release chain is known before it is performed | 5 | `make release-plan`, asked for by the pull-request template |
| Affected modules fail fast | 4 | the `preflight` job — advisory, and cannot fail alone |
| `release.yml` itself works | 6 | performing a release. Unchanged from ADR-0039's assessment |

The last row is inherited and not improved here. It is what
[#105](https://github.com/FabioCaffarello/fdos/issues/105) is for.

## Alternatives considered

**Keep G1 as written and change the ritual instead.** Every ordering was
considered: tag-then-registry reddens `main` for the window; registry-then-tag
fails the pull request; a workflow that merges and tags in one step still fails
the pull request first. There is no order that satisfies G1 as written, which is
what makes this a superseding decision rather than a preference.

**Push the tag automatically when a release pull request merges.** Removes the
button, and the merge is already a human act. Rejected: the merge is a decision
about code, not a decision to publish. Bundling them means a reviewer approving a
diff also signs an artifact, and the signature says the workflow decided.

**Let `release-plan` push tags directly.** One command for the whole chain.
Rejected as ADR-0045 already recorded: a planning command that publishes makes
the plan a publication by accident.

**Skip rule 6 and rely on `release.yml` re-verifying.** It does re-verify, so a
red commit would fail the release rather than publish it. Rejected because the
tag is immutable: the failure leaves a permanent tag for a release that never
happened, which is exactly the state B-008 left behind fourteen times.

**Require a reviewer on the `release` environment.** Stronger, and standard.
Rejected on the evidence in `docs/branch-protection.md`: signed commits were
required and removed the same day because a single maintainer could not satisfy
the rule, and the lesson recorded there is that a rule which must always be
bypassed is worse than no rule.

## Notes

- The three preconditions most likely to fire in practice were exercised as dry
  runs while writing this: a version that does not move forward, a module with
  nothing to release, and a dirty working tree. Each refused with the reason.
- `libs/analysis` still has no tag, and `release-tag` would happily create the
  first one. Whether `fdoslint` should be released as a module remains
  unaddressed, as it was in ADR-0044 and ADR-0045.
