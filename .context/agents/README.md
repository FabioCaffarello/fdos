# Agent Handbook

Role playbooks for AI agents working on FDOS.

Every playbook here describes a role that has a real subject in this repository
today. The default scaffold roster was pruned at M1 — see
[`.context/README.md`](../README.md) for what was removed and why.

## Available agents

| Agent | Role |
|-------|------|
| [Architect Specialist](./architect-specialist.md) | Author and review ADRs and RFCs; defend architectural boundaries |
| [Code Reviewer](./code-reviewer.md) | Review changes against the Constitution and the enforcement ladder |
| [Documentation Writer](./documentation-writer.md) | Maintain `docs/` and the directory contracts |
| [Security Auditor](./security-auditor.md) | Toolchain pinning, secret hygiene, supply chain posture |

## Missing roles are deliberate

There is no backend, frontend, database, mobile, devops, test or refactoring
agent. FDOS has no Go code, no user interface, no database, no CI pipeline and
no tests. Those playbooks arrive at **M2.5 (AI Engineering)**, once they have
something to describe.

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
