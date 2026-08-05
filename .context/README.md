---
directory: .context
purpose: Structured engineering knowledge for AI agents, derived from docs/.
owner: "@FabioCaffarello"
allowed:
  - Documentation written for AI agents (.context/docs/)
  - Agent playbooks describing roles that have a subject in this repository (.context/agents/)
  - Skills describing procedures that are actually performed here (.context/skills/)
  - Harness configuration (.context/config/)
forbidden:
  - Application or deployment configuration
  - Any claim that contradicts docs/ — docs/ is authoritative
  - Playbooks or skills describing code, layers or tooling that does not exist yet
  - Runtime state, caches or plans under version control
---

# .context

Structured engineering knowledge for AI agents (ADR-0006).

`docs/` is the authoritative record, written for humans. This directory is
**derived from it**, never the reverse. Where the two disagree, `docs/` wins and
the disagreement is a bug in the derivation.

## Contents

| Path | Contents | Tracked |
|------|----------|---------|
| `docs/` | Repository knowledge for agents | yes |
| `agents/` | Role playbooks | yes |
| `skills/` | Procedure definitions | yes |
| `config/` | Harness policy and sensor catalog | yes |
| `plans/` | Local working plans | no |
| `cache/`, `runtime/` | Generated state | no |

## A playbook must have a subject

The scaffold generates a full roster by default. FDOS keeps only what describes
something that exists in this repository today.

This is not tidiness. A playbook describing structure that does not exist is
worse than no playbook: an agent acts on it confidently, and nothing reports
that it was wrong.

### Returned at M2.5, because they acquired a subject

| Restored | What gave it a subject |
|----------|------------------------|
| `test-writer`, `test-generation` | M2 produced Go code and a specific testing discipline — fixtures that prove specificity, not just sensitivity |
| `devops-specialist` | M3 produced workflows, hooks and a supply chain |
| `security-audit` | M3 produced a real security posture, and an equally real gap list |

### Still absent, with the reason

| Absent | Reason | Returns |
|--------|--------|---------|
| `database-specialist` | The domain must not depend on a database (Constitution §10). A first-class database role invites exactly the coupling that principle forbids | never |
| `frontend-specialist`, `mobile-specialist` | No user interface, no mobile surface | if one exists |
| `backend-specialist`, `feature-developer` | No services and no features; the only Go is the analyser toolchain | M6 |
| `bug-fixer`, `bug-investigation` | Almost no code to have bugs in | M6 |
| `refactoring-specialist`, `refactoring` | Too little code to refactor | M6 |
| `performance-optimizer` | No performance work, and no benchmarks to reason from | M6 |
| `api-design` | API design is generated from contracts. A skill now would pre-judge the proto → buf → OpenAPI chain | M4 |

## Derivation, and how it is checked

These files are derived from `docs/` by hand. Since M2.5 the derivation is
**checked**, not assumed:

| Check | Asserts |
|-------|---------|
| `make context-check` | Every link, `make` target, script path and ADR/RFC identifier named here exists |
| `make agent-contract-check` | Every playbook declares `must_read`, `must_not` and `evidence`; every `must_read` path exists; the rosters match disk in both directions; nothing is left `unfilled` |

What that does **not** cover: whether a paragraph is merely wrong, and whether
an agent actually reads its `must_read` or honours its `must_not`. Those stay at
rung 6, recorded as such in ADR-0015. A green check is not evidence of agent
compliance.

### `.claude/` is an export, not a source

Exporting this directory for Claude Code produces `.claude/`, and the exporter
re-adds its built-in skills — including `api-design`, `refactoring` and
`bug-investigation`, which are absent here on purpose.

`.claude/` is **versioned** (ADR-0017) but **not authoritative**. If it
disagrees with this directory, this directory wins.

It is versioned despite carrying that contradiction because Claude Code loads
agents and skills from `.claude/` and nowhere else: ignoring it would leave a
fresh clone with a complete `.context/` and not one active agent. The agents are
symlinks into this directory and cannot drift; the skills are copies, and a
skill here that disagrees with `.context/skills/` is a defect in the copy.

Generating `.context/` from `docs/` outright was considered and rejected for now
— the two have genuinely different audiences, and a generator would either
produce unusable prose or push so much annotation into `docs/` that the
annotation becomes the second copy. The trigger to revisit is these checks
failing routinely, which would be evidence the derivation is mechanical after
all.
