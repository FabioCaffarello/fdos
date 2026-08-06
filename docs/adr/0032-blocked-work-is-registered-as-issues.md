---
id: ADR-0032
title: Blocked work is registered as issues, and the register file is a tombstone
status: Accepted
date: 2026-08-06
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0032 — Blocked work is registered as issues, and the register file is a tombstone

## Context

ADR-0020 created `docs/blocked.md`: the register of work FDOS decided to do and
could not finish, kept because an unrecorded block becomes an unexplained gap.
It worked — nine entries, several resolved with their history intact — and it
had two structural problems that grew with use:

- **It is the only governance artifact at pure rung 6.** Nothing verifies an
  entry's format, its uniqueness, or that a "what unblocks it" ever gets
  checked again. `adr-check`, `rfc-check` and `contracts-check` guard their
  registers; nothing guards this one.
- **A second register already exists and is the one the ecosystem uses.**
  Cross-repository coordination is issues by invariant (E3: versioned
  artifacts and GitHub), `docs/ecosystem/labels.md` defines `status/blocked`
  using — its own words — the same argument that produced `blocked.md`, and
  real blocks were already being mirrored as issues (`fdos#32`, `fdos#20`).
  Two registers for one concept is the drifted-copy problem, and the file was
  becoming the copy that drifts.

Meanwhile the file is cited from 52 places, including immutable ADRs and code
comments in `libs/*` that explain their own existence by `B-003` and `B-007`.
Those references cannot be repointed — the decision log is append-only — so
whatever replaces the file must keep every `B-NNN` citation resolvable.

The human decided the direction on 2026-08-06: issues become the live
register, with a tombstone. Constitution §14's ordering is why the decision
lands here before the file changes.

## Decision

Blocked work is registered as **GitHub issues carrying the `status/blocked`
label**, in the repository whose work is blocked, using the shared taxonomy in
`docs/ecosystem/labels.md`. An issue states the blocker, what was delivered
anyway, and what unblocks it — the same three questions the file asked. When
the blocker sits in the other repository, the existing `xrepo/*` mirror
convention applies unchanged.

`docs/blocked.md` becomes a **tombstone**: frozen at B-001 … B-009, each open
entry annotated with the issue that now tracks it, each resolved entry left as
the historical record it already is. The `B-NNN` identifiers remain permanent
and citable — every existing reference stays valid. New blocked work gets an
issue, never a new `B-NNN`.

Nothing else moves. A decision still goes in the decision log; an issue is the
register of what a decision could not reach, exactly as the file was.

## Consequences

### Positive

- One live register instead of two, and it is the one that can carry labels,
  milestones, cross-repository mirrors, assignees and closure — none of which
  a markdown file can verify either.
- A block on human action (`B-005`'s repository setting, `B-006`'s signing
  key) becomes an open item on a list the human actually triages, instead of
  a paragraph read mostly by agents.
- The 52 existing references keep resolving, including the ones in immutable
  ADRs and in code comments.

### Negative

- **Issues are mutable and live outside the repository.** A register entry
  can now be edited without a commit, and a GitHub outage takes the register
  with it. Accepted because the register was never the durable record — the
  decision log is — and the file's own header said so.
- **The rung does not climb.** Nothing verifies that blocked work has an
  issue, that `status/blocked` is used honestly, or that tombstone links
  resolve. This trades one rung-6 register for another with better
  affordances; it does not mechanize anything.
- The tombstone is permanent. `docs/blocked.md` can never be deleted, because
  the references into it cannot all be repointed.

### Enforcement

Human discipline (rung 6), same as the file it replaces. What would climb: a
check that every `status/blocked` issue names its blocker, and that every
tombstone annotation resolves to a real issue — both possible against the
GitHub API, both deliberately not built until the issue register has been
lived with for a milestone.

## Alternatives considered

**Keep the file as the register.** Lost because the second register already
existed and was winning: the ecosystem's coordination channel is issues by
invariant, and every cross-repository block was already being written twice.

**Delete the file and repoint references.** Lost on arithmetic: 52 references,
several inside immutable ADRs that cannot be edited and code comments whose
commits explain them. A tombstone preserves every citation for the cost of one
frozen file.

**Run both live (file and issues, mirrored).** The drifted-copy problem as a
policy. Lost for the reason the repository names that problem everywhere else.

## Notes

- Migrated in M9.5: B-002 → #53, B-004 → #54, B-005 → #55, B-006 → #56,
  B-007's open items → #57. Resolved entries (B-001, B-003, B-008, B-009's
  published half) stay in the tombstone as history.
- Issue templates encoding the label taxonomy are the following slice of the
  same milestone.
