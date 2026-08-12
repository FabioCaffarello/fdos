---
id: ADR-0048
title: Drift is reported and never applied, and protection settings are committed as files
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0048 — Drift is reported and never applied, and protection settings are committed as files

> Phase 6 of [RFC-0018](../rfc/0018-the-delivery-pipeline.md), the last of the six.

## Context

Three gaps this repository had written down and not closed, and one it had
written down and already closed without noticing.

### The mitigation ADR-0014 promised was never built

ADR-0014 pinned every action by commit SHA and accepted the cost explicitly:

> **Pinned actions do not receive security fixes automatically.** This is the
> real cost, and it is not small: an unpatched action stays unpatched until
> someone re-resolves the SHA. The mitigation is the scheduled supply-chain
> workflow plus deliberate review.

The scheduled workflow exists and checks the freshness of **nothing**. Measured
the first time a check was pointed at it: of nine pinned actions, **two are
behind** — `actions/attest-build-provenance` at `v4.1.1` against `v4.2.2`, and
`actions/upload-artifact` at `v4.6.2` against `v7.0.1`, three major versions,
pinned by [ADR-0047](0047-a-release-carries-what-the-module-publishes.md) hours
earlier.

### Protection settings were repository state nobody could diff

`docs/branch-protection.md` has said so since it was written:

> **Nothing checks that the live rulesets match this document.** They are
> repository state, not files: someone can change them in the UI with no commit
> here and nothing would notice.

Since then the surface has grown: `release-tags` was extended to three tag
namespaces (ADR-0043) and a `release` environment now carries the only
`contents: write` grant in the repository
([ADR-0046](0046-publishing-a-tag-is-a-dispatched-act.md)).

### A permanently skipped check on every pull request

