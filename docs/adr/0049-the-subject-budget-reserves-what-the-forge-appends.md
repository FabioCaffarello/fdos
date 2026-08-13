---
id: ADR-0049
title: The subject budget reserves what the forge appends, and squash is the only merge
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0049 — The subject budget reserves what the forge appends, and squash is the only merge

## Context

The `commit-msg` hook checks the subject an author writes. GitHub then rewrites
it, appending ` (#NNN)` to the squashed commit, **after every check has passed**.
Nothing checks what lands.

Measured over the last forty commits on `main`:

| | |
|---|---|
| landed subjects over 72 characters | **9** |
| of those, over 72 without the suffix | 2 |
| **violations produced by the forge alone** | **7** |

Seven of forty — 17.5% — break this repository's own rule on subjects that were
compliant when written, and that nobody can now amend (Constitution §4).

### A second cause, found while measuring the first

Five recent commits carry no suffix at all, which should be impossible if every
merge is a squash. `docs/branch-protection.md` says:

> | Squash merge only | One logical change per commit, matching the
> commit-message convention |

The repository settings said otherwise:

```
allow_squash_merge: true   allow_merge_commit: true   allow_rebase_merge: true
```

`required_linear_history` in the `main` ruleset blocks a merge commit, so that
setting was inert. **Rebase merge was not**: it lands the author's commits
unchanged, with no suffix. So the landed-subject rule was not merely violated —
it was violated *unpredictably*, depending on which button was pressed.

`make ruleset-check` (ADR-0048) did not catch this because it covers rulesets and
the `release` environment, and merge behaviour is a repository setting.

### The obvious fix is not available

Suppressing the suffix would remove the cause. GitHub offers
`squash_merge_commit_title` with `PR_TITLE` and `COMMIT_OR_PR_TITLE`, and appends
the number under both — this repository is already on `COMMIT_OR_PR_TITLE` and
every squashed commit carries one. There is no setting for it.

## Decision

### 1. An authored subject is budgeted against what it will become

`SUBJECT_HARD_LIMIT` stays 72 for a landed subject. An authored one is checked
against **72 minus a reserve of 8** — the width of ` (#9999)`.

The check distinguishes the two by whether the subject already ends in
` (#NNN)`: one that does has landed and is measured as it is; one that has not is
about to acquire one and is measured against what it will become. That makes
`branch` mode correct over merged history and `message` mode correct at the
moment of writing, without a second rule.

The failure says why, rather than reporting a limit the author cannot see in the
message they wrote:

```
commit-msg: subject is 67 characters; the limit is 64 because squash
  merge appends " (#NNNN)" after every check has passed, and 67 + 8 > 72
```

**Eight characters is an assumption with an expiry.** At five-digit pull-request
numbers the reserve becomes nine, and this ADR is where that is written down.

### 2. Squash is the only merge, in settings and not only in prose

`allow_merge_commit` and `allow_rebase_merge` are off. The documented policy has
been "squash merge only" since it was written; it is now true.

This is what makes §1 sound: with rebase merges available, an authored subject
might or might not acquire a suffix, and a budget that reserves for one is wrong
half the time.

### 3. `ruleset-check` covers the repository's merge settings

`.github/rulesets/repository-merge.json` holds them, and the check diffs the
live values. The settings that decide what a merge *is* now sit beside the
rulesets that decide what may merge.

### 4. What this does not decide

**No back-fill.** The seven landed violations stay. They are history, and
Constitution §4 is the reason there is a rule about not rewriting it.

**No policy on pull-request titles.** With `COMMIT_OR_PR_TITLE`, a single-commit
pull request lands its commit subject and a multi-commit one lands the pull
request title. The second path is unchecked by anything here, and stating that
is better than implying the budget covers it.

## Consequences

### Positive

- What lands satisfies the rule the repository says it holds, rather than what
  was written before the forge edited it.
- The suffix is now universal and predictable, so the budget is right for every
  commit rather than for most of them.
- A documented policy that was false for the whole life of the document is true.
- Merge settings can no longer drift unnoticed.

### Negative

- **The effective subject limit drops from 72 to 64 for authors.** That is
  tighter than the documented number, and the message has to explain why every
  time it fires. Several commits in this repository's recent history would have
  been rejected by it — which is the point, and is also a real cost in rework.
- **The reserve is a guess with a known expiry.** Eight characters is right
  through `#9999`. Nothing detects the transition; the number is in one place and
  in this ADR, and that is all.
- **A multi-commit pull request lands its *title*, which no hook ever saw.**
  The budget does not help there. It is the gap that remains and it is not
  small — it is exactly the shape of the defect this ADR fixes, one level up.
- **Turning off rebase merge removes a workflow somebody may have been using.**
  The evidence says it was used at least five times recently.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| An authored subject leaves room for the suffix | 4 | `commit-msg` hook and `make commit-msg-check`, negative-tested |
| A landed subject is within 72 | 4 | the same check, over `origin/main..HEAD` |
| Squash is the only merge | 3 | repository settings |
| Merge settings match what is committed | 3 locally, 6 in CI | `make ruleset-check`, negative-tested |
| A pull-request title is within budget | 6 | nothing. Stated in §4 rather than implied |

## Alternatives considered

**Suppress the suffix (option B in [#111](https://github.com/FabioCaffarello/fdos/issues/111)).**
The cleanest fix — remove the cause. Rejected because it is not available:
GitHub appends the number under both title settings, and this repository is
already on the one that would avoid it if any did.

**Raise the hard limit to 76 (option C).** Honest about the forge's behaviour,
one number to change. Rejected: it gives up two characters of the reason the
limit exists — `git log --oneline` in an 80-column terminal — to accommodate a
suffix the author did not write. Reserving is the same arithmetic without moving
the goal.

**Check the merge commit on `main` instead (option D).** Catches everything,
including the pull-request-title path §4 leaves open. Rejected as the primary
mechanism: it reports after the subject is unamendable, which is rung 3
enforcing a rule nobody can then satisfy. It remains the honest answer to §4's
gap and is not built here.

**Warn rather than fail.** #111 proposed a warning, which costs nobody anything.
Rejected: the existing 50-character guideline already warns and is routinely
exceeded — including throughout this ADR's own series. A warning would have
produced the same seven violations.

**Leave the merge settings alone and reserve anyway.** Fewer moving parts.
Rejected: with rebase available the reserve is wrong whenever it is used, and a
budget that is right most of the time is how the 50-character guideline became
decorative.
