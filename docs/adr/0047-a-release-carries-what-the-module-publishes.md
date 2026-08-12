---
id: ADR-0047
title: A release carries what the module publishes, and the release path is rehearsable
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0047 — A release carries what the module publishes, and the release path is rehearsable

## Context

[ADR-0039](0039-applications-are-released-as-signed-binaries.md) is accepted as
of today. It asked for one signing path, extracted into a composite action and
parameterised by module directory, binary name and package, so that library and
application releases could not drift apart.

Building it found that the path was not signing what anyone would assume.

### Every library tag published a linter

`release.yml` triggered on `libs/*/v*` and hardcoded `libs/analysis` and
`fdoslint`. Verified against what is actually published:

```
$ gh release view libs/kernel/v0.9.0 --json assets
fdoslint.spdx.json, fdoslint_darwin_amd64, fdoslint_darwin_arm64,
fdoslint_linux_amd64, fdoslint_linux_arm64, SHA256SUMS, SHA256SUMS.bundle

$ gh release view libs/ledger-sqlite/v0.3.0 --json assets
… the same seven files.
```

So the release for `libs/kernel/v0.9.0` carries four `fdoslint` binaries, an SBOM
named `fdoslint.spdx.json`, and a **signed** `SHA256SUMS` describing none of the
kernel. The provenance attestation attached to a kernel tag attests a linter.

**Every one of these statements is true and none of them is about the module the
tag names.** An adopter who verifies the signature on a kernel release is
verifying a linter, and the signature is real — which makes this worse than an
unsigned release, because it looks like evidence. ADR-0039 was written to prevent
two signing paths drifting; it did not ask whether the one path was pointed at
the right thing.

### The artifact anyone consumes was the one not attested

ADR-0014 left this open in its notes:

> SLSA provenance is attested for `fdoslint` only. Go libraries are released as
> tags served by the module proxy and have no artifact to attest; whether the
> module zips themselves should be attested is unresolved.

`libs/contracts` is pinned by an external private repository. That module's zip
is the thing another organisation's build actually consumes, and it carried no
attestation while the linter carried one.

Measured: the zip is retrievable and stable — `go mod download` yields
`v0.9.0.zip` with a fixed digest, which is exactly a subject an attestation can
name.

### Nothing exercised the release path

ADR-0039 stated its own weak point:

> **Nothing tests a release workflow except releasing.** There is no `make`
> target for it and this ADR does not add one; the extraction is verified by
> performing a release, which is a permanent tag.

That is the condition B-008 grew in — fourteen tags whose release published
nothing while looking fine. And the mitigation ADR-0039 offered, a disposable
tag, is now worse than it was: tags are immutable by ruleset
([ADR-0043](0043-downloaded-artifacts-are-pinned-by-digest.md)), so the residue
is permanent. `libs/release-smoke/v0.0.0-rc.1` and `rc.2` are still in the
namespace from the last drill.

## Decision

### 1. A release carries what the module publishes

`make release-artifacts` assembles `dist/` from what the module **is**, not from
the shape of its tag:

- **every `main` package the module contains**, cross-compiled for four
  platforms. A library with none produces no binaries and says so.
- **for a `libs/*` module, the zip the proxy serves**, fetched with
  `go mod download` rather than built here. Attesting a zip this workflow made
  would attest bytes nobody downloads; the bytes worth signing are the ones
  `go mod download` returns.

There is no conditional on the tag anywhere in the assembly. `apps/submitd` has a
`main` package and gets binaries because of that, not because it is under
`apps/`.

### 2. One signing path, and it signs whatever is in `dist/`

The composite action `release-evidence` generates the SBOM, regenerates the
manifest so it covers the SBOM too, attests provenance over `dist/*`, and signs
`SHA256SUMS` with keyless `cosign`. It is used by the release and by the
rehearsal, so there is one definition to drift and it drifts for both — which is
the extraction ADR-0039 asked for and the reason `setup-toolchain` exists.

The SBOM is named after the module. `fdoslint.spdx.json` in twenty releases
described the wrong thing twenty times.

### 3. `consumer-check` stays on the library path

Unchanged from ADR-0039 §3: proving a published *module* is importable is a
statement about a library. It is one `if` on one step in the release workflow,
rather than a conditional inside the shared action.

### 4. The release path is rehearsable, against an already published tag

