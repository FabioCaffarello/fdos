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
  - Executable entry points (main packages belong in apps/)
  - Provider-specific or institution-specific concepts in domain packages
  - Dependencies from a lower layer onto a higher one
  - Modules without a README declaring their own contract
---

# libs

Every subdirectory of `libs/` is an **independent Go module** with its own
`go.mod`, published under `github.com/FabioCaffarello/fdos/libs/<name>`
(ADR-0003, ADR-0004).

## Status: empty by design

`libs/` contains no modules at M0. The layer structure — the boundaries between
pure domain, application orchestration and infrastructure adapters — is an
output of the **M1.5 canonical-architecture RFCs**. Creating modules before that
RFC lands would pre-judge its outcome, which the review process exists to
prevent.

Modules arrive in **M2**, together with the static analysis that enforces the
boundaries between them.

## Module contracts are executable

Each module will carry its own `README.md` declaring its `allowed` and
`forbidden` dependencies in front matter, exactly as this file does.

From M2 these declarations are the **source** of the import-boundary
configuration, not a description of it. The linter is generated from the
READMEs. A README that misdescribes its module's real dependencies will fail the
build.

This is the concrete meaning of "documentation is production code"
(Constitution §14): the document is not a record of the rule, it *is* the rule.

## Layering

The layer names below are the working hypothesis to be confirmed, refined or
replaced by the M1.5 RFCs. They are recorded here as intent, not as decided
architecture.

| Layer | May depend on | Notably forbidden |
|-------|---------------|-------------------|
| domain | nothing outside itself | I/O, concurrency, clocks, randomness, serialisation, binary floating point |
| app | domain | direct infrastructure imports |
| adapters | domain, app | — |

The domain layer is intended to be a pure functional core: data in, facts and
decisions out. It is the layer where Constitution §2 (Deterministic
Engineering), §3 (Canonical Model First) and §10 (Domain Before Infrastructure)
are enforced mechanically rather than observed by convention.

## Cross-module development

`go.work` is committed for local convenience. CI builds with `GOWORK=off`, so
inter-module coupling is always proven against published module versions rather
than local paths (ADR-0004). A change that works locally and fails in CI is
almost always this.
