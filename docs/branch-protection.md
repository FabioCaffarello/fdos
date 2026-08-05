# Branch and tag protection

**Applied**, as two repository rulesets (ADR-0020). This document records what
is configured and why, so a change made in the GitHub UI can be recognised as a
change.

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

## `release-tags` — tag ruleset

Applies to `refs/tags/libs/*/v*`.

| Rule | Why |
|------|-----|
| No deletion | — |
| No update | A moved release tag makes every provenance attestation pointing at it meaningless (ADR-0014) |
| No force push | — |

This matters more than it looks. `release.yml` signs artifacts and attests build
provenance against a tag; if the tag can move, the attestation describes
something that is no longer there.

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
```

**Nothing checks that the live rulesets match this document.** They are
repository state, not files: someone can change them in the UI with no commit
here and nothing would notice. A check calling the API and diffing against
committed JSON is feasible but needs a token in CI, which is the risk ADR-0014
declined. Recorded as an open gap in ADR-0020 rather than solved badly.

If the API output disagrees with this document, the repository settings are what
actually gate merges — assume the document is stale and fix it.
