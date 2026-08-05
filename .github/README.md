---
directory: .github
purpose: Forge configuration — CI workflows, review templates and repository automation.
owner: "@FabioCaffarello"
allowed:
  - GitHub Actions workflows
  - Pull request and issue templates
  - Dependency and security automation configuration
  - Actions pinned to full commit SHAs
forbidden:
  - Enforcement logic implemented inline in workflow YAML
  - Actions referenced by tag or branch instead of commit SHA
  - Secrets in plaintext
  - Checks that cannot be reproduced locally through make
---

# .github

Forge configuration for FDOS.

## Status: empty by design

`.github/` contains no workflows at M0. CI/CD is **M3**, and adding workflows
before the enforcement mechanisms they run would produce a pipeline with nothing
to enforce.

## CI runs `make`, and nothing else

The single most important rule for this directory: **CI invokes Makefile targets
and contains no logic of its own.**

If a check exists only in workflow YAML, it cannot be run locally, cannot be
debugged without pushing, and drifts from what developers actually execute. The
result is the familiar failure where `make verify` passes and CI does not, and
nobody can say why.

Local and CI verification must be the same check set, invoked the same way.

## Pinned actions

Every action is referenced by full commit SHA, never by tag or branch. Tags are
mutable: an action referenced by `@v4` can change under the repository without
any commit here. That is an unreviewed third party with write access to the
build — unacceptable for software that will ask institutions to trust its output
(Constitution §9, §14).

## `GOWORK=off`

Every Go workflow must set `GOWORK=off` (ADR-0004). Without it, module
boundaries resolve through local paths and the open-core boundary silently stops
being verified — with no failure to indicate that it has stopped.
