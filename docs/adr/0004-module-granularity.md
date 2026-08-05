---
id: ADR-0004
title: Each libs/* is an independent Go module, and CI builds with GOWORK=off
status: Accepted
date: 2026-08-04
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0004 — Each `libs/*` is an independent Go module, and CI builds with `GOWORK=off`

## Context

FDOS must eventually publish contract modules consumed by private connector
repositories (Constitution §13). Two module layouts were available: a single
root module with boundaries enforced by `internal/` and import linting, or one
module per `libs/*` with boundaries enforced by module resolution itself.

The single-module layout is materially simpler and defers cost. The multi-module
layout charges a continuous maintenance tax — cross-module version bumps,
subdirectory-prefixed tags, release coordination — in exchange for boundaries
that are real rather than conventional.

There is a decisive asymmetry. With a single module, the open-core extraction is
a future event: a large, one-time, high-risk change whose failure modes are
discovered on the day it happens. With multiple modules, the extraction is not
an event at all — the boundary is exercised by every build from the first
commit.

## Decision

Each `libs/*` is an independent Go module with its own `go.mod`, published under
the path defined in ADR-0003.

A `go.work` file is committed at the repository root as a developer-experience
convenience.

**CI builds with `GOWORK=off`.** This is the load-bearing half of the decision.

Releases use Go's subdirectory-prefixed tag convention (`libs/<name>/vX.Y.Z`),
driven by automation from M3. Manual tagging is not an acceptable fallback.

Modules are not created in M0. The layer structure — `domain`, `app`, `adapters`
— is an output of the M1.5 canonical-architecture RFCs, and creating modules
before that RFC lands would pre-judge it.

## Consequences

### Positive

- Module boundaries are enforced by the toolchain, not by convention or lint
  configuration that can be suppressed.
- The open-core extraction is continuously verified rather than deferred. Every
  pull request proves the modules compile against each other by published
  version.
- Public contract modules can be versioned and released independently of the
  applications that consume them, which is what Constitution §11 requires.

### Negative

- Continuous maintenance tax: cross-module version bumps propagate in chains,
  and a change spanning three modules requires three coordinated releases.
- Subdirectory-prefixed tagging is unfamiliar and easy to get wrong by hand.
  Until the release automation of M3 exists, this is a real hazard.
- `go.work` actively masks version incompatibilities during local development.
  A developer can work for days against a local path and discover the breakage
  only in CI. This is the direct cost of the convenience and is accepted
  knowingly.
- The tax is paid from day one, while the benefit — open-core extraction —
  arrives at M5. If FDOS never extracts, this decision was wrong.

### Enforcement

Rung 2 (static analysis) for the boundary itself: with `GOWORK=off`, an
undeclared cross-module dependency does not compile.

Rung 3 (CI) for the discipline: CI must set `GOWORK=off` explicitly. Without it
the entire decision silently degrades into a single module wearing several
`go.mod` files, and nothing would report the degradation. The CI workflow that
sets it lands in M3; until then this sits at rung 6 and is the most fragile
commitment in this ADR.

## Alternatives considered

**Single root module.** Rejected, and it was the recommended option. It is
simpler, has no version-bump tax, and the later split is mechanical. It lost on
the asymmetry above: a mechanical split is still a one-time event whose failure
modes surface all at once, whereas continuous verification surfaces them one at
a time. If open-core extraction were speculative rather than planned, this would
be the right answer.

**Multi-module without committing `go.work`.** Rejected: it would remove the
masking hazard, but every developer would maintain a local `go.work` by hand,
producing inconsistent local environments — a worse and less visible problem
than the one it solves.

**Single module now, split at M5.** Rejected: it concentrates all boundary risk
at exactly the milestone that can least absorb it, since M5 is also when the
private repositories first depend on those boundaries.
