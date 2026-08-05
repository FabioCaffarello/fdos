---
id: ADR-0017
title: The Claude Code export is versioned, and .context/ stays authoritative
status: Superseded
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by:
  - ADR-0019
---

# ADR-0017 — The Claude Code export is versioned, and `.context/` stays authoritative

> **Superseded by [ADR-0019](0019-claude-export-is-not-versioned.md).** The
> observation below is correct — Claude Code loads agents and skills from
> `.claude/` and nowhere else. The conclusion did not survive measurement: ten
> of the seventeen skills that load were never in the reviewed roster. The text
> is preserved unaltered, as ADR-0000 requires.

## Context

ADR-0016 established that the repository states nothing twice, and that
`CLAUDE.md` and `.cursorrules` are pointers at `AGENTS.md` rather than copies of
it. Applying that rule, its Notes resolved a conflict it had surfaced —
`.claude/` is a generated export of `.context/`, and the exporter re-adds
built-in skills that `.context/` deliberately pruned — by gitignoring `.claude/`
entirely.

That resolution held for one day. Syncing the export for the first time showed
what it costs.

The pointer discipline works for `CLAUDE.md` and `.cursorrules` because Claude
Code and Cursor read those filenames. **Agents and skills have no equivalent
indirection.** Claude Code loads them from `.claude/agents/` and
`.claude/skills/` and nowhere else; there is no setting that points either at
`.context/`. The export itself is a manual MCP call, in no `make` target and
named in no onboarding path.

Gitignoring `.claude/` therefore means a fresh clone has a complete `.context/`
and not one active agent or skill, until someone takes a step nothing tells them
to take. The roster is not merely a second copy in that arrangement — it is a
first copy that never loads.

This serves the same Constitution principle ADR-0016 does: one source of truth.
It disagrees only about which arrangement actually produces one.

## Decision

**FDOS versions `.claude/`.** `.context/` remains authoritative; `.claude/` is
its export, and the two are kept in the relationship below.

- **`.claude/agents/` are symlinks** into `.context/agents/`. There is one file
  per agent, so the agent roster is physically incapable of drifting. Export
  with `mode: "symlink"`, never `markdown`.
- **`.claude/skills/` are copies**, because the exporter normalises skill
  frontmatter on write and cannot symlink them. `.context/skills/` is
  authoritative: a skill that disagrees with it is a defect in the copy, not a
  competing decision.
- **`.claude/settings.local.json` stays gitignored** — per-user state, the same
  rule `.vscode/` follows.
- **`.claude/` declares a directory contract** like every other versioned
  top-level directory. The exclusion in
  `scripts/verify-directory-contracts.sh` is removed; it existed only because
  the tree was generated and ignored.

`AGENTS.md` remains the entry point, and `CLAUDE.md` remains a pointer at it.
This decision changes what is committed, not where the rules live.

## Consequences

### Positive

- A clone has a working agent and skill roster with no manual step.
- The agent roster cannot drift, because symlinks make it one file rather than
  two.
- `.claude/` becomes reviewable. A gitignored tree changes without appearing in
  any diff, which is the worse failure mode for something that steers agents.

### Negative

- **The repository now contains skills `.context/` pruned for having no
  subject** — `api-design`, `refactoring`, `bug-investigation`, and the
  exporter's `dotcontext-*` set. This is a second roster, and ADR-0016 was right
  that it contradicts ADR-0015. It is accepted because the alternative was a
  roster that never loads, not because the objection was answered.
- Regenerating the export produces diff noise, and a reviewer who sees only
  churn in `.claude/skills/` will stop reading it — which is how the
  contradicting roster would go unnoticed.
- Symlinks are a portability cost. They survive git, but a Windows checkout
  without `core.symlinks` gets files containing paths, and the agents silently
  fail to load.
- **This decision is wrong the moment Claude Code learns to read agents and
  skills from a configurable path.** At that point the export becomes
  unnecessary and `.claude/` should go back to being ignored.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| `.claude/` declares its contract | 3 | `make contracts-check` |
| `.claude/agents/` symlinks resolve | 3 | `make context-check` — links must resolve |
| `.claude/skills/` does not contradict `.context/skills/` | 6 | review |
| Export uses `mode: "symlink"` for agents | 6 | review |

The last two are rung 6. The skills row is the weaker of them, because the
exporter writes those files and no human edits them, so review is inspecting
machine output for a disagreement it has no reason to expect.

Climbing it is cheap and was deliberately not done yet: the export normalises
frontmatter, so a check comparing body text between `.claude/skills/*/SKILL.md`
and `.context/skills/*/SKILL.md` is a short script. It is worth writing the
first time a copy is found to have drifted, and not before — a check with no
recorded failure is a check nobody trusts.

## Alternatives considered

**Keep `.claude/` gitignored (ADR-0016's resolution).** Rejected on the
mechanism: there is no path that loads agents or skills from `.context/`, so
this trades a visible contradiction for an invisible absence. The contradiction
is at least in a diff.

**Version `.claude/agents/` only, and ignore the skills.** Tempting, and it
removes the entire negative consequence above — the symlinked agents cannot
drift, and the pruned-roster problem lives wholly in the skills. Rejected
because it leaves skills in exactly the state that motivated this ADR: present
in `.context/`, never loaded, with nothing indicating why.

**Prune the exporter's built-in skills after each sync.** Rejected: it is a
manual step that must be repeated on every export, so it will be skipped, and a
partially-pruned tree is harder to reason about than a fully generated one.
Worth revisiting if the exporter grows an `includeBuiltIn: false` that holds.

**Supersede ADR-0016.** Rejected. Its `## Decision` section does not state that
`.claude/` is ignored — that resolution appears in its `## Notes`, as part of an
incident report. ADR-0000 §Decision requires supersession for a change to what a
decision *says*, and superseding ADR-0016 wholesale would retire five decisions
that remain in force to reverse one paragraph that was never in Decision.
ADR-0016 carries an added pointer at this ADR instead, which its immutability
check permits.

## Notes

The exporter's `exportSkills` action has no `dryRun` parameter, unlike every
other action on that tool. It writes on the first call. This is worth knowing
before running it against a tree whose state matters.

`.cursorrules` is unaffected and remains a pointer at `AGENTS.md`. Cursor reads
rules from a root file, so the problem this ADR solves does not arise there.
