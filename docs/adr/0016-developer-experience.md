---
id: ADR-0016
title: One entry point, one source of truth, and no second copy of the rules
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0016 — One entry point, one source of truth, and no second copy of the rules

## Context

M3.5 is developer experience: devcontainer, editor configuration, task
ergonomics, and support for the tools people actually use — VS Code, JetBrains,
Cursor, Claude Code.

Every item on that list is a candidate second copy of something the repository
already states. A devcontainer that pins Go duplicates `mise.toml`. A task
runner duplicates the Makefile. A `CLAUDE.md` restating the rules duplicates
`AGENTS.md`. In each case the duplicate is cheap to add, invisible when it
drifts, and — because it is closer to hand — the one that gets read.

The original M3.5 sketch listed "task runner" as a deliverable. This ADR
declines it, with evidence.

## Decision

### `make` stays the only entry point

No Taskfile, no Just, no npm scripts. A second runner would be a second place
checks are defined, and CI runs `make` (ADR-0014).

The usual argument for adding one is speed. Measured on the current tree, the
full gate is **9.2 seconds**:

```
test         3.20s     vuln-check   1.22s     context-check  1.14s
repro-check  0.56s     analyze      0.55s     lint           0.42s
                       …everything else under 0.5s
```

That does not justify a second path. A `make quick` was considered and rejected
for the same reason: a fast target that omits checks eventually becomes the one
people run, and the gate becomes the thing that surprises them at push time.

**Revisit when the full gate exceeds roughly 60 seconds.** At that point the fix
is `scripts/affected-modules.sh` driving a separate CI job — not a narrower local
gate.

### Editor configuration is committed, and scoped

`.vscode/settings.json` and `.vscode/extensions.json` are committed, restricted
to settings that make the repository's own rules easy to satisfy: `goimports`
local prefix matching `.golangci.yaml`, `-race` matching `make test`,
`staticcheck` matching `make lint`.

Theme, font, keybindings and layout are user settings and stay out. A committed
`settings.json` that carries personal preference makes the repository fight the
person using it, which is how the practice earned its reputation.

A setting here that contradicts `make` is a bug: the editor would report green
while the gate reports red. The Makefile wins.

### `.idea/` is not committed

JetBrains mixes shared project configuration with machine-local paths and
per-user state in the same files. Committing it produces continuous noise and
occasional breakage for whoever opens it next.

GoLand reads `.editorconfig` natively, which covers formatting — the part that
actually needs to be shared. The rest is documented in `.vscode/README.md`.

### One agent entry point

`AGENTS.md` is the entry point for every agent. `CLAUDE.md` and `.cursorrules`
exist because those tools look for those names, and each is a **pointer**, not a
copy.

Both carry exactly two facts inline — that there is no domain code by design,
and that `make verify` is the gate — because those are the two an agent most
needs before it opens anything, and both are stable across milestones.

### The devcontainer is not a build input

The container installs `mise`, and `mise` installs the toolchain from
`mise.toml`. No version is declared in `.devcontainer/`. Devcontainer *features*
for Go were rejected because a feature declares its own version.

The image is pinned by tag rather than digest, and the `mise` installer is
fetched with `curl | sh`. Both would be unacceptable in `.github/workflows/`.
They are acceptable here for one specific reason: **CI does not use this
container, and nothing built in it is ever released or attested to.** Releases
are built by `release.yml` on a GitHub runner with every action digest-pinned.

If a release is ever built in this container, it becomes a build input and must
be pinned like every other one.

### `make doctor` never fails

It diagnoses and always exits zero. `make toolchain-check` answers "is this
correct" and fails; `doctor` answers "why does this not work on my machine",
which is a question only asked when something is already broken. A diagnostic
that exits non-zero cannot be chained, and gets skipped exactly when it is
needed.

Every finding names its fix. A doctor that reports symptoms without remedies
relocates the confusion rather than resolving it.

## Consequences

### Positive

- No configuration is stated twice, so nothing can drift into contradiction.
- A new contributor runs `make doctor` and gets a list of problems with fixes,
  rather than a failure they have to interpret.
