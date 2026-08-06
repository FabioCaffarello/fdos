# CLAUDE.md

Read [`AGENTS.md`](AGENTS.md). It is the entry point for every agent working on
FDOS, regardless of which tool is driving.

This file exists only because Claude Code looks for it by name. Restating the
rules here would create a second copy, and the drifted copy is always the one
that gets read.

Two things worth knowing before opening anything else:

- **The domain code has landed.** `libs/kernel` and `libs/ledger` exist, and
  `libs/contracts` is consumed outside this repository at a pinned version. The
  sequencing rule survives that: a new bounded context, canonical type or
  published message still needs the ADR that sequences it first.
- **`make verify` is the whole gate.** CI runs exactly it (ADR-0014). Never
  report work complete without it passing, and say so plainly if you could not
  run it.
