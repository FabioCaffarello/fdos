# CLAUDE.md

Read [`AGENTS.md`](AGENTS.md). It is the entry point for every agent working on
FDOS, regardless of which tool is driving.

This file exists only because Claude Code looks for it by name. Restating the
rules here would create a second copy, and the drifted copy is always the one
that gets read.

Two things worth knowing before opening anything else:

- **There is no domain code, deliberately.** The canonical model is decided
  (ADR-0007 … ADR-0012) but lands with the Ledger at M6. Creating `libs/kernel`
  or a bounded context ahead of that is the most damaging thing that can be done
  to this repository right now.
- **`make verify` is the whole gate.** CI runs exactly it (ADR-0014). Never
  report work complete without it passing, and say so plainly if you could not
  run it.
