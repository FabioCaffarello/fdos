# FDOS — Financial Data Operating System

FDOS stores immutable financial facts, never financial state.

Every position, balance, performance metric and recommendation it produces is
derived — reproducibly, deterministically, and with full provenance — from an
append-only ledger of events. A report generated today must be regenerable
years from now, byte for byte, from the same ledger and the same versioned
reference data.

That constraint is the whole design. Everything else follows from it.

> **Status: M6 and M7 complete.** M6 built the Ledger as a vertical slice; M7
> added the codecs and round-trip conformance suites that keep each canonical
> type's domain and wire definitions from diverging.
> Six principles reached **rung 1** — the type system. A position cannot be
> stored, a stream cannot be shortened, a fact cannot omit provenance, a query
> cannot omit its as-of, and a projection returns `Explained[Position]` or
> nothing at all.
>
> M5's outstanding acceptance criterion is now **met**: `fdos-connectors` builds
> against `libs/contracts` at a pinned version with no filesystem path
> dependency. What remains untested is the *private*-module resolution path. See
> [`docs/blocked.md`](docs/blocked.md) — B-001.
>
> **Next: M8, ingestion** — the path by which a fact produced outside FDOS
> enters the ledger. Its gating deliverable is a decision, not code: what a
> `SourceRef` must resolve to. See
> [`docs/ecosystem/roadmap.md`](docs/ecosystem/roadmap.md).

## Quick start

```sh
make doctor      # what is installed, what is missing, what to do about it
make bootstrap   # validate the toolchain, install git hooks
make verify      # run every enforcement mechanism available at this milestone
make help        # list available targets
```

Or open the repository in the [devcontainer](.devcontainer/README.md): it
installs `mise`, `mise` installs the toolchain from `mise.toml`, and nothing
declares a version twice.

`make verify` is the whole gate. CI runs exactly this and nothing else
(ADR-0014), so a green local run is a meaningful prediction of a green pipeline.
Git hooks run a fast subset on commit and the full gate on push; they are
bypassable, because CI re-runs everything regardless.

A clean clone must pass `make verify` with no tribal knowledge. If it does not,
that is a bug in this repository, not in your machine.

## Principles

The [Engineering Constitution](docs/constitution.md) is the highest authority
here. Fourteen principles govern FDOS; the ones that shape most decisions are:

- **Financial truth is immutable.** Facts are appended, never overwritten.
  Corrections are new events. State is always derived.
- **Everything is deterministic.** Business rules, calculations and reports are
  reproducible. LLMs explain, summarise and prioritise — they are never the
  source of financial truth.
- **The canonical model comes first.** No provider concept reaches the domain
  without normalisation. Provider-specific detail never leaks in.
- **Provenance is never lost.** Source, timestamps, parser version,
  transformation history and confidence travel with every datum.
- **Infrastructure serves the domain.** The domain depends on no database, no
  broker, no framework, no API. Infrastructure is always replaceable.

## The enforcement ladder

A principle stated in prose is a wish. FDOS enforces every principle at the
highest feasible mechanism (ADR-0005):

| Rung | Mechanism | Fails at |
|------|-----------|----------|
| 1 | Type system | authoring time |
| 2 | Static analysis | build |
| 3 | CI | pull request |
| 4 | Automated review | review |
| 5 | Documentation | never, automatically |
| 6 | Human discipline | — |

Human discipline is the last line of defence, never the first. Constitution §15
records where every principle currently sits — including, honestly, the ones
still at rung 6.

As of M2, six principles reached their target rung. Determinism, canonical-model
purity and layer boundaries are enforced by static analysis; reproducibility by a
double-build diff. M3 and M2.5 raised no rungs — they made the existing
mechanisms run without anyone remembering to. The seven principles still
unenforced need types that do not exist yet, and climb at M4 and M6.

```
libs/ledger/domain/rule.go:14:37: nondet: time.Now in domain package;
  the clock is injected; a domain rule that reads it cannot be replayed (Constitution §2)
```

## Layout

