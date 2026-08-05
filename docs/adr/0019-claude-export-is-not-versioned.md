---
id: ADR-0019
title: The Claude Code export is not versioned
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes:
  - ADR-0017
superseded_by: []
---

# ADR-0019 — The Claude Code export is not versioned

## Context

ADR-0016 gitignored `.claude/`. ADR-0017 reversed that a day later on a real
observation: Claude Code loads agents and skills from `.claude/` and nowhere
else, so an ignored export means a fresh clone has a complete `.context/` and
not one active agent — a first copy that never loads.

That observation is correct and this ADR does not dispute it. What it disputes
is that versioning the export solves it.

Measured on the tree ADR-0017 produced:

```
skills in .context/  (the reviewed roster)      7
skills in .claude/   (what actually loads)     17
```

The ten that load but were never reviewed: `api-design`, `bug-investigation`,
`refactoring`, and the exporter's seven `dotcontext-*` skills. Three of those
were removed from `.context/` **by name, with recorded reasons**, at M1 and
M2.5.

So versioning does not deliver "the roster that loads". It delivers a roster
that loads and contradicts the one that was reviewed, in which the majority of
entries were never reviewed at all. ADR-0017 named this cost and accepted it;
the number is what makes it unacceptable.

`.context/` being "authoritative" is a claim no mechanism enforces at load time.
Claude Code reads `.claude/`. In practice the pruned-out skills are the live
ones, and the authoritative designation is documentation.

## Decision

**`.claude/` is not versioned.** It returns to `.gitignore`, and to the
exclusion lists in `scripts/verify-directory-contracts.sh` and
`scripts/verify-doc-references.sh`.

`.context/` is the reviewed roster and the only one in the repository.
`AGENTS.md` remains the entry point; `CLAUDE.md` and `.cursorrules` remain
pointers at it.

`make doctor` reports whether the export exists and how to produce it, so the
absence ADR-0017 identified is at least visible rather than silent.

## Consequences

### Positive

- The repository contains one roster, and it is the one that was reviewed. No
  committed file contradicts an accepted decision (ADR-0015).
- A reviewer reading `.context/skills/` sees the whole of what FDOS decided to
  keep, with no second set to reconcile.
- No symlink portability cost, and no export diff noise that a reviewer learns
  to skip past — which was ADR-0017's own stated risk for how the contradiction
  would go unnoticed.

### Negative

- **The problem ADR-0017 identified is real and is not solved here.** A fresh
  clone loads no agents and no skills into Claude Code until someone runs the
  export. This ADR trades an invisible absence for a visible contradiction and
  judges the absence the lesser harm; it does not pretend the absence is gone.
- The mitigation is weak. `make doctor` reports the missing export, which is
  rung 5 at best: it tells a person something, and relies on them acting.
- **`make bootstrap` cannot fix this.** The export is an MCP call with no
  reachable CLI equivalent, so it cannot be automated from a Makefile target.
  If a CLI appears, `bootstrap` should run it and this consequence disappears.
- Agents reading through Claude Code lose the convenience of loaded playbooks
  and must read `.context/agents/*.md` as ordinary files. They are markdown, not
  executable, so nothing is unavailable — only less ergonomic.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| `.claude/` is not committed | 3 | `.gitignore`, and CI would fail `contracts-check` on an uncontracted directory |
| One reviewed roster | 3 | `make agent-contract-check` — rosters match `.context/` on disk |
| The export exists locally | 5 | `make doctor` reports it |

## Alternatives considered

**Keep ADR-0017 as it stands.** Rejected on the measurement: ten of seventeen
loaded skills were never reviewed, three of them removed by name with recorded
reasons. Committing that makes the repository state a decision and its
contradiction simultaneously.

**Version `.claude/agents/` only, ignore `.claude/skills/`.** ADR-0017
considered and rejected this; on the numbers it is the strongest surviving
option. The agents are symlinks and cannot drift, and the entire contradiction
lives in the skills. Rejected here for a different reason than ADR-0017 gave: a
half-versioned generated directory is harder to reason about than either
extreme, and `.gitignore` rules for "this generated subtree but not that one"
rot quietly. Worth revisiting if the loading problem becomes acute in practice
rather than in principle.

**Prune the exporter's built-ins after each sync and version the result.**
Rejected for ADR-0017's reason, which stands: a manual step repeated on every
export will be skipped, and a partially-pruned tree is worse than a fully
generated one.

**Ask the exporter not to add built-ins.** The `exportSkills` action takes
`includeBuiltIn`. If setting it false produces a tree matching `.context/`
exactly, the entire conflict dissolves and versioning becomes correct. This was
not verified before deciding, and it is the first thing to check before
revisiting.

## Notes

This is the second reversal on the same question in two days: ADR-0016 →
ADR-0017 → ADR-0019. The log shows the oscillation rather than hiding it, which
is what an append-only decision log is for.

The disagreement was never about the principle — both ADRs serve one source of
truth. It was about which arrangement produces one, and the measurement is what
settled it. That measurement was available before ADR-0017 and nobody took it.

Open:

- Whether `includeBuiltIn: false` yields an export matching `.context/` exactly.
  If it does, revisit this decision.
- Whether Claude Code can be pointed at `.context/` directly. ADR-0017 noted
  that no such setting exists; if one appears, the export becomes unnecessary
  and this decision becomes permanent by default.
