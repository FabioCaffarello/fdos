---
type: doc
name: architecture
description: Repository structure, intended layering, and the boundaries already binding
category: architecture
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Architecture

## What is decided and what is not

**Decided, binding today:** repository structure, module granularity, module
path, licence, decision process, enforcement philosophy (ADR-0000 through
ADR-0006) — and, since M1.5 closed, the canonical domain architecture
(ADR-0007 through ADR-0012).

**The M1.5 RFC set — all Accepted.** Each acceptance is recorded by an ADR
stating what it settled; the ADR is the binding statement, the RFC holds the
full design exploration.

| RFC | Decides | ADR |
|-----|---------|-----|
| [RFC-0001](../../docs/rfc/0001-identity-and-aggregate-boundaries.md) | Internal deterministic identity; external identifiers as timestamped assertions; positions are projections, not aggregates | [ADR-0007](../../docs/adr/0007-internal-deterministic-identity.md) |
| [RFC-0002](../../docs/rfc/0002-money-and-numeric-representation.md) | Arbitrary-precision decimals, `Money` carrying currency, no default rounding context | [ADR-0008](../../docs/adr/0008-decimal-money-explicit-rounding.md) |
| [RFC-0003](../../docs/rfc/0003-bitemporal-event-model.md) | Universal bitemporality; no default as-of on any query | [ADR-0009](../../docs/adr/0009-universal-bitemporality.md) |
| [RFC-0004](../../docs/rfc/0004-provenance-and-reference-data.md) | Structural provenance envelope; reference datasets and code pinned by version | [ADR-0010](../../docs/adr/0010-provenance-envelope-reference-versioning.md) |
| [RFC-0005](../../docs/rfc/0005-event-taxonomy-and-schema-evolution.md) | Occurrences vs Observations; upcast on read, never migrate | [ADR-0011](../../docs/adr/0011-fact-taxonomy-and-upcasting.md) |
| [RFC-0006](../../docs/rfc/0006-explainability-as-a-return-type.md) | `Explained[T]` — a calculation that cannot explain itself does not compile | [ADR-0012](../../docs/adr/0012-explained-return-type.md) |

Open questions each ADR deliberately leaves unresolved are listed in its Notes
section, with the milestone that must settle them. Where this document
describes layering, it still records *intent*, marked as such — the layering
itself has no ADR yet.

## Repository structure

| Directory | Role |
|-----------|------|
| `libs/` | Reusable libraries. One independent Go module per subdirectory. Empty until M2. |
| `apps/` | Deployable applications. Composition roots only. Empty until after M2. |
| `docs/` | Constitution, ADRs, RFCs. Authoritative record. |
| `deploy/` | Deployment topology. Empty until there is something to deploy. |
| `examples/` | Executable demonstrations of the public contract surface. Empty until M4. |
| `scripts/` | Enforcement mechanisms, invoked through `make`. |
| `.github/` | CI workflows. Empty until M3. |
| `.context/` | This directory — knowledge for agents. |

Every directory declares its contract in the front matter of its `README.md`:
`purpose`, `owner`, `allowed`, `forbidden`. `make contracts-check` enforces that
these exist, are complete, and agree with `CODEOWNERS`.

From **M2** those `allowed`/`forbidden` lists become the **source** of the
import-boundary linter configuration rather than a description of it. A README
that misdescribes its module's real dependencies will fail the build.

## Modules

Each `libs/*` is an independent Go module published under
`github.com/FabioCaffarello/fdos/libs/<name>` (ADR-0003, ADR-0004).

`go.work` is committed for local convenience. **CI builds with `GOWORK=off`**, so
inter-module coupling is always proven against published module versions rather
than local paths. A change that works locally and fails in CI is almost always
this.

Releases use Go's subdirectory-prefixed tags (`libs/<name>/vX.Y.Z`).

## Intended layering — hypothesis, not decision

To be confirmed, refined or replaced by the M1.5 RFCs:

| Layer | May depend on | Forbidden |
|-------|---------------|-----------|
| `domain` | nothing outside itself | I/O, concurrency, clocks, randomness, serialisation, binary floating point |
| `app` | domain | direct infrastructure imports |
| `adapters` | domain, app | — |

The intent is a **pure functional core with an imperative shell**: data in, facts
and decisions out.

Two consequences that are easy to get wrong:

- **Ports do not live in `domain`.** If port interfaces are declared in the pure
  domain they will want `context.Context` in their signatures, and the ban on
  `context` collapses immediately. Ports belong to `app`.
- **Serialisation is not a domain concern.** JSON struct tags on canonical models
  are provider leakage into the domain — precisely what Constitution §3 forbids.

## Determinism constraints (M2)

The domain layer is where Constitution §2, §3 and §10 stop being conventions.
Planned static analysis bans, inside domain packages:

| Banned | Why |
|--------|-----|
| `float64`, `float32` | Binary floating point destroys reproducibility and silently corrupts money. Decimals or integer minor units only. |
| `time.Now()` | Non-deterministic. Clocks are injected. |
| `math/rand` | Non-deterministic. Entropy is injected. |
| `os.Getenv` | Hidden input; makes a calculation unreproducible from the ledger alone. |
| goroutines, channels, `sync.Mutex` | A pure core has no shared mutable state. Their presence signals misplaced I/O. |
| `context.Context` | An I/O concern. Its presence means the code belongs in `app`. |
| `encoding/json` | Serialisation is an adapter concern. |

The highest-leverage item is the `float` ban: it turns "financial calculations
are correct and reproducible" from an aspiration into a build error.

## The enforcement ladder

Every principle is enforced at the highest feasible mechanism (ADR-0005): type
system > static analysis > CI > automated review > documentation > human
discipline.

Constitution §15 records where each of the fourteen principles currently sits. At
M1 most sit at rung 6 — the honest reading of a repository this young. When
proposing work, prefer whatever raises a principle's rung.

## Knowledge Graph

Not built, and not required initially (Constitution §12). Every canonical model,
identifier strategy and event schema must nonetheless *support* graph projection
— a real constraint on the M1.5 identifier RFC, not a future concern. The graph
is a projection over immutable facts, never a source of truth.