| Directory | Contents |
|-----------|----------|
| `libs/` | Reusable libraries. One independent Go module per subdirectory. |
| `apps/` | Deployable applications. Composition roots only, no business logic. |
| `docs/` | Constitution, ADRs, RFCs. The authoritative record. |
| `deploy/` | Deployment topology and runtime environments. |
| `examples/` | Executable demonstrations of the public contract surface. |
| `scripts/` | Enforcement mechanisms, invoked through `make`. |
| `.github/` | CI workflows. Runs `make`, contains no logic of its own. |
| `.context/` | Structured engineering knowledge for AI agents (ADR-0006). |
| `.devcontainer/` | Reproducible dev environment. Declares no versions of its own. |
| `.vscode/` | Editor settings that mirror what `make` enforces. Nothing personal. |

Every top-level directory and every module under `libs/` declares its
architectural contract in the front matter of its `README.md`: what it permits,
what it forbids, and who owns it. `make contracts-check` enforces this from
commit #1 — though it only reached `libs/` at M7, having reported ten valid
contracts while the two modules holding the domain core had none.

The M0 plan was for those declarations to *generate* the import-boundary
configuration. M2 found a better answer: the layer rule turned out to be
structural rather than per-module, so it lives in the `layering` analyser, which
is tested rather than configured. The contracts stay the human record.

## Governance

[`CONTRIBUTING.md`](CONTRIBUTING.md) is the working guide. In short:

Any change to repository structure, module boundaries, the public contract
surface, the toolchain or the enforcement mechanisms requires an
[ADR](docs/adr/). Decisions requiring design exploration first go through an
[RFC](docs/rfc/).

The decision log obeys the same law as the ledger: append-only, immutable,
corrections recorded as new decisions that supersede the old (ADR-0000).

| ADR | Decision |
|-----|----------|
| [0000](docs/adr/0000-record-architecture-decisions.md) | Decisions are recorded as an append-only log |
| [0001](docs/adr/0001-context-management.md) | ~~`.dotcontext` is the canonical AI knowledge directory~~ — superseded by 0006 |
| [0002](docs/adr/0002-license.md) | The public core is Apache-2.0 |
| [0003](docs/adr/0003-module-path.md) | Modules publish under `github.com/FabioCaffarello/fdos` |
| [0004](docs/adr/0004-module-granularity.md) | One module per `libs/*`; CI builds with `GOWORK=off` |
| [0005](docs/adr/0005-enforcement-ladder.md) | Principles are enforced at the highest feasible mechanism |
| [0006](docs/adr/0006-context-directory-naming.md) | `.context` is the canonical AI knowledge directory |
| [0007](docs/adr/0007-internal-deterministic-identity.md) | Entity identity is internal, deterministic and assertion-based |
| [0008](docs/adr/0008-decimal-money-explicit-rounding.md) | Money is arbitrary-precision decimal; no division has a default rounding |
| [0009](docs/adr/0009-universal-bitemporality.md) | Every canonical fact is bitemporal; no query has a default as-of |
| [0010](docs/adr/0010-provenance-envelope-reference-versioning.md) | Every fact carries a provenance envelope and pins its reference-data versions |
| [0011](docs/adr/0011-fact-taxonomy-and-upcasting.md) | Facts are Occurrences or Observations; schemas evolve by upcast-on-read |
| [0012](docs/adr/0012-explained-return-type.md) | Domain calculations producing financial values return `Explained[T]` |
| [0013](docs/adr/0013-layer-structure-and-module-topology.md) | Modules follow bounded contexts; layers are packages within them |
| [0014](docs/adr/0014-ci-runs-make-and-pins-everything.md) | CI invokes `make` and nothing else; every build input is pinned by digest |
| [0015](docs/adr/0015-ai-engineering-policy.md) | AI-assisted work is bounded by enforced gates and a checkable knowledge base |
| [0016](docs/adr/0016-developer-experience.md) | One entry point, one source of truth, and no second copy of the rules |
| [0017](docs/adr/0017-claude-export-is-versioned.md) | ~~The Claude Code export is versioned~~ — superseded by 0019 |
| [0018](docs/adr/0018-contract-surface-is-protobuf.md) | The contract surface is protobuf, and wire types are never domain types |
| [0019](docs/adr/0019-claude-export-is-not-versioned.md) | The Claude Code export is not versioned |
| [0020](docs/adr/0020-open-core-boundary-and-pull-request-workflow.md) | The repository is named `fdos`; the boundary is proven; work moves to pull requests |
| [0021](docs/adr/0021-purity-rules-scope.md) | The purity rules cover the kernel and exempt test and generated files |
| [0022](docs/adr/0022-minting-an-identity-is-a-fact.md) | Minting an identity is a fact, and a connector emits a claim |
| [0023](docs/adr/0023-ecosystem-boundary-and-one-way-contract-flow.md) | The ecosystem boundary is written down, and contracts flow one way |
| [0024](docs/adr/0024-contract-lifecycle-and-versioning.md) | Contracts are versioned per module, and a breaking change is a process |

