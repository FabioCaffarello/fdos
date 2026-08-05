---
id: ADR-0006
title: FDOS uses .context as the canonical AI knowledge directory
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes:
  - ADR-0001
superseded_by: []
---

# ADR-0006 — FDOS uses `.context` as the canonical AI knowledge directory

## Context

ADR-0001 adopted `.dotcontext/` one day ago, on the argument that the name
should distinguish structured engineering knowledge from generic application
configuration.

The argument was reconsidered before any content existed in the directory, which
is the cheapest possible moment to reverse it.

Two things decided the reversal.

**The name was carrying semantics that belong to content.** A directory does not
become configuration because of what it is called. `.context/` holds the
Constitution, agent playbooks and skills; its README and the Constitution state
what it is far more durably than a prefix can. If the directory ever accretes
generic configuration, that is a review failure, and renaming it would not have
prevented it.

**The divergence was not free.** `.context` is the tooling default and the
convention already established across seven sibling repositories in this
workspace. Every scaffolding call would have needed `outputDir` passed
explicitly, forever, with a stray `.context/` appearing the first time anyone
forgot. ADR-0001 acknowledged this and deferred a guard to M1 — which is to say
it planned to build a mechanism whose only purpose was to defend a naming
preference.

## Decision

FDOS uses `.context/` as the canonical AI knowledge directory.

No `outputDir` override is required: the tooling default and the repository
convention are the same value, so there is nothing to remember and nothing to
enforce.

Tracking follows the tool's own classification, unchanged from ADR-0001:

| Path | Classification | Tracked |
|------|----------------|---------|
| `.context/docs/**` | versioned | yes |
| `.context/agents/**` | versioned | yes |
| `.context/skills/**` | versioned | yes |
| `.context/config/**` | versioned | yes |
| `.context/plans/**` | local | no |
| `.context/cache/**` | runtime | no |
| `.context/runtime/**` | runtime | no |

## Consequences

### Positive

- Zero divergence from tooling defaults, from sibling repositories, and from any
  future generic integration. Nothing must be configured, documented or
  remembered.
- The M1 guard that ADR-0001 required in order to defend its own naming choice
  is no longer needed. A mechanism that exists only to protect a preference is
  pure cost, and it is now deleted rather than built.
- Contributors arriving from any other repository in this workspace find what
  they expect.

### Negative

- The semantic distinction ADR-0001 wanted to make is real, and the name no
  longer makes it. It now rests entirely on `docs/README.md` and on review — a
  rung 5 mechanism where ADR-0001 wanted a rung that names cannot actually
  provide anyway.
- `.context/` will attract generic configuration over a decade. This is a
  genuine risk and the honest mitigation is a directory contract in
  `.context/README.md` once the directory is populated in M1, enforced by the
  same `contracts-check` as every other directory.
- Reversing a decision one day after accepting it is a small signal that
  ADR-0001 was decided on preference rather than on cost. Recorded here rather
  than smoothed over.

### Enforcement

Rung 3 (CI). `make contracts-check` excludes `.context/` from directory-contract
enforcement today; in M1, when the directory is populated, it gains its own
contract and stops being an exception.

No guard against a stray directory is needed in either direction: with the
convention and the tooling default aligned, there is no second directory for the
tooling to create.

## Alternatives considered

**Keep `.dotcontext/` (ADR-0001).** Rejected on the reasoning above: it bought a
distinction that content already makes, and charged a permanent `outputDir`
discipline plus a guard mechanism to defend it.

**Edit ADR-0001 in place rather than superseding it.** Rejected. ADR-0000 makes
the decision log append-only precisely so that reversals stay visible. Editing
would have produced a document that read as though `.context` had been the
choice all along, destroying the record of a trade-off that a future reader may
need to re-evaluate. The log obeys the same law as the ledger: corrections are
new facts.

## Notes

This is the first supersession in the FDOS decision log, one commit into the
repository's life. That is the process working, not failing — the cost of
reversing a decision is one small file, and it is paid at the moment the
reversal is cheapest.
