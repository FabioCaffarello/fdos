# Branch protection and merge queue

These are GitHub repository settings, not files. They cannot be enforced from
this repository, which makes them the weakest link in the M3 gate: everything
else is a mechanism, and this is a checklist.

Recorded here so the gap is visible and the intended configuration is
reconstructible. Raising it above documentation requires committing a repository
ruleset as JSON and applying it from a workflow with an admin token — which
introduces an admin-scoped credential into CI, a worse risk than the one it
solves. Deliberately not done (ADR-0014).

## Required settings for `main`

### Protection

| Setting | Value | Why |
|---------|-------|-----|
| Require a pull request before merging | on | The ADR process is a review process; direct pushes bypass it |
| Required approvals | 1 | Minimum meaningful review |
| Dismiss stale approvals on new commits | on | An approval describes a diff, not a branch |
| Require conversation resolution | on | An unresolved blocking finding must not merge |
| Require linear history | on | The bisect that finds a reproducibility regression years later needs it |
| Require signed commits | on | Authorship is part of provenance (Constitution §6) |
| Allow force pushes | **off** | History is never rewritten (Constitution §4, ADR-0000) |
| Allow deletions | **off** | — |

### Required status checks

Require branches to be up to date before merging, and require:

```
verify
```

That is the whole list, and deliberately so. `verify` runs `make verify`, which
runs every mechanism the repository has (ADR-0014). Enumerating individual
checks here would create a second place the gate is defined, and the two would
drift.

`supply-chain / dependency-review` is **not** required: it only runs on pull
requests that change dependencies, and a required check that does not always run
blocks merges for the wrong reason.

### Merge queue

Enable, with:

| Setting | Value |
|---------|-------|
| Merge method | Squash |
| Build concurrency | 2 |
| Only merge if the combined status is green | on |

The queue matters more here than in most repositories: with one module per
bounded context (ADR-0004), two pull requests can each pass on their own branch
and fail once merged, because module resolution with `GOWORK=off` sees the other
one's published state.

## Tag protection

Protect `libs/*/v*` — release tags trigger the signing and provenance workflow.
A tag that can be moved makes every attestation pointing at it meaningless.

## Verification

There is no automated check. To audit:

```sh
gh api repos/FabioCaffarello/financial-data-operating-system/rulesets
gh api repos/FabioCaffarello/financial-data-operating-system/branches/main/protection
```

If the output disagrees with this document, one of the two is wrong — and the
repository settings are what actually gate merges, so assume the document is
stale and fix it.
