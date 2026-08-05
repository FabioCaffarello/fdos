---
id: ADR-0003
title: Go modules are published under github.com/FabioCaffarello/fdos
status: Accepted
date: 2026-08-04
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0003 — Go modules are published under `github.com/FabioCaffarello/fdos`

## Context

Go module paths are load-bearing identifiers. They appear in every import
statement, in every downstream repository, and in the module proxy's immutable
record. Changing one later is not a rename: it is a new module, and every
consumer must migrate deliberately.

FDOS will publish contract modules consumed by private connector repositories
(Constitution §13), which makes the path a cross-repository interface, not an
internal detail.

## Decision

FDOS Go modules are published under the `github.com/FabioCaffarello/fdos`
prefix.

Module paths follow the repository layout:

```
github.com/FabioCaffarello/fdos/libs/<name>
github.com/FabioCaffarello/fdos/apps/<name>
```

Note that the module prefix is `fdos`, while the repository is named
`financial-data-operating-system`. This is deliberate: import statements are
read far more often than repository URLs, and `fdos` is the project's actual
name in every other context.

## Consequences

### Positive

- No additional infrastructure. The path resolves directly through the standard
  module proxy with no hosting, DNS or availability dependency.
- Provenance is self-evident: the import path names the repository that produced
  the code, which matters for a project asking institutions to trust it.

### Negative

- The path is bound to GitHub and to a personal account namespace. Migrating to
  an organisation account, or off GitHub entirely, requires a module path change
  and a coordinated migration across every consumer, including private
  repositories.
- If FDOS ever adopts an organisational identity, the personal namespace will
  read as a historical accident.
- Reaching v2 requires the `/v2` suffix in the path, per Go's semantic import
  versioning. This is a general Go constraint, but it compounds with any future
  path migration.

### Enforcement

Rung 2 (static analysis) from M2, once modules exist: the module path is
declared in each `go.mod` and any inconsistency is a compile error. A check that
every module path matches its directory location belongs in `make verify` at M2.

## Alternatives considered

**A vanity import path (for example `fdos.dev/libs/...`).** Rejected for now,
and this is the alternative with the strongest case. It would decouple the
import path from the hosting provider, making a future migration invisible to
consumers — precisely the risk listed above. It was rejected because it requires
owning a domain and serving `go-import` meta tags, and an outage or lapsed
registration of that endpoint breaks builds for every consumer. That is a new
availability dependency introduced before the project has a single user. This
decision should be revisited before the first external consumer exists;
afterwards the migration cost rises sharply.

**`github.com/FabioCaffarello/financial-data-operating-system/...`.** Rejected:
accurate but unwieldy in every import statement in the codebase.