- The devcontainer cannot drift from `mise.toml`, because it contains no
  version to drift.

### Negative

- **Committed editor settings are contentious**, and the line between "helps
  satisfy the rules" and "personal preference" is a judgement that will be
  argued at the margin. The rule stated above is the tiebreaker, not a
  formula.
- Contributors using JetBrains get less out of the box than VS Code users. That
  is a real asymmetry, accepted because committing `.idea/` costs everyone to
  benefit some.
- Declining a task runner means `make`'s ergonomics are permanent. Make is a
  poor language, and complex targets will stay awkward.
- The devcontainer exception for `curl | sh` is a genuine inconsistency with the
  supply-chain rules. It rests entirely on "CI does not use it", which is true
  today and is exactly the kind of premise that quietly stops being true.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| Directory contracts for `.devcontainer`, `.vscode` | 3 | `make contracts-check` |
| Pointers do not rot into copies | 3 | `make context-check` — links must resolve |
| No version declared outside `mise.toml` | 6 | review |
| Editor settings agree with `make` | 6 | review |

The last two are rung 6 and there is no cheap mechanism for either. A check
that a devcontainer contains no version string is possible but would be
trivially evaded; the honest position is that review carries it.

## Alternatives considered

**Add a task runner (Task, Just) for ergonomics.** Rejected on the measurement
above: there is no speed problem to solve, and it would create a second
definition of the checks. Worth revisiting only if `make`'s expressiveness — not
its speed — becomes the constraint.

**Pin Go in the devcontainer with a devcontainer feature.** Simpler, and the
conventional approach. Rejected: it declares a version, which makes
`.devcontainer/` a second source of truth alongside `mise.toml`.

**Commit `.idea/` as well, for parity.** Rejected on the noise argument. Sharing
run configurations only, without the rest, is worth reconsidering if JetBrains
usage grows.

**Restate the operating rules in `CLAUDE.md` and `.cursorrules` so agents need
one file.** Rejected: three copies of the rules is three places to update and
one place that will be wrong. The pointer costs an agent one extra read.

**Skip the devcontainer entirely; `mise install` is enough.** Genuinely
tempting — it is what a local contributor does anyway. Kept because it gives a
zero-configuration path for someone evaluating the project, and because it
documents the environment executably rather than in prose.

## Notes

### Two conflicts this milestone surfaced

**The decision log cannot satisfy the freshness check.** ADR-0000 makes accepted
ADRs immutable; ADR-0015 requires documentation to name only things that exist.
An ADR recording a *rejected* alternative names something that deliberately does
not exist — and if a `make` target is renamed in five years, every ADR citing it
becomes "stale" with no permitted correction.

`docs/adr/` and `docs/rfc/` are therefore exempt from the mutable-name checks
(`make` targets, script paths). Links and ADR/RFC identifiers are still verified
there, because those name stable things and a broken one is a real defect rather
than the passage of time. The exemption was negative-tested in both directions.

This surfaced because this very ADR names `make quick`, the fast target it
declines to create.

**Generated AI-tool exports contradict the pruned roster.** `.claude/` is
produced by exporting `.context/` for Claude Code, and the exporter re-adds its
built-in skills — including `api-design`, `refactoring` and `bug-investigation`,
which `.context/` deliberately removed for having no subject.

It is gitignored and excluded from the checks: it is derived, regenerable, and
not authoritative. Versioning it would put a contradicting second roster in the
repository, which is exactly the failure ADR-0015 exists to prevent. `AGENTS.md`,
`CLAUDE.md` and `.cursorrules` all point at `.context/` for this reason.

### Open, deliberately

- The `curl | sh` in `.devcontainer/post-create.sh` is inconsistent with the
  digest-pinning rule everywhere else. It is bounded by "not a build input",
  which is a premise worth re-checking whenever release plumbing changes.
- Nothing verifies that `.vscode/settings.json` agrees with `.golangci.yaml` and
  the Makefile. A check comparing the `goimports` local prefix across the three
  is feasible and small; it was not built because there is one prefix and it
  would be enforcement theatre at this size.
