---
directory: .claude
purpose: Generated export of .context/ for Claude Code, which loads agents and skills from here and nowhere else.
owner: "@FabioCaffarello"
allowed:
  - Symlinks into .context/agents/
  - Skills exported from .context/skills/
  - Skills the exporter adds on its own
forbidden:
  - Hand-edited agents or skills — edit .context/ and re-export
  - A copy of the operating rules (CLAUDE.md points at AGENTS.md)
  - Per-user state (settings.local.json is gitignored)
  - Anything with no counterpart in .context/ that a human wrote here
---

# .claude

Generated, versioned, and **not** authoritative. [`.context/`](../.context/) is
the source; this directory is what Claude Code can actually read.

That split is the whole reason this tree is committed. `CLAUDE.md` and
`.cursorrules` get to be pointers at [`AGENTS.md`](../AGENTS.md) because those
tools read those filenames. Agents and skills have no such indirection — there
is no setting that aims Claude Code at `.context/` — so a gitignored `.claude/`
means a fresh clone loads no agents and no skills at all. ADR-0017 records that
trade, including what it costs.

## How it is produced

Via the `dotcontext` MCP server, against this repository:

| What | Action | Mode |
|------|--------|------|
| `agents/` | `sync exportAgents` | `symlink` |
| `skills/` | `sync exportSkills` | copy (the only mode) |

`exportContext` runs both, and skips `CLAUDE.md` and the docs index unless
forced. Leave it skipping: `CLAUDE.md` is written by hand and points at
`AGENTS.md`.

**`exportSkills` has no `dryRun`.** It writes on the first call, unlike every
other action on that tool.

## agents/ — symlinks, so they cannot drift

Each entry links to `../../.context/agents/<name>.md`. One file, one roster.
Export with `mode: "symlink"`; `markdown` makes copies and reintroduces exactly
the drift this arrangement rules out.

`.context/agents/README.md` is not exported — it has no frontmatter and Claude
Code would reject it as an agent definition.

## skills/ — copies, and where the known contradiction lives

The exporter normalises skill frontmatter on write, so these cannot be
symlinked. `.context/skills/` wins every disagreement: a skill here that differs
is a stale copy, never a decision.

The export also re-adds its own built-in skills, including `api-design`,
`refactoring` and `bug-investigation` — roles `.context/` deliberately pruned
for having no subject in this repository, plus the `dotcontext-*` set that
drives PREVC. They are committed with the rest. ADR-0017 records this as an
accepted cost rather than a solved problem.

## Changing an agent or a skill

Edit `.context/`, re-export, commit both. An edit made here is overwritten by
the next sync — silently, since nothing checks for it.
