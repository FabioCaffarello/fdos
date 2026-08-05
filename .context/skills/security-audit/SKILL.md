---
type: skill
name: Security Audit
description: Audit FDOS for integrity and supply-chain failures. Use when reviewing a dependency change, a workflow change, anything touching the build inputs, or a proposal that routes model output toward persistence
skillSlug: security-audit
phases: [R, V]
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

## Workflow

1. Read `.context/docs/security.md` — it holds the threat model and the explicit
   list of what is **not** enforced.
2. Rank by integrity impact, not by conventional severity label.
3. Check build inputs: is everything identified by digest?
4. Check the truth boundary: can any model output reach the ledger?
5. Check determinism: a non-reproducible calculation is an integrity defect.
6. Run `make secrets-check` and `make vuln-check`. Report what you ran.

## What actually matters here

The consequential failures in FDOS are integrity and auditability failures, not
confidentiality breaches. Rank accordingly:

1. Silent corruption of a financial fact — undetectable later, poisons every
   derived report.
2. Loss of provenance — a datum with no origin cannot be audited.
3. Unreproducible calculation — a regulator's question in 2031 becomes
   unanswerable.
4. Compromised build input — alters output with no commit.
5. Credential leakage.

A finding about #4 outranks most conventional vulnerability classes in this
codebase.

## Examples

**Build input, blocking:**
```
.github/workflows/release.yml:41 uses anchore/sbom-action@v0.

A tag can be moved under the repository with no commit here. This action
generates the SBOM attached to signed releases, so a moved tag rewrites
what we attest to — the attestation is worth what its weakest input is
worth (ADR-0014).

  gh api repos/anchore/sbom-action/git/ref/tags/v0.24.0 --jq .object.sha
```

**Determinism as an integrity finding:**
```
libs/ledger/domain/accrual.go:31 calls time.Now().

Not a style issue. The calculation cannot be reproduced from the ledger,
which makes every report derived from it unauditable (Constitution §2, §9).
The clock is injected at the app boundary, where the reading becomes
recorded provenance.
```

**Correct answer to "is FDOS secure":**
```
Partially, and here is the boundary.

Enforced: full-history secret scanning, reachable-vulnerability scanning,
SHA-pinned build inputs, ADR immutability, SBOM + provenance + signature
at release.

Not enforced: the LLM truth boundary (M4). Branch protection is a GitHub
setting, documented in docs/branch-protection.md as a checklist, not a
mechanism. The gitleaks CI install is pinned by version, not checksum.
```

## Quality Bar

- Never imply coverage that does not exist. The gap list in
  `.context/docs/security.md` is the authority, and it is deliberately
  unflattering.
- State the concrete failure: input, state, consequence. Label a hypothesis as a
  hypothesis.
- A secret found anywhere in history is a leak, not a finding to fix in the
  working tree — the object stays reachable and anyone who cloned already has it.
- `govulncheck` reports **reachable** vulnerabilities. Do not escalate an
  unreachable finding; a scanner people learn to ignore enforces nothing.
- Treat any proposal routing model output toward persistence as a design error
  and name Constitution §2.

## Resource Strategy

- No `references/`: the threat model and gap list live in
  `.context/docs/security.md`; duplicating them here guarantees the copy goes
  stale and gets quoted.
