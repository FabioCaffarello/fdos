---
id: ADR-0051
title: A tag pushed by a workflow triggers nothing, so the release is dispatchable
status: Accepted
date: 2026-08-13
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0051 — A tag pushed by a workflow triggers nothing, so the release is dispatchable

## Context

[ADR-0046](0046-publishing-a-tag-is-a-dispatched-act.md) built a dispatched
publication path: a human dispatches `release-tag.yml`, it checks six
preconditions and pushes the tag, and `release.yml` picks the tag up and builds,
signs and publishes. Its own output said so — *"That creates the tag and pushes
it, which starts release.yml."*

**It does not.** GitHub suppresses workflow runs for events created with
`GITHUB_TOKEN`, to stop workflows triggering themselves. A tag pushed by a
workflow is such an event.

Measured, on two tags a few hours apart:

| Tag | Pushed by | `release.yml` triggered |
|---|---|---|
| `libs/kernel-wire/v0.3.0` | a maintainer's own credentials, locally | **yes** |
| `libs/ledger-wire/v0.5.0` | `release-tag.yml`, via `GITHUB_TOKEN` | **no run at all** |

So the path ADR-0046 built creates a tag that nothing builds — and the local
path it was meant to replace works. The end state is B-008's again: a published
tag with no release, arrived at from the opposite direction.

This was not findable by reading either workflow. It is a property of the token,
documented by GitHub and invisible in the repository.

## Decision

### 1. `release.yml` is dispatchable against an existing tag

It keeps its `push` trigger and gains `workflow_dispatch` with a `tag` input.
The job is the same either way — one entry point, two ways in. `github.ref_name`
is the tag on a push and the branch on a dispatch, so every step names the tag
explicitly through `inputs.tag || github.ref_name` rather than inferring it.

### 2. Publication is two dispatches, and the second is said out loud

`release-tag.yml` now prints the exact command to run next, because a tag with
nothing building it is precisely the state nobody notices:

```
Tag pushed. Nothing is building it: a tag created with
GITHUB_TOKEN does not trigger release.yml.

Publish the evidence:

  gh workflow run release.yml -f tag=libs/ledger-wire/v0.5.0
```

Two human acts rather than one. That is consistent with ADR-0046 rather than a
retreat from it: the decision that publishing is a human act was the point, and
the second act — *publish the evidence* — is a real decision that was previously
implicit.

### 3. What this does not decide

**No `workflow_call`.** Making `release.yml` a reusable workflow invoked from
`release-tag.yml` would keep it to one dispatch, and it is the obvious
alternative. It is not taken here — see below.

**No personal access token.** A PAT would make the push trigger workflows and is
exactly the credential ADR-0014 spent its argument avoiding.

## Consequences

### Positive

- A release can be completed for any existing tag, including the two that were
  stranded — `libs/ledger-wire/v0.5.0` was published by dispatching it.
- The failure mode is loud: `release-tag.yml` tells you the tag is inert and
  gives the command.
- The local path and the dispatched path now converge on the same workflow
  rather than depending on which one you used.

### Negative

- **Two steps where the design promised one**, and the second can be forgotten.
  Nothing enforces it; the printed command is rung 5.
- **A tag can still sit unbuilt**, which is B-008's shape. This makes it
  recoverable rather than impossible — the earlier `libs/kernel-wire/v0.3.0` is
  not recoverable, for an unrelated reason (#125).
- **`inputs.tag || github.ref_name` appears in four places.** A fifth step added
  without it would silently build the wrong ref on a dispatch.
- **This is the fourth defect found by performing a release** rather than by
  reasoning about one, in a path whose ADR said nothing else would find them.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A release can be built for any tag | 3 | `release.yml` `workflow_dispatch` |
| The second dispatch is not forgotten | 5 | `release-tag.yml` prints the command |
| The right ref is built | 6 | review of `inputs.tag \|\| github.ref_name` in every step |

## Alternatives considered

**`workflow_call`, invoking `release.yml` from `release-tag.yml`.** One
dispatch, no forgotten second step, and no event is involved so the guard does
not apply. Rejected for now on evidence rather than principle: this is the fourth
consecutive defect in this path, three of them from adding indirection, and
`workflow_call` changes permission inheritance and the `github` context in ways
that would need their own release to discover. The simpler shape is what this
path has earned. It remains the right answer once the path has published
something without incident.

**A personal access token, so the push triggers workflows.** Restores the
original design exactly. Rejected: ADR-0014's whole argument is about not
granting the build a credential nobody reviews, and this would be one.

**Merge the two workflows.** Tag and release in one job — no trigger, no guard,
no second step. Rejected because it puts `contents: write` in the same job as the
build and signing, where ADR-0046 deliberately confined it to the smallest
possible surface.

**Go back to tagging locally.** It demonstrably works — `libs/kernel-wire/v0.3.0`
triggered the release. Rejected: the six preconditions and the audit trail are
what the dispatched path is for, and "it works from a laptop" is what the path
was built to stop relying on.

## Notes

- The recursion guard is documented by GitHub and was still a surprise. Nothing
  in this repository could have shown it; the two-row table above is the whole
  of the evidence and it took two releases to assemble.
