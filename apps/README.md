---
directory: apps
purpose: Deployable FDOS applications. Composition roots only — no business logic.
owner: "@FabioCaffarello"
allowed:
  - Executable Go modules with a main package, one per subdirectory
  - Dependency wiring and composition of libs/ modules
  - Configuration loading, flag parsing and process lifecycle
  - Transport bootstrapping (HTTP, gRPC, CLI, workers)
forbidden:
  - Business rules or financial calculations of any kind
  - Canonical model definitions
  - Logic that cannot be tested without starting a process
  - Imports between applications
---

# apps

Each subdirectory is a deployable application: an independent Go module with a
`main` package, published under `github.com/FabioCaffarello/fdos/apps/<name>`.

## Status: empty by design

`apps/` contains no applications at M0. Applications compose libraries, and
`libs/` is deliberately empty until M2 (see `libs/README.md`).

## Composition roots, not logic

An application in FDOS is a **composition root**. It reads configuration,
constructs adapters, wires them into application services, and starts a process.
That is the whole of its responsibility.

If a rule about money lives in `apps/`, it is unreachable by the determinism
analyser, unreachable by property-based tests, and unreproducible outside a
running process. Every financial rule belongs in a domain module where
Constitution §2 and §9 are mechanically enforced.

The practical test: an application should be almost entirely uninteresting to
read, and removing it should destroy no business knowledge.

## No inter-application imports

Applications never import each other. Shared behaviour belongs in `libs/`. Two
applications needing the same code is the signal that the code was placed in the
wrong directory.
