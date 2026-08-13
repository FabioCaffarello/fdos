# Branch and tag protection

**Applied**, as two repository rulesets (ADR-0020) and one deployment
environment (ADR-0046). This document records what is configured and why, so a
change made in the GitHub UI can be recognised as a change.

It was previously a checklist with no mechanism. ADR-0014 declined to apply it
from CI because that needs an admin-scoped token — a worse risk than the one it
solves. Applying it by hand from an authenticated CLI does not need that token,
so the objection did not transfer.

## `main` — branch ruleset

Applies to the default branch.

| Rule | Why |
|------|-----|
| Pull request required | The ADR process is a review process; direct pushes bypass it |
| Required status check: `verify` | `make verify` is the whole gate (ADR-0014) |
| Strict status checks | The branch must be up to date, so the gate ran against what merges |
| Required linear history | The bisect that finds a reproducibility regression years later needs it |
| Conversation resolution required | An unresolved blocking finding must not merge |
| Squash merge only | One logical change per commit, matching the commit-message convention |
| No deletion | — |
| No force push | History is never rewritten (Constitution §4, ADR-0000) |

### Required approvals is 0

Deliberate, and the one setting that looks wrong.

A single-maintainer repository with one required approval cannot merge anything:
the author cannot approve their own pull request. Zero still requires the pull
request, the green status check and resolved conversations.

**It rises to 1 the day there is a second maintainer.** That is the only reason
it is not 1 now.

### Only `verify` is required

Deliberately the whole list. `verify` runs `make verify`, which runs every
mechanism the repository has. Enumerating individual checks here would create a
second place the gate is defined, and the two would drift.

`supply-chain / dependency-review` is **not** required: it only runs on pull
requests that change dependencies, and a required check that does not always run
blocks merges for the wrong reason.

## Merge settings — squash only, and now actually so

Applies to the repository, not to a ruleset, and committed as
`.github/rulesets/repository-merge.json` (ADR-0049).

| Setting | Value | Why |
|---|---|---|
| `allow_squash_merge` | true | one logical change per commit |
| `allow_merge_commit` | **false** | inert anyway under required linear history, and misleading while enabled |
| `allow_rebase_merge` | **false** | it lands the author's commits unchanged, with no ` (#NNN)` suffix — so the landed-subject rule was unpredictable |

**"Squash merge only" was documented above and was not true** until ADR-0049:
merge commits and rebase merges were both enabled, and five recent commits had
landed without a suffix as a result. `make ruleset-check` now covers these
settings.

## `release-tags` — tag ruleset

Applies to `refs/tags/libs/*/v*`, `refs/tags/apps/*/v*` and
`refs/tags/ecosystem/*` (ADR-0043).

It covered `libs/*` alone until the other two were checked against the live API
and found unprotected. `apps/*/v*` matters because ADR-0039 proposes attesting
build provenance against those tags. `ecosystem/*` mattered already: four such
tags existed, all movable, and `fdos-connectors` vendors the governance corpus
pinned to `ecosystem/v0.1.0` and **byte-compares it** — so another repository's
comparison anchor could have been changed from here with nothing reporting it.

| Rule | Why |
|------|-----|
| No deletion | — |
| No update | A moved release tag makes every provenance attestation pointing at it meaningless (ADR-0014) |
| No force push | — |

This matters more than it looks. `release.yml` signs artifacts and attests build
provenance against a tag; if the tag can move, the attestation describes
something that is no longer there.

## `release` — deployment environment

Applied, and used by `release-tag.yml` alone (ADR-0046).

| Rule | Why |
|------|-----|
| Deployment branch policy: protected branches only | A release is cut from `main` or not at all |

**Required reviewers are deliberately not set**, for the same reason required
approvals is 0 above: a single maintainer cannot approve their own dispatch, and
a rule that must always be bypassed is worse than no rule. It rises the day
there is a second maintainer.

This is the only place `contents: write` is granted in this repository.

## Signed commits — required, then removed

`required_signatures` was applied and **removed the same day**, because it
blocked every merge.

No local signing is configured, so branch commits are unsigned. GitHub signs its
own squash-merge commit, and the expectation was that this would satisfy the
rule. It did not: the first pull request reached `mergeable_state=blocked` with a
green `verify` and zero required approvals, and merged only with `--admin`.

**A protection rule that must always be bypassed is worse than no rule.** It
trains the one person who can bypass it to reach for `--admin` by reflex, and
the next rule that fires for a real reason gets the same treatment.

Restoring it needs three things, in order:

```sh
gh auth refresh -h github.com -s admin:ssh_signing_key   # account scope
gh ssh-key add ~/.ssh/id_ed25519.pub --type signing
git config gpg.format ssh
git config user.signingkey ~/.ssh/id_ed25519.pub
git config commit.gpgsign true
```

Then re-add the rule. Registered as B-006 in `docs/blocked.md`. Constitution §6
says authorship is part of provenance; until this is done, it is not.

## Verification

```sh
gh api repos/FabioCaffarello/fdos/rulesets -q '.[] | "\(.name)  \(.target)  \(.enforcement)"'
gh api repos/FabioCaffarello/fdos/rulesets/<id>
gh api repos/FabioCaffarello/fdos/environments/release
```

**`make ruleset-check` now does this** (ADR-0048). `.github/rulesets/` holds the
normalised definition of each ruleset and of the `release` environment; the check
fetches the live settings, normalises both sides identically and diffs.

It runs **locally and deliberately not in CI**. Reading rulesets needs an
admin-scoped token, which ADR-0014 declined to grant a workflow and ADR-0020
recorded as an open gap — but that objection is about CI, not about checking.
From a maintainer's own authenticated CLI it needs no new credential. `make
doctor` invokes it, because a check nobody runs is not a check.

So it is rung 3 when a maintainer runs it and rung 6 from CI's perspective, and
the committed JSON can be updated to match a drift instead of the drift being
reverted — only the review of that commit can tell those apart.

If the API output disagrees with this document, the repository settings are what
actually gate merges — assume the document is stale and fix it.
