---
directory: .github
purpose: Forge configuration — CI workflows, review templates and repository automation.
owner: "@FabioCaffarello"
allowed:
  - GitHub Actions workflows that invoke make targets
  - Tool installation steps, versioned from mise.toml
  - Local composite actions that install tools, shared by workflows
  - Pull request and issue templates
  - Actions pinned to full commit SHAs
forbidden:
  - Enforcement logic implemented inline in workflow YAML
  - Actions referenced by tag or branch instead of commit SHA
  - Tool versions declared here instead of read from mise.toml
  - Secrets in plaintext
  - Checks that cannot be reproduced locally through make
---

# .github

Forge configuration for FDOS. Governed by ADR-0014.

## Workflows

| Workflow | Trigger | Runs |
|----------|---------|------|
| `verify.yml` | every push to `main`, every pull request | `make affected-preflight` (advisory), `make verify`, then `make ci-summary` |
| `supply-chain.yml` | weekly schedule | scheduled vulnerability and secret scans |
| `release.yml` | tags matching `libs/*/v*` and `apps/*/v*`, or manual dispatch with a tag | `make release-artifacts`, then the shared `release-evidence` action |
| `release-rehearse.yml` | manual dispatch | the whole release path against a published tag, stopping before publication |
| `ci-telemetry.yml` | weekly schedule, manual dispatch | `make ci-stats`, `make action-freshness`, `make verify-timings` |
| `release-prepare.yml` | manual dispatch | `make release-prepare`, then opens the pull request |
| `release-tag.yml` | manual dispatch | `make release-tag` — the only job that may write a tag |

`release-tag.yml` is the one place `contents: write` appears, behind a `release`
environment restricted to `main`. Publishing is a dispatched act because keyless
signing binds the artifact's identity to the workflow: a tag pushed without a
human choosing to publish is a signed statement nobody decided to make
(ADR-0046). Its `publish` input must be typed as `yes`; anything else is a dry
run.

`ci-telemetry.yml` gates nothing. It exists because nothing here could say what
the gate costs: measured across twenty runs, `verify` ranged from 87s to 279s —
a 3.2× spread with no record of which runs restored the build cache and which
did not. `make ci-summary` now writes that per run, and the weekly job turns
single readings into a distribution, logged in
[issue #112](https://github.com/FabioCaffarello/fdos/issues/112).

## CI runs `make`, and nothing else

The single most important rule for this directory.

If a check exists only in workflow YAML, it cannot be run locally, cannot be
debugged without pushing, and drifts from what developers actually execute. The
result is the familiar failure where `make verify` passes and CI does not, and
nobody can say why — at which point the pipeline stops being trusted and starts
being worked around.

The narrow exception is tool installation, which is inherently
environment-specific. Even there the **version comes from `mise.toml`** via
`scripts/tool-version.sh`. CI never declares a version of its own, so the pins
developers use and the pins CI uses cannot diverge.

That exception lives in one place: `actions/setup-toolchain`, used by both
`verify.yml` and `release.yml`. It was extracted after the two drifted —
`release.yml` had been given Go alone, so `make verify` failed at
`toolchain-check` on every one of fourteen tags and no release was ever
published ([`docs/blocked.md`](../docs/blocked.md) — B-008). A second copy of
the setup is how that happened, so there is now only one.

## Pinned actions

Every action is referenced by full commit SHA, never by tag or branch.

A tag is mutable: an action referenced as `@v4` can change under the repository
with no commit here. That is an unreviewed third party with write access to the
build — and therefore to every artifact, SBOM and provenance attestation the
build produces. An attestation is worth exactly as much as the weakest input to
it.

`make action-pinning-check` fails the build on any unpinned reference, in
workflows **and** in local composite actions under `actions/`. The second half
was added when extracting `setup-toolchain` moved two pinned references out of
the check's sight: the only symptom was the reported count falling from 14 to
13, and an unpinned reference in that file would have passed silently.

To resolve a SHA:

```sh
gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha
```

The cost is real and accepted: pinned actions do not receive security fixes
automatically, so the pins will lag. ADR-0014 records why that trade was made.

## `GOWORK=off`

Every workflow sets it, as every Makefile target does (ADR-0004). Without it,
module resolution goes through local workspace paths and the open-core boundary
silently stops being verified — with nothing to indicate that it has stopped.

## What is not here

**Branch protection, required checks and the merge queue are repository
settings, not files.** They cannot be enforced from this directory. The intended
configuration is in [`docs/branch-protection.md`](../docs/branch-protection.md),
which states plainly that it is a checklist rather than a mechanism.

Git hooks live in `lefthook.yml` at the repository root. They call the same
`make` targets, run a fast subset pre-commit and the full gate pre-push, and are
explicitly bypassable — CI re-runs everything regardless, so a hook adds speed,
not safety.
