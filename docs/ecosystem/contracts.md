# Contract registry

Every contract `fdos` publishes, its current version, its status, who consumes
it, and its deprecation state. This file is how a consuming repository discovers
what exists without asking anyone. Keeping it accurate is not documentation
work; it *is* the interface (E3).

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
| `fdos.ingest.v1` | `github.com/FabioCaffarello/fdos/libs/contracts` | `v0.5.0` | published | none yet | — |

Tag form is `libs/contracts/vX.Y.Z` — Go requires a submodule tag to carry the
module's directory prefix, so the tag names the path, not the concept.

### Version history

| Version | What a consumer gained | Breaking |
|---|---|---|
| `v0.1.0` | The initial kernel and ledger surface | — |
| `v0.2.0` | Schema and kernel additions for the ledger codec | no |
| `v0.3.0` | `kernel.v1.IdentifierClaim`, `ledger.payload.v1.HoldingClaimed`, `ledger.payload.v1.EntityMinted` — the shape a connector can populate without minting an identity ([ADR-0022](../adr/0022-minting-an-identity-is-a-fact.md)) | no |
| `v0.4.0` | The `SourceRef` grammar, the `collected_at` correction, and the reserved `unmediated` interpreter — documentation only ([ADR-0028](../adr/0028-provenance-admissibility.md)) | no |
| `v0.5.0` | `ingest.v1.HoldingClaimSubmission` — the first shape a producer outside FDOS can fill ([ADR-0030](../adr/0030-the-submission-shape.md)) | no |
| `v0.6.0` | `kernel.v1.RoundingContext` carries a scale, so a rounded amount states the minor units it was rounded to rather than leaving them inferred | no |

No breaking change has been published. `buf breaking` gates every pull request
(`make proto-check`), so the first one will be deliberate.

## The governance corpus

Not a contract — nothing imports it — but a versioned artifact this repository
publishes and another vendors, so it is discovered the same way.