`release-rehearse.yml` runs `make verify`, assembles artifacts, generates the
SBOM, attests, signs and runs `consumer-check` — every step the release takes —
then stops. No tag, no GitHub release; the evidence is uploaded to the run for
inspection.

**What it does not cover is stated rather than implied:** `gh release create`,
and the tag trigger itself. Those are exercised by the first real release. The
rehearsal reduces what that release discovers for the first time from everything
to two lines.

### 5. What this does not decide

**No versioning policy for applications**, unchanged from ADR-0039 §4.

**No back-fill.** The twenty existing releases carrying `fdoslint` are not
corrected. Tags are immutable and the releases attached to them are history; the
registry describes what each version is, and this ADR is the record of what those
release assets were.

**No container image, package manager or installer.** One channel at a time.

## Consequences

### Positive

- A release describes the module it names. The signature and the attestation are
  about the artifact the tag is for.
- The module zip — the artifact an external build consumes — is attested, which
  closes the inversion ADR-0014 recorded.
- `apps/*` can be released, which is what ADR-0039 asked for, without a second
  workflow or a second signing path.
- The release path can be exercised without leaving a permanent tag, which is
  the first time that has been true.

### Negative

- **This rewrote the workflow with the worst record in the repository.** B-008
  is what happens when that path is wrong and nobody notices; the rehearsal
  exists because of it and does not cover the last two steps.
- **The zip fetch depends on proxy propagation**, so a release now fails if the
  proxy has not seen the tag yet. `consumer-check` already had that dependency
  and it is why neither is in `make verify` — but the failure moves earlier, into
  artifact assembly, where it is less obviously a propagation problem.
- **The manifest is regenerated inside the composite action**, so the one
  `make release-artifacts` prints is not the one that ships. That is deliberate —
  the SBOM has to be covered — and it means the local command's output is
  indicative rather than final.
- **Twenty published releases remain wrong**, and nothing marks them. An adopter
  reading `libs/kernel/v0.9.0`'s assets today still finds a linter.
- **A rehearsal that passes proves less than it appears to.** It runs against a
  published tag, so it cannot exercise the first release of a module — which is
  precisely the case `apps/submitd` will be.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A release carries only the tagged module's artifacts | 3 | `make release-artifacts`, driven by `go list` rather than the tag |
| A library release attests the zip the proxy serves | 3 | `release-evidence` over `dist/*` |
| The manifest covers everything published | 3 | regenerated after the SBOM, inside the shared action |
| The signing path cannot drift between artifact types | **2** | one composite action; there is no second copy |
| The release path can be exercised | **4** | `release-rehearse.yml`, against a published tag |
| `gh release create` and the tag trigger work | **6** | performing a release. Still nothing else runs them |

The last row is what remains of ADR-0039's honest weak point. It is two steps
rather than the whole path.

## Alternatives considered

**Keep the hardcoded binary and add an `apps/` branch beside it.** The smallest
change, and it is what ADR-0039 literally describes. Rejected once the published
assets were read: it would have carried the defect into the application path and
signed `fdoslint` under `apps/submitd` tags too.

**Skip the release entirely for library tags with no binary.** Defensible — a Go
module release *is* a tag, and the proxy serves it. Rejected because it leaves
the consumed artifact unattested, which is the gap ADR-0014 named, and because a
release page is where an adopter looks for the SBOM.

**Build the module zip here rather than fetching it.** Removes the proxy
dependency and lets the first release of a module be attested. Rejected: a zip
built here is not the zip anyone downloads, and an attestation over bytes nobody
consumes is a statement about nothing.

**Delete the wrong assets from the twenty existing releases.** Tempting, and it
would stop an adopter finding a linter under a kernel tag. Rejected: the releases
are history attached to immutable tags, and rewriting what a published release
contained is the same class of act as moving a tag.

**A disposable tag for the rehearsal, as ADR-0039 suggested.** It is the proven
instrument — it is what proved B-008 fixed. Rejected now that tags cannot be
deleted: every rehearsal would leave permanent residue, and
`libs/release-smoke/v0.0.0-rc.1` and `rc.2` are the two already there.

## Notes

- The defect was found by reading what is published, not by reasoning about the
  workflow. `gh release view` on two tags took less time than this paragraph.
- `libs/analysis` still has no tag, so `fdoslint` has never been released under
  its own module's version — only under twenty other modules' versions. Its first
  correct release will also be the first release of that module.
