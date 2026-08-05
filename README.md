# FDOS — Financial Data Operating System

FDOS stores immutable financial facts, never financial state.

Every position, balance, performance metric and recommendation it produces is
derived — reproducibly, deterministically, and with full provenance — from an
append-only ledger of events. A report generated today must be regenerable
years from now, byte for byte, from the same ledger and the same versioned
reference data.

That constraint is the whole design. Everything else follows from it.

> **Status: M1 complete — Governance Substrate. Next: M1.5.**
> There is no domain code, and that is deliberate: the canonical model is an
> output of the M1.5 RFCs, and building before that design lands would pre-judge
> it. What exists today is the governance and enforcement substrate everything
> else will be held to.

## Quick start

```sh
make bootstrap   # validate the toolchain against the pins in mise.toml
make verify      # run every enforcement mechanism available at this milestone
make help        # list available targets
```

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

Every directory declares its architectural contract in the front matter of its
`README.md`: what it permits, what it forbids, and who owns it. `make
contracts-check` enforces this from commit #1, and from M2 those declarations
generate the import-boundary configuration rather than merely describing it.

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

## Roadmap

| Milestone | Objective |
|-----------|-----------|
| M0 | Repository genesis — governance and enforcement substrate ✅ |
| M1 | Governance substrate — `.context`, contribution and release process ✅ |
| **M1.5** | Canonical domain architecture — RFCs only: model, identifiers, aggregates, event taxonomy, bitemporality, reference data, explainability |
| M2 | Determinism toolchain — layer boundaries, custom analysers, reproducible builds |
| M3 | CI/CD and supply chain — pipeline, SBOM, provenance attestation, signing |
| M2.5 | AI engineering — agent playbooks, prompt contracts, staleness checks |
| M3.5 | Developer experience — devcontainer, IDE configuration, task ergonomics |
| M4 | Contracts and observability — proto → buf → OpenAPI → SDK → MCP → docs |
| M5 | Open core boundary — published contract modules, plugin conformance suite |
| M6 | First domain — the Ledger, as a vertical slice validating everything above |

## Open Core

FDOS follows an Open Core architecture. This repository is the public core —
engineering platform, canonical models, ledger, SDKs, APIs, documentation and
testing infrastructure — under Apache-2.0.

Authenticated providers, browser connectors and institution-specific plugins
live in separate private repositories. They depend on this one exclusively
through published, versioned contract modules — a boundary proven by every CI
run rather than assumed (ADR-0004).

## License

[Apache-2.0](LICENSE). See [NOTICE](NOTICE) for the open-core boundary.
