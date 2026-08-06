# Contract registry

Every contract `fdos` publishes, its current version, its status, who consumes
it, and its deprecation state. This file is how a consuming repository discovers
what exists without asking anyone. Keeping it accurate is not documentation
work; it *is* the interface (I3).

Lifecycle and versioning rules:
[ADR-0024](../adr/0024-contract-lifecycle-and-versioning.md).

## Schema

| Field | Meaning |
|---|---|
| `package` | Protobuf package, `fdos.<context>[.<sub>].<version>` |
| `module` | Go import path that carries the generated code |
| `version` | Current published version — a git tag, resolvable through the Go module proxy |
| `status` | `draft` · `published` · `deprecated` · `removed` |
| `consumers` | Repositories known to depend on it, and at which version |
| `deprecation` | Removal milestone, or `—` |

A package marked `draft` is **not implementable** in any repository. Building
against draft content converts a hypothesis into a decision without an ADR.

## Published

All three packages ship in one Go module. They version together, because
`fdos.ledger.v1.Fact` references `fdos.kernel.v1.Provenance` and a payload is
only meaningful inside a fact — splitting them would create three tags that can
never legally disagree.

| package | module | version | status | consumers | deprecation |
|---|---|---|---|---|---|
| `fdos.kernel.v1` | `github.com/FabioCaffarello/fdos/libs/contracts` | `v0.3.0` | published | `fdos-connectors` @ `v0.3.0` | — |
| `fdos.ledger.v1` | `github.com/FabioCaffarello/fdos/libs/contracts` | `v0.3.0` | published | `fdos-connectors` @ `v0.3.0` | — |
| `fdos.ledger.payload.v1` | `github.com/FabioCaffarello/fdos/libs/contracts` | `v0.3.0` | published | `fdos-connectors` @ `v0.3.0` | — |

Tag form is `libs/contracts/vX.Y.Z` — Go requires a submodule tag to carry the
module's directory prefix, so the tag names the path, not the concept.

### Version history

| Version | What a consumer gained | Breaking |
|---|---|---|
| `v0.1.0` | The initial kernel and ledger surface | — |
| `v0.2.0` | Schema and kernel additions for the ledger codec | no |
| `v0.3.0` | `kernel.v1.IdentifierClaim`, `ledger.payload.v1.HoldingClaimed`, `ledger.payload.v1.EntityMinted` — the shape a connector can populate without minting an identity ([ADR-0022](../adr/0022-minting-an-identity-is-a-fact.md)) | no |

No breaking change has been published. `buf breaking` gates every pull request
(`make proto-check`), so the first one will be deliberate.

## Published, but not the contract surface

These are Go modules under the same licence, published because
[ADR-0004](../adr/0004-module-granularity.md) makes every `libs/*` an
independent module. They are **not** part of what a consumer is invited to
depend on: [ADR-0018](../adr/0018-contract-surface-is-protobuf.md) says the
contract surface is protobuf, and these are domain code and codecs.

| Module | Version | Consumed externally |
|---|---|---|
| `libs/kernel` | `v0.5.0` | no |
| `libs/ledger` | `v0.2.0` | no |
| `libs/kernel-wire` | `v0.2.0` | no |
| `libs/ledger-wire` | `v0.2.0` | no |

Whether a consumer *may* import them is not written down anywhere — it is only
true that none does. Recorded as D5 in [`boundary.md`](boundary.md).

`libs/analysis` is not published at all: it is tooling, and nothing outside this
repository has reason to link it.

## Not published, and frequently assumed to be

**There is no `fdos.acquisition.v1`.** No `AcquisitionEnvelope`, no
`ProviderObservation`. The governance brief anticipates both, and a reader
arriving from that brief will look for them.

What exists instead is described in [`roadmap.md`](roadmap.md): the connector
hand-off happens through `ledger.payload.v1.HoldingClaimed` plus a host↔plugin
contract that `fdos-connectors` owns. Whether `fdos` should also define an
acquisition envelope is D4 and D5 in [`boundary.md`](boundary.md), not a
scheduled deliverable.

## What a release does *not* currently carry

Stated because the alternative is a consumer assuming otherwise.

Every tag is resolvable through the Go module proxy — that is what makes
`fdos-connectors` build, and it works. Nothing else does. **All fourteen release
workflow runs have failed, and zero GitHub releases exist**, so no published
version carries an SBOM, a build-provenance attestation, a cosign signature, or
release notes.

The cause is a defect in `.github/workflows/release.yml`, not a design
limitation. Recorded as B-008 in [`../blocked.md`](../blocked.md).
