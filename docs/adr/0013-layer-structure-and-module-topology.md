---
id: ADR-0013
title: Modules follow bounded contexts; layers are packages within them
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0013 — Modules follow bounded contexts; layers are packages within them

## Context

ADR-0004 decided one Go module per `libs/*` but deliberately deferred *what* a
`libs/*` is, because the layer structure was an open M1.5 question. That question
is now answerable: RFC-0001 through RFC-0006 are accepted, and they constrain the
layering more tightly than the earlier hypothesis did.

Three constraints now come from accepted decisions rather than from preference:

- **Ports cannot live in the pure domain.** RFC-0003 forbids `context.Context`
  there, and any port interface doing I/O will want it. Ports belong to the
  application layer.
- **`Explained[T]` and the canonical primitives are cross-domain.** RFC-0002's
  `Money`, RFC-0001's `EntityID`, RFC-0004's envelope and RFC-0006's
  `Explained[T]` are shared vocabulary. They cannot live inside any one domain.
- **Each domain owns its ubiquitous language** (Constitution, Domain Vision).
  Ledger, Portfolio, Market Data and Credit Intelligence must not share domain
  types beyond the kernel.

The remaining question is whether modules are drawn per *layer* or per *bounded
context*.

## Decision

**Modules follow bounded contexts. Layers are packages inside them.**

```
libs/kernel/                 module — cross-domain canonical primitives
libs/<context>/              module — one bounded context
    domain/                  pure: no I/O, no clocks, no concurrency, no float
    app/                     use cases, ports, orchestration
    adapters/                pure adapters only (in-memory, codecs)
libs/<context>-<tech>/       module — infrastructure-heavy adapters
apps/<name>/                 module — composition root
```

### Layer dependency rule

| Layer | May import |
|-------|------------|
| `kernel` | nothing outside itself |
| `<context>/domain` | `kernel` |
| `<context>/app` | `kernel`, own `domain` |
| `<context>/adapters` | `kernel`, own `domain`, own `app` |
| `<context>-<tech>` | the context module, `kernel` |
| `apps/*` | anything in `libs/` |

No context module imports another context module's `domain` or `app`. Contexts
integrate through published contracts (Constitution §11), never by reaching into
each other's language.

### Infrastructure-heavy adapters are separate modules

This is the non-obvious half of the decision.

Go resolves dependencies per module, not per package. If `libs/ledger` contained
`adapters/postgres`, then every consumer importing `libs/ledger/domain` would
inherit the Postgres driver in its module graph and `go.sum` — even though it
imports none of it.

That would make Constitution §10 true at the package level and false at the
dependency level, which is the level that actually determines what a consumer is
coupled to. So a context module stays dependency-light, and anything requiring a
driver, broker, browser or SDK lives in `libs/<context>-<tech>`.

### The kernel is deliberately small

`libs/kernel` holds only what RFC-0001, RFC-0002, RFC-0004 and RFC-0006 defined
as universal: identity, money and quantity, the provenance envelope, temporal
coordinates, `Explained[T]`.

Anything a single context could own is not kernel. A shared kernel that grows
becomes a second place where domain language lives, and then every context is
coupled to every other through it.

### Tooling modules carry their own `cmd/`

`libs/README.md` forbade main packages outright, reserving them for `apps/`.
That rule is correct for deployable services and wrong for developer tooling.

Splitting a linter into `libs/analysis` plus `apps/fdoslint` puts a main package
in a *different module* from the library it wires. With nothing published yet,
the only way to build it is a local-path `replace` directive — reintroducing
precisely the by-path coupling ADR-0004 exists to prevent, and defeating
`GOWORK=off`.

So: **modules whose purpose is developer tooling may contain `cmd/<name>`.**
`apps/` remains reserved for deployable applications. The distinction is
deployment, not the presence of a `func main`.

### Which modules exist at M2

Only the toolchain:

```
libs/analysis/                 the determinism analysers
libs/analysis/cmd/fdoslint/    the runner that applies them
```

`libs/kernel` and the context modules arrive when there is something to put in
them — the kernel at M6 with the Ledger, per the roadmap. Creating them empty
now would reproduce the M0 scaffold problem: directories that survive no clone
and carry no meaning.

## Consequences

### Positive

- Module boundaries coincide with language boundaries. A context's ubiquitous
  language cannot leak, because leaking would require a module dependency that
  the layer rule forbids and `GOWORK=off` proves.
- Domain modules stay dependency-light. A consumer wanting canonical types does
  not inherit a database driver.
- Layer violations are cheap to enforce: they are package-path rules within a
  module, expressible in `depguard` and generated from directory contracts.

### Negative

- **`libs/` grows two module kinds** (`<context>` and `<context>-<tech>`), and
  the split is a judgement call at the margin. A pure in-memory adapter belongs
  inside; a driver-backed one does not; something using only `net/http` from the
  standard library is genuinely arguable.
- Splitting a heavy adapter out later is a breaking change for its importers, so
  the judgement is expensive to revise. The bias should be to split early.
- Per-context modules mean a change spanning kernel and two contexts is three
  coordinated releases (the ADR-0004 tax, now concrete).
- Layers-as-packages means the layer rule is enforced by a linter rather than by
  the compiler — rung 2, not rung 1. Per-layer modules would have made it rung 1
  via module resolution, and that is a real loss, accepted below.

### Enforcement

Rung 2 (static analysis). `depguard` rules generated from each module's
`README.md` contract, so the documentation *is* the configuration. Cross-context
imports and layer inversions fail the build.

Rung 2 also for domain purity, via the analysers in `libs/analysis`.

Rung 3 for the module graph itself: `GOWORK=off` in CI proves that a
context module does not depend on a `-tech` module (ADR-0004).

## Alternatives considered

**Modules per layer (`libs/domain`, `libs/app`, `libs/adapters`).** Rejected,
and it was the stronger option on one axis: it makes the layer rule a compiler
error rather than a lint rule — rung 1 instead of rung 2. It lost because it
forces every bounded context to share one `domain` module, which puts Ledger and
Market Data types in the same package namespace and guarantees exactly the
language leakage Constitution §3 and the Domain Vision forbid. Trading a rung
for a guaranteed modelling failure is a bad trade.

**One module for all of `libs/`.** Rejected by ADR-0004 already; repeated here
because layers-as-packages makes it superficially attractive again. It gives up
independent versioning of published contracts, which Constitution §11 requires.

**Adapters always inside the context module.** Simpler, one fewer module kind.
Rejected on the dependency-graph argument above: it is the difference between
Constitution §10 being true in the import graph and true in the build.

**Layer packages named `internal/domain` etc.** Would let the compiler enforce
that outsiders cannot import a context's internals. Attractive, and worth
revisiting per module. Rejected as a blanket rule because a context's `domain`
types are frequently part of its published contract, and `internal/` would make
publishing them impossible.

## Notes

Open, deliberately:

- The precise line between `<context>` and `<context>-<tech>` for adapters using
  only the standard library.
- Whether `kernel` should itself be split (identity, numerics, provenance) once
  its size is known. Splitting later is a breaking change; the bias is to keep it
  as one module until there is evidence.
- Whether context modules publish their `domain` types directly or only through
  the M4 contract surface. This interacts with `internal/` above and should be
  settled before the first context module is published.
