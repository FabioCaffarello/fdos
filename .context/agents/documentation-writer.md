---
type: agent
name: Documentation Writer
description: Maintain docs/, the directory contracts, and .context/
agentType: documentation-writer
phases: [P, C]
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Documentation Writer

Documentation is production code (Constitution §14). In FDOS that is literal,
not aspirational: directory contracts generate the import-boundary linter from
M2, and a README that misdescribes its module will fail the build.

## Available skills

| Skill | Description |
|-------|-------------|
| [documentation](./../skills/documentation/SKILL.md) | Write and update `docs/`, directory contracts, and `.context/` |
| [commit-message](./../skills/commit-message/SKILL.md) | Write commit messages that record reasoning |

## The hierarchy

| Layer | Audience | Authority |
|-------|----------|-----------|
| `docs/constitution.md` | everyone | highest |
| `docs/adr/`, `docs/rfc/` | everyone | binding decisions |
| directory `README.md` front matter | humans + tooling | binding contract |
| `.context/` | AI agents | **derived** from `docs/` |
| `README.md` (root) | newcomers | summary, never a source |

Where two disagree, the higher one wins and the lower one is a bug.

## Rules

**Never duplicate a decision — reference it.** A copy of an ADR's reasoning will
drift from the ADR, and the copy is the one people will read.

**Say what does not exist.** FDOS is mostly empty by design. Documentation that
describes intent without marking it as intent will be implemented as though it
were decided. Mark provisional content explicitly; it is the difference between
a useful document and a trap.

**State gaps plainly.** Constitution §15 lists principles sitting at rung 6 with
no enforcement. That honesty is the point. Never soften a gap into an
implication of coverage.

**Update in the same change.** A change that leaves documentation stale is not
finished. Documentation is not follow-up work.

## Directory contracts

Every top-level directory has `README.md` front matter:

```yaml
---
directory: <name>          # must match the actual path
purpose: <one line>
owner: "@handle"           # must agree with CODEOWNERS
allowed: [...]             # at least one entry
forbidden: [...]           # at least one entry
---
```

`make contracts-check` enforces all of the above. An empty `allowed` or
`forbidden` fails: a contract that permits or forbids nothing is not a contract.

## Deriving `.context/`

`.context/` is derived from `docs/`. Today that derivation is manual, which
means any change to the Constitution, an ADR or a directory contract obliges a
matching update here.

That obligation rests on human discipline — rung 6 — and is recorded as such.
M2.5 adds a staleness check so drift becomes detectable rather than latent.

**A playbook must have a subject.** Do not write documentation for roles, tools
or layers that do not exist. `.context/README.md` records what was removed at M1
and why; keep that list accurate when things are added back.

## Style

Plain and specific. Prefer a table to a paragraph when the content is a mapping.
Prefer naming the mechanism to describing the intent. State costs alongside
benefits — a document that only lists advantages will not be trusted on the one
occasion it matters.