| Artifact | Version | Status | Consumers |
|---|---|---|---|
| `docs/ecosystem/` | `ecosystem/v0.3.1` | published | `fdos-connectors` — last pinned `v0.1.0`, two versions behind ([fdos-connectors#18](https://github.com/FabioCaffarello/fdos-connectors/issues/18), mirror [fdos#32](https://github.com/FabioCaffarello/fdos/issues/32)) |

The tag does not match `libs/*/v*` and so does not trigger `release.yml`. There
is nothing to build, sign or attest: the artifact is the tree at the tag.

A version bump is a migration for everyone who vendors it, which is what the
tag is for — an unpinned vendor cannot tell a deliberate upstream change from an
accidental one.

## Published, and not offered

These are Go modules under the same licence, published because
[ADR-0004](../adr/0004-module-granularity.md) makes every `libs/*` an
independent module and every release tags one. Publication is a consequence of
that decision, **not an offer**.

| Module | Version | Offered | Consumed externally |
|---|---|---|---|
| `libs/kernel` | `v0.9.0` | no | no |
| `libs/ledger` | `v0.8.0` | no | no |
| `libs/kernel-wire` | `v0.3.0` | no | no |
| `libs/ledger-wire` | `v0.4.0` | no | no |
| `libs/ledger-sqlite` | `v0.3.0` | no | no |

This table went stale for four milestones, and ADR-0024 calls the registry part
of the interface rather than documentation about it — so the drift was a defect,
not untidiness. It listed `libs/kernel` at `v0.5.0` while `apps/submitd` pinned
`v0.7.0`, `libs/ledger` at `v0.2.0` against a pinned `v0.5.0`, and omitted
`libs/ledger-sqlite` entirely although it is published, tagged and imported.
Nothing in the gate compares the table against the tags, which is why: the
mechanism is rung 6 and this is what rung 6 costs.

Four releases carry
[ADR-0040](../adr/0040-encoding-integrity-and-the-fdos-root-namespace.md), and
each is the class of change `buf breaking` cannot see, because no schema moved:

- **`libs/kernel/v0.8.0`** changes *every identifier the package derives* — a new
  root namespace and a framed pre-image — while adding rather than removing API.
- **`libs/ledger/v0.6.0`** stops flattening a claim when it mints and resolves,
  which closed the collision the framing made closable.
- **`libs/ledger/v0.7.0`** adds the sub-second case to the shared store suite.
- **`libs/ledger-sqlite/v0.2.0`** stores temporal columns as integer nanoseconds
  and records the format in `PRAGMA user_version`. **It refuses a database written
  by `v0.1.0`**, which is the only breaking change in the set and is deliberate:
  a store this build cannot order is one it must not answer an as-of query from.

All four are safe now only because no store exists holding an older identifier,
and `v0.1.0` of the store being published means that safety is unverifiable
rather than guaranteed.

**`libs/ledger/v0.8.0`** carries
[ADR-0041](../adr/0041-the-write-path-serialises-in-the-store.md): `app.Store`
gains `Serialise`, and the write path reads the clock inside it.

It is the same invisible class as the four above, and worse in one respect —
this one **breaks every implementation of `app.Store`**, including any out of
tree, and no mechanism here can see it. `buf breaking` cannot, because no schema
moved. `make verify` cannot, because `FOR_EACH_MODULE` runs `GOWORK=off` and
resolves siblings from the proxy, so `libs/ledger-sqlite` is compiled against
the *previous* release of this module and passes
([#79](https://github.com/FabioCaffarello/fdos/issues/79)). A workspace build is
what says otherwise, and `make verify` does not run one.

So this release is deliberately published into a tree that does not build as a
workspace. That is step one of ADR-0041's release sequence rather than an
accident: `libs/ledger-sqlite` cannot implement the method until this version is
resolvable on the proxy, and the conformance suite that defines what the method
means ships inside this module.

> This entry originally ended *"the next release closes it"*. **That was wrong
> and the error is worth keeping**, because it is the shape of mistake this
> registry exists to catch. Publishing `libs/ledger-sqlite/v0.3.0` makes the
> gap *closable*; what closes it is `apps/submitd` and `examples/ingest`
> adopting both pins, which is a separate step. A release enables an adoption
> and is not one — and a registry that conflates the two would have this repo
> believing a gap was shut while a workspace build still failed.

**`libs/kernel-wire/v0.3.0`** carries the scale across the wire: the codec now
maps `RoundingContext`'s scale, added to the schema at
[`libs/contracts` v0.6.0](#version-history), so a rounded amount states the minor
units it was rounded to instead of leaving them inferred. Additive on both
sides — an older reader ignores a field it does not know, and the round-trip
conformance suite gained the cases that prove nothing is dropped in either
direction.

It is also **the first release cut through the dispatched path**
([ADR-0046](../adr/0046-publishing-a-tag-is-a-dispatched-act.md),
[ADR-0047](../adr/0047-a-release-carries-what-the-module-publishes.md)), so it
is the first tag whose release carries the module's own zip rather than a
linter's binaries.

**`libs/ledger-sqlite/v0.3.0`** is step two: it adopts `libs/ledger/v0.8.0` and
implements `Serialise` as a `BEGIN IMMEDIATE` transaction held open across the
caller's callback, so the clock read happens inside SQLite's write lock.

It is breaking in the only way that matters here — a consumer holding
`libs/ledger/v0.7.0` cannot use it, and nothing reports that beyond a compile
error in their build. What it buys is measured rather than argued: with each
process holding its own `Ledger` over one database, 128 concurrent admissions to
one stream admitted 127 at two processes and 106 at sixteen before this, and
**128 at every process count after**. ADR-0041 predicted exactly that, on the
engine already shipped and with no second engine involved, which is why
[ADR-0042](../adr/0042-postgresql-is-the-second-engine.md) is a capacity and
topology decision rather than a correctness one.

The region is per *database* rather than per stream, because SQLite has one
writer per file. That is recorded in ADR-0041 as the cost ADR-0042's per-stream
advisory locks exist to pay down.

**They carry no compatibility promise across versions.** A consumer importing
one is depending on FDOS's internal structure rather than on its contract: a
rename or a changed constructor signature breaks that consumer while breaking no
contract, and `buf breaking` cannot see it because there is nothing protobuf
about it.

Decided by [ADR-0025](../adr/0025-consumer-facing-surface-is-the-contracts-module.md).
It is **rung 6** — Go cannot express "published but not offered", so nothing
reports an import that ignores this. If you are reading this registry to decide
what to depend on, that decision is here rather than in a compiler error.

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

## Pricing a contract change

**Price it in the units of the contract, never in the units of the consumer's
generated code.** The generated code is where a change *appears*; it is not
where it *costs*.

A field rename reads as one mechanical edit per call site — `GetValue()` becomes
`GetContentHash()`, and `go build` finds every one. Priced that way it looks
almost free. Priced in contract units it is a wire break, a wire break is a new
package path (ADR-0024), and a new package path is the most expensive thing this
programme knows how to do: two published packages, both maintained through N-1,
two imports in every consumer, and the major-version slot spent.

That mispricing has happened once, in the open, and `buf breaking` caught what
the argument did not — the third time a gate has caught something reasoning
missed.

## Verifying a release

**Every version published so far — `v0.1.0` through `v0.3.0` — has no
supply-chain evidence.** The release workflow failed on all fourteen tags and
produced no releases (B-008). Those tags remain resolvable through the Go module
proxy, which is what makes a build work, and that is all they offer. They are
not back-filled: attaching provenance to a version after the fact is a decision
about what an attestation means, not a repair.

**The pipeline now works, proven end to end** by a disposable tag rather than
asserted. From the next tag onwards a release carries:

| Artifact | What it answers |
|---|---|
| `SHA256SUMS` | what the released bytes are |
| `SHA256SUMS.bundle` | who signed them — cosign keyless bundle: signature, certificate and transparency-log entry together |
| build-provenance attestation | which workflow run, from which commit, built them |
| `*.spdx.json` | what went into the binary |

Verifying the manifest, which is what
[fdos#26](https://github.com/FabioCaffarello/fdos/issues/26) asked for:

```sh
cosign verify-blob \
  --bundle SHA256SUMS.bundle \
  --certificate-identity-regexp '^https://github\.com/FabioCaffarello/fdos/\.github/workflows/release\.yml@' \
  --certificate-oidc-issuer https://token.actions.githubusercontent.com \
  SHA256SUMS
```

The identity is the workflow itself, bound by an OIDC token. There is no private
key to leak and no key custody to get wrong — but note what it proves: that
*this workflow in this repository* produced those bytes. It says nothing about
whether the commit it built was authored by anyone in particular. Commit signing
is separate and still blocked (B-006).
