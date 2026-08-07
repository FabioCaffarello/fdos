---
id: ADR-0039
title: Applications are released as signed binaries, through one signing path rather than two
status: Proposed
date: 2026-08-07
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0039 — Applications are released as signed binaries, through one signing path rather than two

> **Proposed, not accepted.** `.github/**` is approval-gated and no workflow is
> touched by this ADR. It is the last slice of M11 and the decision is the
> human's.

## Context

[ADR-0037](0037-delivery-includes-a-service-the-adopter-operates.md) decided that
delivery includes a service **an adopter operates**. `apps/submitd` exists and is
on `main`. What has not been decided is how an adopter gets it.

### What already works, measured rather than assumed

Before proposing a release pipeline it is worth establishing whether one is
needed. It is not, for the narrow purpose of running the thing. From a clean
`GOBIN`, with nothing built here:

```
$ go install github.com/FabioCaffarello/fdos/apps/submitd@latest
go: downloading github.com/FabioCaffarello/fdos/apps/submitd v0.0.0-20260807181203-e0eb3e6111db

$ submitd
submitd: -store is required; there is no default database path

$ submitd -listen 0.0.0.0:8080 -store /tmp/x.db
submitd: refusing to listen off-loopback without -callers-are-authenticated

$ submitd -store /tmp/probe.db
INFO listening addr=127.0.0.1:8080 store=/tmp/probe.db
  POST  /v1/holding-claim-submissions  -> 400   (garbage body)
  GET   /v1/holding-claim-submissions  -> 405
```

**`E9` is met by that transcript.** A third party installs the ingress from
source, through the module proxy and its checksum database, and the guards fire
in the shipped binary rather than only in tests — including the D2 refusal, which
is the one that would have been embarrassing to discover was test-only.

So this decision is **not** "make the open core usable". It is already usable.
It is about what FDOS additionally attests to.

### Why a binary release is still worth deciding

The release workflow states its own rationale for the one artifact it produces:

> The only artifact FDOS produces today is `fdoslint` […] It is released so that
> a consumer can verify the tool gating their code was built from the source it
> claims.

That argument transfers, and strengthens. `fdoslint` gates somebody's code;
`submitd` **admits their financial facts** and holds the ledger's write path. An
adopter performing due diligence on an open-core truth engine asks the same
question about the ingress that FDOS thought worth answering about a linter.

`go install` gives integrity of *source* — the checksum database proves the
module is what the proxy served. It does not produce a signed artifact, an SBOM,
or a provenance attestation binding a binary to the workflow that built it.

### The constraint that shapes the answer

The release workflow is the one that **failed on all fourteen tags** and shipped
nothing while looking like it had (B-008). It is entirely hardcoded to
`libs/analysis` and `fdoslint`, it triggers on `libs/*/v*` only, and one of its
steps — `make consumer-check`, which proves a published *module* is importable —
is meaningless for an application nobody imports.

So the question is not only *whether* to release applications. It is **how to add
a second artifact type to a signing path with that history without putting
library releases at risk**, and those releases are the ones an external
repository already depends on.

## Decision

### 1. FDOS releases applications as signed binaries

A tag matching `apps/<name>/vX.Y.Z` produces the same evidence a library tool
does: cross-platform binaries, a `SHA256SUMS` manifest signed with keyless
`cosign`, an SPDX SBOM, and a build-provenance attestation.

`go install` remains the supported path and is not deprecated by this. The
release adds attestation; it does not replace source distribution, and the
transcript above stays true.

### 2. One signing path, extracted rather than duplicated

The build-sign-attest-publish sequence moves into a **composite action** under
`.github/actions/`, parameterised by the module directory, the binary name and
the package to build. Both the library and the application release use it.

This is the repository's own precedent and its stated reason. `setup-toolchain`
exists because:

> both workflows use one definition of "the toolchain", so they cannot drift
> from each other either (B-008)

A second copy of a signing pipeline is the same defect with worse consequences:
the copy that drifts is the one nobody is watching, and B-008 is the evidence
that a broken release path here looks exactly like a working one.

### 3. `consumer-check` stays on the library path only

Proving a module is resolvable from the proxy is a statement about a published
*module* ([ADR-0020](0020-open-core-boundary-and-pull-request-workflow.md)).
Applications are not imported, so the step is not generalised and not skipped
conditionally inside a shared action — it stays in the library workflow, above
the shared step.

### 4. What this does not decide