`supply-chain.yml` triggered on `pull_request` for a job disabled with
`if: false`, because `dependency-review` needs a Dependency Graph this repository
does not have ([#55](https://github.com/FabioCaffarello/fdos/issues/55)). A
skipped job still creates a check run, so every pull request carried a
`supply-chain` check with conclusion `skipped`.

The comment that disabled it made the argument against itself: *"a permanently
red check trains people to ignore CI, which costs more than the check was
worth."* A permanently skipped one is the quieter version of the same thing.

### The platform-parity gap was already closed

[#67](https://github.com/FabioCaffarello/fdos/issues/67) proposed, as option B,
*"a CI job that runs on changes to `Makefile`, `mise.toml` and `.github/**`, so
the platform-sensitive gate runs early"*. Checked before building it: `verify`
already runs `make verify` — including `make test` with `-race` and
`CGO_ENABLED=1`, the exact check that failed in M10 — on `ubuntu-latest` for
**every** pull request, not only those touching those paths. Option B would add a
job that duplicates one that already runs more often.

## Decision

### 1. Freshness is reported, never applied

`make action-freshness` compares every pinned action against its upstream
release and reports what has moved. The weekly `ci-telemetry` run appends the
result to the telemetry log ([#112](https://github.com/FabioCaffarello/fdos/issues/112)).

**It opens no pull request and merges nothing.** Dependabot or Renovate would
close this gap with less code and contradict ADR-0014 head-on, where an input
that can change without a reviewed commit is the whole thing being defended
against. Taking the reporting half and leaving the applying half to a person is
the only version compatible with the decision already made.

It reports a retagged upstream release pointing at the already-pinned SHA as
*current*, not as an upgrade. A list that cries wolf is a list nobody reads.

### 2. Protection settings are committed as JSON and compared

`.github/rulesets/` holds the normalised definition of each ruleset and of the
`release` environment. `make ruleset-check` fetches the live settings, normalises
both sides identically — ids, timestamps and URLs change without anything
changing — and diffs.

### 3. That check runs locally and deliberately not in CI

Reading rulesets needs an admin-scoped token. ADR-0014 declined to put one in CI
and [ADR-0020](0020-open-core-boundary-and-pull-request-workflow.md) recorded the
consequence as an open gap. **That objection is about CI, not about checking**:
run from the maintainer's own authenticated CLI it needs no new credential and
grants nothing to a workflow. It is the same argument `branch-protection.md` used
to apply the rulesets by hand rather than from a workflow.

So it is rung 3 when a maintainer runs it and rung 6 from CI's perspective, and
`make doctor` invokes it — a check nobody runs is not a check.

When it cannot read the settings it says so and exits non-zero. Reporting success
because a token was missing is the failure mode it exists to prevent.

### 4. The phantom check is removed rather than quietened

`supply-chain.yml` loses its `pull_request` trigger and the dormant job. What it
was, why it is gone, and exactly how to restore it once the Dependency Graph is
enabled are recorded in the workflow itself and in #55.

### 5. #67 option B is declined as already satisfied

No job is added. The finding — that `verify` already runs the platform-sensitive
gate on CI's platform for every pull request — is reported on #67, which stays
open for options A and C. Those address the real gap, which is that a developer
cannot reproduce CI's platform *before* pushing, and neither is decided here.

### 6. What this does not decide

**No policy on when a lagging pin must be updated.** The report says what is
behind; a person decides whether and when.

**No automatic remediation of drifted settings.** `ruleset-check` diffs and
refuses; restoring or committing the change is a human act, as applying them was.

## Consequences

### Positive

- The cost ADR-0014 accepted knowingly is now visible instead of theoretical,
  and it found two lagging pins on its first run.
- A ruleset changed in the UI is detectable, including the environment that
  carries the only write permission in the repository.
- Pull requests stop carrying a check that never runs.
- One proposal was closed by measurement rather than by building it.

### Negative

- **`ruleset-check` is enforced by a person running `doctor`.** That is rung 6
  in every practical sense, and calling it rung 3 would be the kind of claim this
  repository keeps finding in its own documents.
- **The committed JSON is a second description of the settings**, and it can be
  updated to match a drift instead of the drift being reverted. The check cannot
  tell those apart; only the review of that commit can.
- **Freshness depends on upstream publishing releases or tags**, and reports
  `upstream unreadable` when it does not. That is a silent-ish gap dressed as a
  line of output.
- **Removing the dormant `dependency-review` job loses the working configuration
  from the workflow file.** It survives as a comment and in #55, which is worse
  than code and better than a check nobody trusts.
- **The freshness report will be ignored if it is long.** Nothing escalates,
  nothing blocks, and a weekly list of two entries is easy to skip past.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| Lagging action pins are visible | 4 | `make action-freshness`, weekly, into #112 |
| A pin is never updated without review | 3 | there is no mechanism that can update one |
| Live protection matches what is committed | 3 locally, 6 in CI | `make ruleset-check`, invoked by `make doctor`, negative-tested |
| No check runs that never runs | 3 | the trigger is gone; nothing to skip |

`ruleset-check` was negative-tested against the live repository: removing
`refs/tags/ecosystem/*` from the `release-tags` ruleset produced a precise diff
and a failure, and the setting was restored and re-verified.

## Alternatives considered

**Dependabot or Renovate.** Near-universal practice, less code, and it would
close the freshness gap properly. Rejected: it is the exact thing ADR-0014
decided against, and adopting it here would reverse an accepted decision by
implementation rather than by supersession.

**Open an issue per lagging action.** More visible than a comment on a log.
Rejected: it produces issues nobody closes, and the weekly comment keeps the
trend in one readable place.

**Put `ruleset-check` in CI with an admin token.** Raises it to a genuine rung 3.
Rejected for the third time in this repository's history, on the same grounds:
the token is a worse risk than the drift it detects.

**Fail `doctor` when protection has drifted.** `doctor` deliberately always
exits 0 — "a diagnostic that fails cannot be run when things are broken."
Rejected to preserve that; the drift is reported as a problem and
`make ruleset-check` is the command that exits non-zero.

**Build #67 option B anyway, for completeness.** It would have looked like
progress. Rejected because it duplicates a job that already runs on every pull
request, and a redundant check is a maintenance cost that teaches nothing.

## Notes

- `actions/upload-artifact` was pinned at `v4.6.2` by ADR-0047 and the freshness
  check reported it three majors behind within the hour. The pin is left as it
  is: updating it is a reviewed commit, and this ADR is not that commit. It is a
  fair demonstration of the mechanism finding something its own author did.
- This closes the sixth and last phase of the delivery-pipeline plan tracked from
  [#107](https://github.com/FabioCaffarello/fdos/issues/107). What that plan
  declined — an affected-pruned gate, a job matrix, and a dependency bot —
  stayed declined throughout, and the measurements that justified declining them
  are now produced weekly rather than by hand.