### Requests for Comment

The M1.5 canonical-architecture set. All **Accepted**, each recorded by the ADR
above that states what it settled. Read in order; each depends on the ones
above it.

| RFC | Proposal | Decided by |
|-----|----------|------------|
| [0001](docs/rfc/0001-identity-and-aggregate-boundaries.md) | External identifiers are claims about entities, never keys. Positions are projections, not aggregates. | ADR-0007 |
| [0002](docs/rfc/0002-money-and-numeric-representation.md) | Arbitrary-precision decimals; no binary float; no default rounding context. | ADR-0008 |
| [0003](docs/rfc/0003-bitemporal-event-model.md) | Universal bitemporality; no default as-of, making look-ahead bias impossible. | ADR-0009 |
| [0004](docs/rfc/0004-provenance-and-reference-data.md) | Provenance is structural, not optional. Reference datasets pinned by version — and so is the code. | ADR-0010 |
| [0005](docs/rfc/0005-event-taxonomy-and-schema-evolution.md) | Occurrences vs Observations. Events are never migrated; upcasting happens on read and is pinned. | ADR-0011 |
| [0006](docs/rfc/0006-explainability-as-a-return-type.md) | Calculations return their computation trace, so one that cannot explain itself does not compile. | ADR-0012 |

| [0007](docs/rfc/0007-identity-resolution-and-the-acquisition-boundary.md) | Minting an identity is a fact; a connector emits a claim and resolution is a recorded derivation. | ADR-0022 |

## Roadmap

| Milestone | Objective |
|-----------|-----------|
| M0 | Repository genesis — governance and enforcement substrate ✅ |
| M1 | Governance substrate — `.context`, contribution and release process ✅ |
| M1.5 | Canonical domain architecture — RFCs only: identity, numerics, bitemporality, provenance, event taxonomy, explainability ✅ |
| M2 | Determinism toolchain — layer boundaries, custom analysers, reproducible builds ✅ |
| M3 | CI/CD and supply chain — pipeline, SBOM, provenance attestation, signing ✅ |
| M2.5 | AI engineering — agent playbooks, prompt contracts, staleness checks ✅ |
| M3.5 | Developer experience — devcontainer, IDE configuration, task ergonomics ✅ |
| M4 | Contracts — protobuf schemas, `buf breaking` gate, generated Go SDK ✅ |
| M5 | Open core boundary — published contract module, consumer proof, branch protection ✅ |
| M6 | First domain — the Ledger, as a vertical slice validating everything above ✅ |
| M7 | Wire conformance — codecs and round-trip suites proving the domain and wire definitions of every canonical type agree ✅ |
| M8 | Ingestion — how a fact produced outside FDOS enters the ledger: claims resolved, identities minted, unresolved claims made visible |

## Open Core

FDOS follows an Open Core architecture. This repository is the public core —
engineering platform, canonical models, ledger, SDKs, APIs, documentation and
testing infrastructure — under Apache-2.0.

Authenticated providers, browser connectors and institution-specific plugins
live in separate private repositories. They depend on this one exclusively
through published, versioned contract modules — a boundary proven by every CI
run rather than assumed (ADR-0004).

Which repository owns which concern, what is published, and what is still
disputed live in [`docs/ecosystem/`](docs/ecosystem/boundary.md) — authoritative
for both sides of the boundary, not just this one (ADR-0023).

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE) for the open-core boundary.
