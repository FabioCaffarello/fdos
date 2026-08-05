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

Planned for M3, none of it in place yet:

- `govulncheck` on every build
- SBOM generation (`syft`)
- SLSA provenance attestation for release artifacts
- `cosign` signing
- dependency review on pull requests
- **all GitHub Actions pinned to full commit SHAs**

The last is load-bearing. An action referenced by `@v4` can change under the
repository with no commit here — an unreviewed third party with write access to
the build. For software asking institutions to trust its output, that is not
acceptable.

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

## Current gaps

Stated plainly so nobody assumes coverage that does not exist:

| Gap | Closes at |
|-----|-----------|
| No secret scanning | M3 |
| No dependency vulnerability scanning | M3 |
| No SBOM or provenance attestation | M3 |
| No signed releases | M3 |
| No enforcement of the LLM boundary | M4 |
| No branch protection or required checks | M3 |

At M1 the security posture rests almost entirely on review. That is accurate,
and it is why M3 exists.

## Reporting

There is no published vulnerability disclosure process yet. It belongs with M3,
alongside the supply chain work, and before any external consumer exists.
