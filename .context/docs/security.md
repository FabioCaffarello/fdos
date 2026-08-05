---
type: doc
name: security
description: Security posture, secret policy, supply chain plan, and current gaps
category: security
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Security

Security is part of development, not a phase after it (Constitution §14). This
document states both the policy and, explicitly, what is not yet enforced.

## Threat model

FDOS is asked to be trusted with financial truth. The consequential failures are
not the obvious ones:

| Threat | Why it matters here |
|--------|---------------------|
| Silent corruption of a financial fact | Undetectable later, and poisons every derived report |
| Loss of provenance | A datum with no origin cannot be audited or re-verified |
| Unreproducible calculation | A regulator's question in 2031 becomes unanswerable |
| Compromised build input | An unpinned action or dependency can alter output with no commit |
| Credential leakage from connectors | Private repositories hold real institution credentials |

Note that three of five are integrity and auditability failures, not
confidentiality breaches. That ordering should shape review priorities.

## Secrets

No secret value is ever committed, in any form, including examples shaped like
real values. `.env.example` carries placeholders only.

`.gitignore` covers `*.pem`, `*.key`, `.env`, `.env.*`. This is defence in depth,
not enforcement — **gitleaks in CI is the enforcing mechanism and arrives at
M3.** Until then, secret hygiene is rung 6.

Private repositories hold authenticated provider credentials. They are separate
repositories precisely so that this material never shares a trust boundary with
the public core.

## Supply chain

In place since M3:

- `govulncheck` in `make verify` and on a weekly schedule
- SBOM generation (`syft`, SPDX) at release
- SLSA provenance attestation for release artifacts
- `cosign` keyless signing of the checksum manifest
- dependency review on pull requests, copyleft licences denied
- **all GitHub Actions pinned to full commit SHAs**, enforced by
  `make action-pinning-check`

The last is load-bearing. An action referenced by `@v4` can change under the
repository with no commit here — an unreviewed third party with write access to
the build, and therefore to every artifact and attestation it produces. An
attestation is worth exactly as much as the weakest input to it.

The accepted cost: pinned actions do not receive security fixes automatically,
so the pins will lag. ADR-0014 records why that trade was made rather than the
reverse.

Keyless signing means there is no private key to leak and no key custody to get
wrong: the signing identity is the workflow itself, bound by an OIDC token.

## Toolchain integrity

`mise.toml` pins every tool version and `make toolchain-check` enforces the pins
whether or not `mise` is installed. A wrong version is always an error; a
missing not-yet-required tool is a warning.

`GOFLAGS=-mod=readonly` prevents implicit dependency resolution during a build.
An implicit `go mod tidy` is a silent, unreviewed change to the dependency graph.

## The LLM boundary

Model outputs must never become financial truth (Constitution §2). At M4 this
becomes structural: model outputs carry a distinct type the ledger write path
cannot accept, checked in CI.

Until then it is a principle in a document, which is to say it is rung 6. Treat
any proposal that routes model output toward persistence as a design error and
say so explicitly.

## Current posture — M3

| Control | Mechanism | Where |
|---------|-----------|-------|
| Secret scanning | `gitleaks`, **full history** | `make secrets-check`, pre-commit hook, weekly schedule |
| Reachable vulnerabilities | `govulncheck` at a pinned module version | `make vuln-check`, weekly schedule |
| Dependency delta review | `dependency-review-action`, copyleft denied | pull requests |
| Build input integrity | every action pinned by commit SHA | `make action-pinning-check` |
| Decision-log integrity | ADR diffed against its introducing commit | `make adr-immutability-check` |
| Release integrity | SBOM, SLSA provenance attestation, cosign keyless signature | `release.yml` |

The secret scan reads **history, not the working tree**. A secret committed and
then removed is still leaked: the object stays reachable, and anyone who cloned
before the removal already has it.

`govulncheck` runs through `go run` at a pinned version rather than as an
installed binary. A govulncheck built with go1.25 cannot parse go1.26 source and
fails with a toolchain error that reads exactly like a scan result — a failure
mode worth designing out.

## Remaining gaps

Stated plainly so nobody assumes coverage that does not exist:

| Gap | Closes at |
|-----|-----------|
| No enforcement of the LLM boundary | M4 |
| Branch protection is documentation, not a mechanism | — see below |
| The gitleaks CI install is pinned by version, not checksum | M3.5 |

**Branch protection, required checks and the merge queue are GitHub settings,
not files.** They cannot be enforced from this repository.
`docs/branch-protection.md` records the intended configuration and says openly
that it is a checklist. Raising it to a mechanism needs an admin-scoped token in
CI, which is a worse risk than the one it solves (ADR-0014).

The gitleaks install step downloads a release tarball by version but does not
verify a checksum. Every other build input is digest-pinned; this one is not,
and it is recorded rather than glossed over.

## Reporting

There is no published vulnerability disclosure process yet. It belongs with M3,
alongside the supply chain work, and before any external consumer exists.
