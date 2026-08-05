---
directory: libs
purpose: Reusable FDOS libraries. Each subdirectory is an independent, publishable Go module.
owner: "@FabioCaffarello"
allowed:
  - Independent Go modules, one per subdirectory (ADR-0004)
  - Canonical financial models and domain logic
  - Application and orchestration layers
  - Infrastructure adapters implementing domain-owned ports
  - Contract packages published for consumption by private repositories
forbidden:
  - Entry points for deployable services (those belong in apps/)
  - Provider-specific or institution-specific concepts in domain packages
  - Dependencies from a lower layer onto a higher one
  - Imports that cross between bounded contexts
  - Modules without a README declaring their own contract
---

# libs

Every subdirectory of `libs/` is an **independent Go module** with its own
`go.mod`, published under `github.com/FabioCaffarello/fdos/libs/<name>`
(ADR-0003, ADR-0004).

## Current modules

| Module | Kind | Purpose |
|--------|------|---------|
| [`analysis/`](analysis/README.md) | tooling | The static analysers that enforce the rules below |

`libs/kernel` and the bounded-context modules arrive when there is something to
put in them — the kernel at **M6** with the Ledger. Creating them empty now
would reproduce the M0 scaffold problem: directories that survive no clone and
carry no meaning.

## Topology (ADR-0013)

Modules follow **bounded contexts**. Layers are packages inside them.

```
libs/kernel/                 cross-domain canonical primitives
libs/<context>/              one bounded context
    domain/                  pure: no I/O, clocks, concurrency, or float
    app/                     use cases, ports, orchestration
    adapters/                pure adapters only
libs/<context>-<tech>/       infrastructure-heavy adapters
```

| Layer | May import | Enforced by |
|-------|------------|-------------|
| `kernel` | nothing first-party | `layering` |
| `<context>/domain` | `kernel` | `layering`, `impurity`, `nofloat`, `nondet` |
| `<context>/app` | `kernel`, own `domain` | `layering` |
| `<context>/adapters` | `kernel`, own `domain`, own `app` | `layering` |

No context reaches into another context's `domain` or `app`. Contexts integrate
through published contracts (Constitution §11), never through shared types.

**Infrastructure-heavy adapters are separate modules.** Go resolves dependencies
per module: an adapter with a database driver inside `libs/ledger` would put that
driver in the graph of everyone importing `libs/ledger/domain`. That would make
Constitution §10 true at the package level and false at the level that decides
what a consumer is actually coupled to.

## Enforcement

The domain layer is where Constitution §2, §3 and §10 stop being conventions.
`make analyze` fails the build on:

| Forbidden in `domain` | Rule |
|-----------------------|------|
| `float32`, `float64` | `nofloat` — floating-point addition is not associative, so fold order changes the total |
| `time.Now`, `math/rand`, `os.Getenv` | `nondet` — hidden inputs cannot be reproduced from the ledger |
| range over a map (except key collection) | `nondet` — Go randomises iteration order |
| goroutines, channels, `select`, `sync` | `impurity` — a pure core has no shared mutable state |
| `context.Context` | `impurity` — an I/O concern; ports belong in `app` |
| `json`/`db` struct tags | `impurity` — the wire format belongs to adapters |
| layer inversion, cross-context imports | `layering` |

Each module also carries its own `README.md` contract, exactly as this file
does. These are binding: a module whose real dependencies contradict its
declared contract is a defect, caught in review today and by generated
configuration once more than one context module exists.

## Cross-module development

`go.work` is committed for local convenience. CI builds with `GOWORK=off`, so
inter-module coupling is always proven against published module versions rather
than local paths (ADR-0004). A change that works locally and fails in CI is
almost always this.
