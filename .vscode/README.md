---
directory: .vscode
purpose: Editor configuration that makes the repository's own rules easy to satisfy.
owner: "@FabioCaffarello"
allowed:
  - Settings that mirror what make already enforces
  - Recommended and unwanted extensions
forbidden:
  - Personal preferences (theme, font, keybindings)
  - Settings that contradict .editorconfig, the Makefile, or CI
  - Tool versions
  - Anything that changes what a build produces
---

# .vscode

Committed on purpose, and kept small.

## What belongs here

Only settings that make the repository's existing rules easy to satisfy —
`goimports` local prefix matching `.golangci.yaml`, `-race` matching
`make test`, `staticcheck` matching `make lint`.

A setting that contradicts `make` is a bug: the editor would report green while
the gate reports red, or the reverse. If the two disagree, the Makefile wins and
this file is wrong.

## What does not belong here

Theme, font, keybindings, panel layout. Those are user settings. Committing them
makes the repository fight the person using it, which is how a committed
`settings.json` earns its bad reputation.

## `go.work` and the editor

`go.work` is committed for exactly this directory's benefit: cross-module
navigation while editing (ADR-0004).

Every `make` target and every CI job sets `GOWORK=off`, so module resolution
goes through published versions and the open-core boundary stays proven. The
editor is the deliberate exception, and the only one.

## Other editors

**JetBrains (GoLand, IntelliJ).** `.idea/` is *not* committed and stays in
`.gitignore`. It mixes project configuration with machine-local paths and
per-user state in the same files, so committing it produces constant noise and
occasional breakage for whoever opens it next. GoLand reads `.editorconfig`
natively, which covers formatting; point it at `golangci-lint` and enable
`goimports` with the local prefix above.

**Cursor.** Reads `.cursorrules` at the repository root, which points at
[`AGENTS.md`](../AGENTS.md).

**Claude Code.** Reads [`CLAUDE.md`](../CLAUDE.md), which also points at
`AGENTS.md`.

All three point at one file rather than restating it. A second copy of the
project's operating rules would drift, and the drifted copy is the one that gets
read.