**No versioning policy for applications.** Whether `submitd` starts at `v0.1.0`,
and how its version relates to the modules it pins, is a release decision and not
a pipeline one.

**No container image, no package manager, no installer.** Each is a distribution
channel with a support obligation, and ADR-0037 already recorded that publishing
a service means owning its ergonomics for adopters this repository will never
meet. One channel at a time.

## Consequences

### Positive

- An adopter can verify the ingress binary was built from the source it claims,
  which is the question due diligence asks about a truth engine's write path.
- **One signing path.** A change to how FDOS signs cannot land in one workflow
  and miss the other, which is the specific failure `setup-toolchain` was
  extracted to prevent.
- `repro-check` already covers `submitd` — it discovered the binary with no
  configuration and proves it builds reproducibly. The release attests what
  `verify` already checks rather than introducing a new claim.

### Negative

- **It touches the workflow with the worst track record in the repository**, to
  add a second consumer of it. B-008 shipped fourteen empty releases; the
  extraction is exactly the kind of change that could do it again, and **the
  blast radius includes library releases an external repository depends on**.
  That is the cost, it is the reason for §2 rather than a second workflow, and
  it is not eliminated by either choice.
- **Nothing tests a release workflow except releasing.** There is no `make`
  target for it and this ADR does not add one; the extraction is verified by
  performing a release, which is a permanent tag. A disposable tag proved B-008
  fixed and is the same instrument available here.
- **A second support surface.** A published binary is something adopters pin,
  report against and expect to keep working across platforms this repository
  does not run.
- **`E9` is not advanced by this.** It is already met, by `go install`, and the
  transcript in §Context is the receipt. This ADR buys attestation, and stating
  that plainly is the honest scope — a reader should not come away thinking the
  open core became usable here.
- **Four platform binaries per application**, on every tag, with SBOM and
  attestation for each. Release time and artifact storage grow with the number of
  applications, which is one.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A released binary was built from the tagged source | 3 | the workflow runs `make verify` on the tagged commit before building |
| The released bytes are what the manifest says | 3 | `SHA256SUMS`, signed with keyless `cosign` as a bundle |
| The binary builds reproducibly | 3 | `make repro-check`, which already covers `submitd` |
| The signing path cannot drift between artifact types | **2** | one composite action; there is no second copy to diverge |
| The release workflow works | **6** | performing a release. Nothing else runs it |

**Execution-context question.** The last row is the honest weak point and it is
inherited rather than introduced: B-008 is what happens when nothing exercises a
release path, and this ADR adds a consumer to that path without adding a way to
exercise it. The mitigation available is the one that proved B-008 fixed — push
a disposable tag, inspect the artifacts, and confirm before announcing anything.

## Alternatives considered

**No binary release; `go install` is the distribution.** The cheapest, and it is
not a straw man — the transcript in §Context is the argument for it, and `E9` is
satisfied without any of this. Rejected because the repository already decided
this question for a linter, on a rationale that applies with more force to the
component that admits financial facts. Choosing differently for the ingress would
mean the tool gating your code is attested and the service holding your ledger is
not.

**A separate workflow for applications.** Isolates the risk to library releases
entirely, which is a real property given B-008. Rejected: two copies of a signing
pipeline drift, the drifted copy is the one nobody watches, and the repository
already extracted `setup-toolchain` rather than accepting exactly this trade.

**Generalise the existing workflow in place, with conditionals on the tag
shape.** No new file, but every step acquires an `if` and the two artifact types
become readable only by mentally executing the conditionals. Rejected on
readability of a path whose failures are silent.

**Container images instead of binaries.** A more common deployment shape for a
service. Rejected as premature and as a second support surface: it needs a
registry, a base-image policy and a rebuild cadence for base-image CVEs, and
ADR-0038 has just recorded what acquiring an external cadence costs.

## Notes

Proposed by the session; **not accepted**, and no workflow file is touched.
`.github/**` is approval-gated, which is why this stops here.

Open and deliberately not decided:

- the version an application's first tag carries;
- whether a disposable tag should be pushed to verify the extraction before a
  real release, and who pushes it;
- whether `submitd` should be released at all *before* D2
  ([#64](https://github.com/FabioCaffarello/fdos/issues/64)) is answered — a
  signed, attested ingress is a stronger invitation to run one than source is,
  and the thing it invites is exactly what D2 has no answer for.

That last one is the question this ADR would most like a human to look at.
