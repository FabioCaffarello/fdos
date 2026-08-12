---
id: ADR-0043
title: Downloaded build artifacts are pinned by digest, and release tags are protected wherever they are attested
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0043 — Downloaded build artifacts are pinned by digest, and release tags are protected wherever they are attested

> Phase 1 of [RFC-0018](../rfc/0018-the-delivery-pipeline.md).

## Context

[ADR-0014](0014-ci-runs-make-and-pins-everything.md) pinned every GitHub Action
to a full commit SHA, on the argument that a mutable reference is "an unreviewed
third party with write access to the build — and therefore to every artifact,
SBOM and provenance attestation the build produces". It then recorded, in its
own Notes, that the rule did not reach everything:

> **The gitleaks install is pinned by version, not by checksum.** Every other
> build input is digest-pinned; this one is not, and stating it is better than
> implying coverage. Fix is either a checksum-verified download or running
> gitleaks through a SHA-pinned action.

That has stayed open since M3, and `buf` joined it at M4 by the same route.

### Why a version is not a digest

The gap is not hypothetical and it is not about the projects being untrusted.
**A GitHub release asset can be deleted and re-uploaded under the same tag.** So
`gitleaks_8.30.0_linux_x64.tar.gz` names an artifact that can change with no
commit in this repository and no change to `mise.toml` — which is precisely the
mutation ADR-0014 removed for actions, left in place on the step that installs
the tools every later attestation depends on.

`gitleaks` is the secret scanner and `buf` validates the contract surface. A
substituted binary for either reports clean on material it was told to report
clean on, and nothing downstream would contradict it.

### A second gap, found while measuring the first

The `release-tags` ruleset covers `refs/tags/libs/*/v*` and nothing else,
verified against the live API. Two consequences:

- **`apps/*/v*` is unprotected**, and
  [ADR-0039](0039-applications-are-released-as-signed-binaries.md) proposes
  attesting build provenance against exactly those tags.
  `docs/branch-protection.md` already states why that matters — "a moved release
  tag makes every provenance attestation pointing at it meaningless" — and made
  the argument only for the tags it happened to cover.
- **`ecosystem/*` is unprotected**, and this one is already load-bearing.
  `fdos-connectors` vendors `invariants.md` and `boundary.md` pinned to
  `ecosystem/v0.1.0` and **byte-compares them** (`fdos-connectors:ADR-0026`).
  A movable tag there means another repository's byte-comparison anchor can be
  changed underneath it, from here, with nothing to report the change. Four such
  tags existed and all four were movable.

## Decision

### 1. Every build input downloaded by URL is pinned by digest

`tool-checksums.txt` sits beside `mise.toml` and records
`<tool> <version> <platform> <sha256>` for each artifact CI fetches.
`scripts/tool-checksum.sh` is its single parser, the digest counterpart of
`scripts/tool-version.sh`: CI resolves the version through one script and the
digest through another, and declares neither.

**The version is not a parameter of the digest lookup.** It is read from
`mise.toml`, so asking for a digest is asking for the *pinned* version's digest.
There is no way to verify one version's artifact against another version's
entry.

### 2. The comparison happens before the artifact is used

The digest is resolved before the download and compared before anything is
extracted or made executable. Verifying afterwards would mean the artifact had
already been unpacked onto `PATH`, which is most of what an attacker needed.

### 3. `make toolchain-checksum-check` enforces three rules, not one

A digest can fail to be a pin in three distinct ways, and checking only the first
would produce a check that is green while the property is absent:

1. an artifact is downloaded with no digest recorded;
2. a digest is recorded but never compared — decoration, which reads in review
   exactly like a pin;
3. a digest is recorded for a version that is no longer pinned — stale, and it
   satisfies rule 1 while describing an artifact nobody installs.

Rule 3 is what makes a version bump safe: changing `mise.toml` without recording
the new digest is a build failure rather than a silent loss of the pin.

### 4. The tag ruleset covers every namespace whose tags are attested or pinned

`release-tags` now includes `refs/tags/apps/*/v*` and `refs/tags/ecosystem/*`
alongside `refs/tags/libs/*/v*`, with the same three rules — no deletion, no
update, no force push.

### 5. What this does not decide

**Digests are recorded only for the platforms CI downloads.** Recording them for
artifacts nothing fetches would be coverage nobody verifies. A macOS or arm64
runner would need its entries added, and the check would demand them.

