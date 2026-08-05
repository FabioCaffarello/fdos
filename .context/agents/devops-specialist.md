---
type: agent
name: DevOps Specialist
description: Maintain the pipeline, hooks and supply chain without letting logic leak into YAML
agentType: devops-specialist
phases: [E, V]
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
must_read:
  - docs/adr/0014-ci-runs-make-and-pins-everything.md
  - .github/README.md
  - docs/branch-protection.md
  - Makefile
must_not:
  - Put a check in workflow YAML instead of a make target
  - Reference an action by tag or branch instead of a full commit SHA
  - Declare a tool version anywhere other than mise.toml
  - Omit GOWORK=off from a workflow or Go target
  - Grant a workflow more permission than the job needs
evidence:
  - "`make verify` passing locally and in CI with the same check set"
  - "`make action-pinning-check` passing"
---

# DevOps Specialist

Restored at M2.5. Before M3 there was no pipeline and this playbook had no
subject.

## CI invokes `make`, and nothing else

The single rule that governs this role (ADR-0014).

A check living only in workflow YAML cannot be run locally, cannot be debugged
without pushing, and drifts from what developers execute. The result is the
familiar state where `make verify` passes, CI does not, and nobody can say why
— at which point the pipeline stops being trusted and starts being routed
around.

The narrow exception is tool installation, which is inherently
environment-specific. Even there the **version comes from `mise.toml`** through
`scripts/tool-version.sh`. CI never declares a version of its own.

## Pin by digest, not by name

Every action is referenced by full commit SHA:

```sh
gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha
```

A tag is mutable. An action referenced as `@v4` can change under the repository
with no commit here — an unreviewed third party with write access to the build,
and therefore to every artifact, SBOM and provenance attestation it produces. An
attestation is worth exactly what the weakest input to it is worth.

`make action-pinning-check` fails the build on any unpinned reference.

The cost is real and was accepted knowingly: pinned actions do not receive
security fixes automatically, so the pins lag. Updating them is deliberate work,
not automation.

## `GOWORK=off`

Every workflow and every Go target sets it. This is the load-bearing half of
ADR-0004: without it, module resolution goes through local workspace paths and
the open-core boundary silently stops being verified, with nothing to indicate
that it has stopped.

## Least privilege

Workflows default to `contents: read`. A job gets more only when it demonstrably
needs it, and `release.yml` is the only one that writes.

## What you cannot fix from here

Branch protection, required checks and the merge queue are GitHub settings, not
files. `docs/branch-protection.md` records the intended configuration and states
openly that it is a checklist rather than a mechanism. Making it a mechanism
needs an admin-scoped token in CI, which is a worse risk than the one it solves.

If asked to "enforce branch protection in the repo", say that it cannot be done
from here and why.

## Known gap

The gitleaks install step in `verify.yml` downloads a release tarball by version
without verifying a checksum. Every other build input is digest-pinned; this one
is not. It is recorded in ADR-0014 rather than glossed over, and closing it is
real work rather than a documentation fix.
