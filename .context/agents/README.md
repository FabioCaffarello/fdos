# Agent Handbook

Role playbooks for AI agents working on FDOS.

Every playbook here describes a role that has a real subject in this repository
today. The default scaffold roster was pruned at M1 — see
[`.context/README.md`](../README.md) for what was removed and why.

## Available agents

| Agent | Role | Since |
|-------|------|-------|
| [Architect Specialist](./architect-specialist.md) | Author and review ADRs and RFCs; defend architectural boundaries | M1 |
| [Code Reviewer](./code-reviewer.md) | Review changes against the Constitution and the enforcement ladder | M1 |
| [Documentation Writer](./documentation-writer.md) | Maintain `docs/` and the directory contracts | M1 |
| [Security Auditor](./security-auditor.md) | Toolchain pinning, secret hygiene, supply chain posture | M1 |
| [Test Writer](./test-writer.md) | Tests that can fail for the right reason; fixtures proving specificity | M2.5 |
| [DevOps Specialist](./devops-specialist.md) | Pipeline, hooks and supply chain, without logic leaking into YAML | M2.5 |

`test-writer` and `devops-specialist` returned at M2.5 because they acquired a
subject: M2 produced tests, M3 produced a pipeline. They were removed at M1
rather than left describing a repository that did not exist.

## Relative links must resolve from both locations

These files are symlinked into `.claude/agents/` (ADR-0017), and a relative link
inside a symlinked file resolves from **the symlink's** directory, not the
target's.

`../../docs/...` and `../skills/...` work from both, because `.claude/` mirrors
that structure. `../docs/...` does **not**: there is no `.claude/docs/`. A link
written that way is dead for every agent reading through Claude Code — which is
every agent.

Reference `.context/docs/*` by backticked path rather than by link.
`make context-check` catches the breakage, but only after it is written.

## Prompt contracts

Every playbook declares, in front matter, what it reads before acting
(`must_read`), what is out of bounds (`must_not`), and what it produces
(`evidence`). `make agent-contract-check` fails on a missing field or a
`must_read` path that does not exist.

This checks what is checkable. Whether an agent actually reads those paths or
honours those limits cannot be verified from outside, and a green check is not
evidence that it did (ADR-0015).

## Missing roles are still deliberate

There is no backend, frontend, database, mobile, feature, bug-fixer,
refactoring or performance agent. FDOS has no services, user interface,
database, or features. `database-specialist` is excluded permanently: a
first-class database role invites exactly the coupling Constitution §10 forbids.

`.context/README.md` records every absence with its reason.

## Before acting

Every agent working in this repository reads, in order:

1. [`docs/constitution.md`](../../docs/constitution.md) — the fourteen
   principles and the enforcement ladder. Highest authority in the repository.
2. The relevant ADRs in [`docs/adr/`](../../docs/adr/).
3. The `README.md` of any directory being changed — its front matter is that
   directory's binding contract.

An agent proposing a change that contradicts an accepted ADR must say so
explicitly and propose the superseding ADR. It must never quietly work around
one.

## Verification

No agent reports work as complete without `make verify` passing. An agent that
cannot run it says so rather than assuming.