**This does not make the projects trustworthy.** A digest fixes *which* bytes are
installed, not whether those bytes are good. Verifying against the artifact
rather than only against the project's published manifest — which is what was
done here — proves transport and re-upload integrity and nothing more, because
the manifest and the binary come from the same place.

**No automation for digest updates.** ADR-0014 declined automatic pin updates and
that reasoning is unchanged; a digest bump is a reviewed commit.

## Consequences

### Positive

- The claim that every build input is identified by digest is now true. It was
  stated in `.github/README.md` from M0 and was false for two inputs.
- A re-uploaded release asset fails the build instead of silently entering it.
- A version bump cannot quietly lose its pin: rule 3 turns the omission into a
  red gate.
- The cross-repository corpus anchor stops being movable. That exposure existed
  in a form another repository depends on and nobody had looked.

### Negative

- **Bumping a pinned tool is now a two-file change**, and the second file is easy
  to forget. Rule 3 converts forgetting into a build failure rather than a
  silent regression, which is the mitigation and not a removal of the cost.
- **Two places now describe the toolchain**, against this repository's usual
  preference for one. `mise.toml` cannot hold digests without inventing a schema
  `mise` does not define, and a second file whose staleness is mechanically
  detected was judged the smaller cost than a file `mise` might reject.
- **The digest is only as good as the recording.** It is checked in review and by
  nothing else; a wrong digest recorded at the outset would be enforced faithfully
  forever. The procedure in `tool-checksums.txt` says to verify against the
  artifact, and that is rung 6.
- **The `sha256sum` comparison is inline in the composite action**, which is the
  narrow tool-installation exception ADR-0014 carved out, not a new licence for
  logic in YAML.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A downloaded artifact has a recorded digest | 3 | `make toolchain-checksum-check`, negative-tested |
| The digest is compared, not merely recorded | 3 | same check, rule 2, negative-tested |
| A digest describes the pinned version | 3 | same check, rule 3, negative-tested |
| The artifact matches before it is used | 3 | `sha256sum -c` in `setup-toolchain`, before extraction |
| Release and corpus tags cannot move | 3 | `release-tags` ruleset, negative-tested against the live repository |
| The recorded digest is the right digest | 6 | review, against the procedure in `tool-checksums.txt` |

All four checker rules were negative-tested: a removed entry, a removed
comparison, a version bumped without a digest, and a malformed digest each fail
naming what was violated. The ruleset was tested by attempting to move
`ecosystem/v0.3.1`, which was accepted before this change and is refused after.

## Alternatives considered

**Install gitleaks and buf through SHA-pinned actions instead.** ADR-0014's own
second suggestion, and it would remove the download entirely. Rejected because it
moves the trust rather than removing it — the action becomes responsible for
which bytes arrive, and the digest of the *artifact* is what an auditor asks
about. It also adds two more third-party actions to a repository that pins them
because it does not want more.

**Put the digests in `mise.toml`.** One file, which is this repository's stated
preference and the reason `tool-version.sh` exists. Rejected: `mise` defines the
schema of that file, an unknown table is at its discretion to accept, and a
toolchain manifest that a version of `mise` starts rejecting would break
`mise install` for everyone to save one file.

**Verify after installation, by running the tool and checking its version.**
Cheap, and it is what the steps already did. Rejected: it proves the binary
answers `--version` the way the expected one would, which a substituted binary
would also do. It is a liveness check wearing the costume of an integrity check.

**Leave it, and record the gap as ADR-0014 did.** Not a straw man — the entry was
honest, and honesty about an open gap is worth more than a bad fix. Rejected
because the fix is small, ADR-0014 named it, and the gap has outlived four
milestones with the repository's central claim resting on it.

## Notes

- The two digests were produced by downloading the artifacts and hashing them,
  not by copying the projects' published manifests. Both matched their manifest,
  which is the expected result and is recorded because "matched" and "copied" are
  indistinguishable afterwards.
- `docs/branch-protection.md` still has no mechanism comparing the live rulesets
  to what it documents. This ADR extends what is documented and does not close
  that gap; ADR-0020 records it, and
  [issue #106](https://github.com/FabioCaffarello/fdos/issues/106) proposes a
  check run from the maintainer's own CLI rather than from CI — reading rulesets
  needs the admin-scoped token ADR-0014 declined.
