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

The scaffold generates a full roster of roles and skills by default. FDOS keeps
only those describing something that exists in this repository today.

Removed at M1, with the reason:

| Removed | Reason |
|---------|--------|
| `database-specialist` | The domain must not depend on a database (Constitution §10). A first-class database role invites exactly the coupling that principle forbids. |
| `frontend-specialist`, `mobile-specialist` | FDOS has no user interface and no mobile surface. |
| `backend-specialist`, `feature-developer`, `bug-fixer`, `refactoring-specialist`, `performance-optimizer`, `test-writer` | No Go code exists yet. These describe work on a codebase that has not been written. |
| `devops-specialist` | CI/CD is M3. `.github/` is deliberately empty. |
| `api-design` | API design is generated from contracts at M4. A skill now would pre-judge the proto → buf → OpenAPI chain. |
| `bug-investigation`, `refactoring`, `security-audit`, `test-generation` | No code to investigate, refactor, audit or test. |

They are regenerated at the milestone that gives them a subject — most at
**M2.5 (AI Engineering)**, which is where agent playbooks are a deliverable
rather than a side effect of scaffolding.

This is not tidiness. A playbook describing structure that does not exist is
worse than no playbook: an agent will act on it, and nothing reports that it was
wrong. M2.5 adds a staleness check in CI so that this failure mode becomes
detectable rather than latent.

## Derivation

These files are written by hand today. They are intended to become **generated**
from `docs/` at M2.5, so that drift between the human record and the agent
record is impossible rather than merely discouraged.

Until then, any change to the Constitution, an ADR, or a directory contract
obliges a corresponding update here. That obligation currently rests on human
discipline — rung 6, and recorded as such.
