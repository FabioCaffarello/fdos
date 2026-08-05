# Skills

On-demand procedures for AI agents working on FDOS. A skill is activated when a
task matches its description.

The default scaffold roster was pruned at M1 to those procedures actually
performed in this repository today — see [`.context/README.md`](../README.md)
for what was removed and why.

## Available skills

| Skill | Description | Phases |
|-------|-------------|--------|
| [Feature Breakdown](./feature-breakdown/SKILL.md) | Break work into ADR- and RFC-shaped units within the milestone roadmap | P |
| [Documentation](./documentation/SKILL.md) | Write and update `docs/`, directory contracts, and `.context/` | P, C |
| [Code Review](./code-review/SKILL.md) | Review changes against the Constitution and the enforcement ladder | R, V |
| [PR Review](./pr-review/SKILL.md) | Review a pull request end to end, including its governance obligations | R, V |
| [Commit Message](./commit-message/SKILL.md) | Write commit messages that record reasoning, not just change | E, C |

Built-in `dotcontext-*` workflow skills remain available and are not managed
here.

## Missing skills are deliberate

There is no api-design, test-generation, refactoring, bug-investigation or
security-audit skill. API design is generated from contracts at M4 and a skill
now would pre-judge that chain; the rest have no code to act on. They return at
the milestone that gives them a subject.

## Creating a skill

```
.context/skills/
└── my-skill/
    ├── SKILL.md          # required
    ├── scripts/          # optional: deterministic helpers
    ├── references/       # optional: load-on-demand detail
    └── assets/           # optional: output resources
```

Keep activation language in the `description` front matter and the body concise.
A skill that describes a procedure FDOS does not actually perform is removed,
not left to rot.

## PREVC phase mapping

| Phase | Name | Skills |
|-------|------|--------|
| P | Planning | feature-breakdown, documentation |
| R | Review | pr-review, code-review |
| E | Execution | commit-message |
| V | Validation | pr-review, code-review |
| C | Confirmation | commit-message, documentation |
