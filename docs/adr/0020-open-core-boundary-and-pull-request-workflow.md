---
id: ADR-0020
title: The repository is named fdos, the boundary is proven, and work moves to pull requests
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0020 — The repository is named `fdos`, the boundary is proven, and work moves to pull requests

## Context

M5 exists to make the open-core boundary real: private repositories consume a
published contract version, not a local path (ADR-0004, Constitution §13).

Attempting it surfaced that the boundary could not have worked.

**ADR-0003 chose the module path `github.com/FabioCaffarello/fdos/...` while the
repository was named `financial-data-operating-system`, and recorded the
divergence as deliberate**: "import statements are read far more often than
repository URLs." That reasoning ignores how Go resolves modules — the
repository is *derived from* the module path. `github.com/FabioCaffarello/fdos`
did not exist, and the proxy returned 404 for every version of every module.

Nothing caught it for five milestones because nothing had ever tried to consume
a module from outside the workspace. `GOWORK=off` proves modules resolve by
published *version*; it does not prove the published *path* exists.

## Decision

### The repository is renamed to `fdos`

The module path stands. ADR-0003's decision was right and its supporting note
was factually wrong, so the repository is brought into line with the path rather
than the reverse.

Chosen over the alternatives because it costs nothing measurable: zero import
changes, zero consumers to migrate, and GitHub redirects the old URL. Every
month of delay makes it more expensive.

**ADR-0003 is not superseded.** Its `## Decision` — the module path — is
unchanged and in force. What was wrong lived in its rationale, and ADR-0000
requires supersession for a change to what a decision *says*. It gains a
forward pointer here instead, which its immutability check permits.

### The boundary is proven by a consumer outside the repository

`make consumer-check` creates a throwaway module in a temporary directory, with
no workspace, no `replace` and no filesystem path, and:

1. resolves `github.com/FabioCaffarello/fdos/libs/contracts@vX.Y.Z` through the
   Go proxy,
2. resolves the transitive graph and asserts the version was not moved,
3. compiles a program against the published types,
4. asserts no `replace` directive appeared.

It is **not** in `make verify`. It depends on a published tag propagating to a
third-party proxy, and the per-commit gate must not inherit that latency — the
mistake ADR-0018 made once with a remote codegen plugin. It runs at release and
on demand.

`libs/contracts/v0.1.0` is published and passes.

### Branch and tag protection are applied, not documented

`docs/branch-protection.md` described the intended configuration and stated
plainly that it was a checklist rather than a mechanism (ADR-0014). It is now
applied as two repository rulesets:

| Ruleset | Target | Rules |
|---------|--------|-------|
| `main` | default branch | pull request required, `verify` status check (strict), linear history, signed commits, conversation resolution, squash only, no deletion, no force push |
| `release-tags` | `refs/tags/libs/*/v*` | no deletion, no update, no force push |

Tag protection matters more than it looks: a release tag that can be moved makes
every provenance attestation pointing at it meaningless (ADR-0014).

**Required approvals is 0, deliberately.** A single-maintainer repository with
one required approval cannot merge anything — the author cannot approve their
own pull request. Zero still requires the pull request, the status check and the
review threads to be resolved. It rises to 1 the day there is a second
maintainer, and that is the only reason it is not 1 now.

### Work moves to pull requests

Direct pushes to `main` are no longer possible. `.github/pull_request_template.md`
carries the obligations the gate cannot check: whether an ADR is required,
whether a new mechanism was negative-tested, whether §15 moved, and — the field
most likely to be skipped — whether documentation was updated in the same change
rather than deferred.

The template asks explicitly whether `make verify` was run or could not be, on
the principle that a claimed check that was not run is worse than an unchecked
change.

### Blocked work is registered, not silently omitted

`docs/blocked.md` records what FDOS decided to do and cannot finish, with the
blocker and what unblocks it. M5's own acceptance criterion is the first entry.

## Consequences

### Positive

- The open-core boundary is verified end to end in the direction this
  repository controls. `go get` works for anyone.
- A moved release tag, a force push to `main`, and a merge with a red `verify`
  are all now impossible rather than discouraged.
- Constitution §13 gains its first real mechanism beyond `GOWORK=off`.
- Blocked work is visible. The next reader can tell "not done" from
  "deliberately not done".

### Negative

- **M5's stated acceptance criterion is not met.** `financial-connectors` is
  empty, so no *private* repository has consumed the contract. The private path
  involves credentials, `GOPRIVATE` and a private proxy route, none of which is
  tested. Recorded as B-001 rather than quietly rewritten to match what was
  achieved.
- Renaming a public repository breaks any URL not following redirects, and
  redirects are not permanent guarantees. There are no external consumers today,
  which is exactly why it was done today.
- The pull request workflow adds a round trip to every change, including
  one-line documentation fixes, for a repository with one maintainer. The
  benefit is entirely in the future when there is a second.
- Signed commits are required but no signing is configured locally. GitHub signs
  its own squash-merge commits, so merges work; the branch commits themselves
  remain unsigned, which makes the requirement weaker than it reads.
- Rulesets are repository state, not files. They can be changed in the GitHub UI
  with no commit here, and nothing in this repository would notice. The JSON
  that created them is in this ADR's history and in `docs/branch-protection.md`;
  drift between the two is unchecked.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| Published module is consumable | 3 | `make consumer-check`, at release |
| No merge without a green `verify` | 3 | `main` ruleset, strict status check |
| No force push or deletion on `main` | 3 | `main` ruleset |
| Release tags immutable | 3 | `release-tags` ruleset |
| A private repo can consume it | 6 | **blocked** — B-001 |
| Rulesets match what is documented | 6 | nothing; see Negative |

## Alternatives considered

**Change the module path to match the old repository name.** No rename, and
nothing outside the repository changes. Rejected: it makes every import
`github.com/FabioCaffarello/financial-data-operating-system/libs/...` forever,
which is precisely what ADR-0003 weighed and rejected on ergonomics.

**A vanity import path (`fdos.dev/...`).** ADR-0003 considered this and deferred
it with "revisit before the first external consumer exists", which is now.
Rejected again: it introduces an availability dependency — an outage or lapsed
registration of the meta-tag endpoint breaks builds for every consumer — to
solve a problem (migrating off GitHub) that is not on the roadmap.

**Wait for `financial-connectors` before claiming M5.** Rejected: the blocker is
outside this repository and has no date. Delivering the half that is provable,
and registering the half that is not, is more useful than a milestone held open
indefinitely.

**Require one approving review.** Rejected as deadlock for a solo repository,
with the trigger for raising it recorded above.

**Keep branch protection as documentation.** Rejected — the user asked for it
applied, and ADR-0014's objection was specifically to putting an admin-scoped
token in CI. Applying rulesets by hand from an authenticated CLI does not
require that, so the objection does not transfer.

## Notes

Open:

- Nothing verifies that the live rulesets match `docs/branch-protection.md`. A
  check calling `gh api .../rulesets` and diffing against committed JSON is
  feasible; it needs a token in CI, which is the risk ADR-0014 declined. Left
  unresolved rather than solved badly.
- Commit signing is required by the ruleset but not configured locally. Setting
  up SSH signing is cheap and would make the requirement mean what it says.
- `make consumer-check` proves the *public* path. The private-module path
  (`GOPRIVATE`, credentials) is untested and is the substance of B-001.
