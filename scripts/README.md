---
directory: scripts
purpose: Enforcement mechanisms and repository automation invoked through the Makefile.
owner: "@FabioCaffarello"
allowed:
  - Fitness functions that enforce Constitution principles
  - Repository automation (verification, release, code generation)
  - Shared shell helpers under scripts/lib/
forbidden:
  - Business logic or financial calculations
  - Scripts not reachable from a Makefile target
  - Dependencies on tools absent from mise.toml
  - Undocumented behaviour — every script states which principle it enforces
---

# scripts

This directory holds the mechanisms that make the Constitution executable. Its
contents are not conveniences; they are the difference between a principle and a
wish.

## Contents

| Script | Enforces | Ladder rung |
|--------|----------|-------------|
| `toolchain-check.sh` | Constitution §9 — pinned, reproducible toolchain | 3 (CI) |
| `verify-directory-contracts.sh` | Constitution §10 — declared architectural boundaries | 2–3 |
| `verify-adr.sh` | Constitution §14 — append-only decision log | 3 (CI) |
| `lib/frontmatter.sh` | shared helper, enforces nothing on its own | — |

## Rules

**Every script is reachable from `make`.** A script that must be invoked
directly is a script that will be forgotten and then silently stop working. The
Makefile is the only entry point.

**Every script names the principle it enforces**, in a header comment, together
with its position on the enforcement ladder (docs/constitution.md §15). A script
that cannot name its principle is automation, not enforcement, and should be
justified on other grounds.

**No unpinned dependencies.** Scripts may only use tools declared in
`mise.toml`, or POSIX utilities. The M0 verification scripts deliberately use
nothing beyond `bash`, `awk` and `grep`: they are the first thing that runs on a
clean clone, before any language toolchain is guaranteed to exist, and they must
never be the reason a fresh checkout fails to verify.

**Failures explain themselves.** A check that fails must say what was violated,
where, and what to do about it. A red build that requires reading the script to
understand is a broken check.

## Adding a check

1. Write the script here, with a header naming its principle and ladder rung.
2. Add a `make` target for it, and wire that target into `verify`.
3. Update the enforcement table in `docs/constitution.md` §15 — a principle that
   has climbed a rung must be recorded as having climbed.
