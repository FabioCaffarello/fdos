---
id: ADR-0000
title: FDOS records architecture decisions as an append-only log
status: Accepted
date: 2026-08-04
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0000 — FDOS records architecture decisions as an append-only log

## Context

FDOS is intended to be maintained and evolved over a decade. Over that horizon
the most expensive lost asset is not code — code is readable — but *reasoning*.
Why bitemporality was scoped the way it was, why a provider abstraction has an
awkward seam, why an obvious simplification was rejected: this reasoning decays
first and is unrecoverable once gone.

The Constitution requires that every financial fact be reconstructible years
later from an append-only log with no rewriting of history. The same argument
applies, with the same force, to the decisions that shaped the system.

## Decision

FDOS records architecture decisions as ADRs in `docs/adr/`, in MADR-derived
format, using the template in `docs/adr/template.md`.

The decision log obeys the ledger's law:

- **Append-only.** An accepted ADR is never edited to change its meaning.
- **Immutable.** A decision that proves wrong is not rewritten. A new ADR is
  written that supersedes it, and the original's status changes to `Superseded`
  with `superseded_by` naming the successor.
- **Corrections are new facts.** The chain from original to successor must remain
  traversable in both directions.

Typographical fixes that do not alter meaning are permitted. Anything that
changes what the decision *says* requires a superseding ADR.

An ADR is required for any change to: repository structure, module boundaries,
the public contract surface, the toolchain, enforcement mechanisms, or the
Constitution itself.

## Consequences

### Positive

- Reasoning survives the people who held it.
- Review has a stable object: proposals are argued against recorded decisions
  rather than against recollection.
- The supersession chain makes architectural drift visible instead of silent.

### Negative

- Ceremony cost on every structural change, paid up front, benefiting a future
  reader who may never exist.
- ADR processes decay into rubber-stamping unless enforced. Mitigated by
  `make adr-check` today, and from M3 by a CI rule that fails a pull request
  changing a public `libs/*` API without a linked ADR.
- Immutability means early ADRs will look naive. This is correct and must not be
  tidied away.

### Enforcement

Rung 3 (CI). `scripts/verify-adr.sh` validates front matter, id/filename
agreement, uniqueness, status vocabulary, ISO-8601 dates, and that every
superseded ADR names its successor. Immutability itself is enforced by review,
not by tooling — a git-history check for edits to accepted ADRs is a candidate
for M3.

## Alternatives considered

**No formal decision log; rely on pull request descriptions.** Rejected: PR
descriptions are indexed by change, not by decision, and are effectively
unsearchable at a decade's distance. They also disappear if the forge changes.

**A single evolving `ARCHITECTURE.md`.** Rejected: an evolving document records
only the current position. It systematically destroys the record of what was
rejected and why, which is the more valuable half.

**Full RFC process for every decision.** Rejected as the default: RFCs are
appropriate for decisions requiring design exploration, and FDOS keeps them in
`docs/rfc/` for exactly that. Most decisions do not need one, and requiring it
would make the process expensive enough to be routed around.
