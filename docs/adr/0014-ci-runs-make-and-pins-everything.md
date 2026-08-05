---
id: ADR-0014
title: CI invokes make and nothing else; every build input is pinned by digest
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0014 — CI invokes `make` and nothing else; every build input is pinned by digest

## Context

M3 introduces the first automation that runs outside a developer's machine.
Two failure modes are worth deciding against explicitly, because both are the
default outcome.

**Logic migrating into workflow YAML.** A check written directly in a workflow
cannot be run locally, cannot be debugged without pushing, and drifts from what
developers execute. The result is the familiar state where `make verify` passes,
CI does not, and nobody can say why — at which point the pipeline stops being
trusted and starts being worked around.

**Mutable build inputs.** An action referenced as `@v4` can change under the
repository with no commit here. That is an unreviewed third party with write
access to the build, and therefore to every artifact, SBOM and provenance
attestation the build produces. For software asking institutions to trust its
output, the attestation is worth exactly as much as the weakest input to it.

`.github/README.md` asserted both rules from M0. Neither had a mechanism.

## Decision

### CI invokes `make` targets

A workflow may check out code, install a pinned tool, and call `make`. It may
not contain the check itself.

The narrow exception is tool installation, which is inherently
environment-specific. Even there, the *version* comes from `mise.toml` through
`scripts/tool-version.sh` — CI never declares a version of its own, so the pins
developers use and the pins CI uses cannot diverge.

### Every action is pinned to a full commit SHA

No tags, no branches, no floating major versions. Resolve with:

```sh
gh api repos/<owner>/<repo>/git/ref/tags/<tag> --jq .object.sha
```

`make action-pinning-check` fails the build on any unpinned reference.

### `GOWORK=off` everywhere

Every workflow sets it, as every Makefile target does. This is the load-bearing
half of ADR-0004: without it, module resolution goes through local workspace
paths and the open-core boundary silently stops being verified.

### Hooks are convenience, never the guarantee

`lefthook` runs a fast subset pre-commit and the full `make verify` pre-push.
Hooks call the same `make` targets for the same reason CI does.

They are explicitly bypassable. `--no-verify` costs the author a round trip and
cannot let anything through, because CI runs the full gate regardless. A hook
treated as the guarantee would make the guarantee optional.

### Accepted ADRs are immutable, and now checked

ADR-0000 made the decision log append-only. Until now that was enforced by
review alone. `make adr-immutability-check` compares every ADR against the
commit that introduced it, permitting only supersession metadata and added
lines.

## Consequences

### Positive

- Local and CI verification are the same check set, invoked the same way. A
  green `make verify` is a meaningful prediction of CI.
- A compromised or hijacked action cannot silently enter the build.
- The provenance attestations produced at release are worth something, because
  every input to the build is identified by digest.
- ADR immutability moves from rung 6 to rung 3, closing a gap the repository has
  carried since M0.

### Negative

- **Pinned actions do not receive security fixes automatically.** This is the
  real cost, and it is not small: an unpatched action stays unpatched until
  someone re-resolves the SHA. The mitigation is the scheduled supply-chain
  workflow plus deliberate review, and the trade is accepted because an
  unreviewed automatic update is the larger risk for this system.
- Updating an action is now a chore requiring an API call, so it will be done
  less often. Expect the pins to lag.
- `make verify` in CI runs the full set with no affected-graph pruning, so CI
  time grows with the number of modules. `scripts/affected-modules.sh` exists
  but deliberately does not gate correctness; if CI time becomes a problem, the
  fix is a separate fast job, not a narrower gate.
- The gitleaks install step in `verify.yml` downloads a release tarball by
  version but not by checksum — a genuine gap in the pinning claim, noted rather
  than hidden. See Notes.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| Actions pinned by SHA | 3 | `make action-pinning-check`, negative-tested |
| Accepted ADRs not rewritten | 3 | `make adr-immutability-check`, negative-tested |
| No secrets in history | 3 | `make secrets-check` (gitleaks, full history) |
| No reachable vulnerabilities | 3 | `make vuln-check` (govulncheck, pinned by module version) |
| `GOWORK=off` | 3 | set in every workflow and Makefile target |
| CI contains no logic | 6 | review. Partially observable, not mechanically decidable |

## Alternatives considered

**Write checks directly in workflow YAML.** Standard practice, and it avoids a
layer of indirection. Rejected on the local/CI drift argument: the value of a
gate is that developers can run it before pushing.

**Pin actions to major version tags (`@v4`).** What most repositories do, and it
receives security fixes automatically. Rejected: for a project whose output is
meant to be verifiable, an input that can change without a commit invalidates
the whole chain. The cost — manual updates — is accepted knowingly.

**Vendor the actions into the repository.** Maximum control, no external
mutation at all. Rejected as disproportionate: SHA pinning already removes the
mutation, and vendoring adds a maintenance surface with no further guarantee.

**Enforce hooks by making CI reject unhooked commits.** Rejected: it would make
a local convenience into a requirement, and there is no honest way to detect it.
CI already re-runs everything, so the hook adds speed, not safety.

**Make `make verify` run only affected modules.** Faster, and the affected-graph
script exists. Rejected as the *gate*: under-reporting affectedness ships a
broken module, and the failure is silent. Speed belongs in a separate job.

## Notes

Open, deliberately:

- **The gitleaks install is pinned by version, not by checksum.** Every other
  build input is digest-pinned; this one is not, and stating it is better than
  implying coverage. Fix is either a checksum-verified download or running
  gitleaks through a SHA-pinned action in `verify.yml` too.
- Branch protection, required checks and merge queue are GitHub settings, not
  files. They cannot be enforced from this repository and are documented in
  `docs/branch-protection.md` instead. A repository ruleset committed as JSON
  and applied by a workflow would raise this from documentation to CI; it needs
  an admin token, which is its own risk.
- `syft` and `cosign` are pinned by action SHA rather than in `mise.toml`,
  because they run only in CI. If they ever run locally, they need pins.
- SLSA provenance is attested for `fdoslint` only. Go libraries are released as
  tags served by the module proxy and have no artifact to attest; whether the
  module zips themselves should be attested is unresolved.
