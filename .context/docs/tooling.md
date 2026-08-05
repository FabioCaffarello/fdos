---
type: doc
name: tooling
description: Toolchain, Makefile targets, enforcement scripts, and how they fit together
category: tooling
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Tooling

## Toolchain

`mise.toml` is the single source of truth for tool versions.

| Tool | Pin | Required from |
|------|-----|---------------|
| go | 1.26.2 | M0 |
| golangci-lint | 2.12.2 | M2 |
| buf | 1.68.4 | M4 |

**`mise` itself is not a prerequisite.** `scripts/toolchain-check.sh` parses
`mise.toml` directly and validates whatever is on `PATH`, so the pin is enforced
identically for someone using mise, someone installing by hand, and CI.

A wrong version is always an error — a wrong version is worse than no version. A
tool not yet required is reported when absent but does not fail the build.

```sh
mise install     # if you use mise
make bootstrap   # either way: verify what you have
```

## Make is the only entry point

Every script is reachable from a `make` target. A script that must be invoked
directly is a script that will be forgotten and then silently stop working.

From M3, CI invokes `make` targets and contains no logic of its own. If a check
exists only in workflow YAML it cannot be run locally, cannot be debugged
without pushing, and drifts from what developers actually execute.

| Target | Purpose |
|--------|---------|
| `make help` | List targets |
| `make bootstrap` | Prepare a working copy; validate the toolchain |
| `make verify` | Every enforcement mechanism available at this milestone |
| `make toolchain-check` | Installed tools match the pins |
| `make contracts-check` | Every directory declares a valid contract |
| `make adr-check` | Decision log well-formed; supersession bidirectional |
| `make rfc-check` | RFC set well-formed; an Accepted RFC produced ADRs |
| `make constitution-check` | Every principle appears in the §15 enforcement table |
| `make clean` | Remove build output |

`fmt`, `lint`, `test` and `build` arrive at M2 with the first Go code. They are
absent rather than stubbed: a target that does nothing is worse than a missing
one, because it reports work that did not happen.

## Enforcement scripts

| Script | Enforces | Rung |
|--------|----------|------|
| `toolchain-check.sh` | §9 — pinned, reproducible toolchain | 3 |
| `verify-directory-contracts.sh` | §10 — declared architectural boundaries | 2–3 |
| `verify-adr.sh` | §14 — append-only decision log | 3 |
| `verify-rfc.sh` | §14 — design is decided before it is built | 3 |
| `verify-constitution-coverage.sh` | ADR-0005 — the ladder table stays honest | 3 |
| `lib/frontmatter.sh` | shared helper | — |

**They deliberately use nothing beyond `bash`, `awk` and `grep`.** These are the
first thing that runs on a clean clone, before any language toolchain is
guaranteed to exist, and they must never be the reason a fresh checkout fails to
verify. They also run on macOS's bash 3.2, so no associative arrays and no
`${var,,}`.

## Adding a check

1. Write it in `scripts/`, with a header naming the principle and ladder rung.
2. Add a `make` target; wire it into `verify`.
3. Test it against negative cases — break the invariant, confirm a useful
   failure, restore.
4. Update Constitution §15.

Step 3 caught real defects in two of the four existing checks. Skipping it
produces checks that are green because they cannot fail.

## Not yet present

`golangci-lint` and `buf` are pinned but unused — there is no Go code and no
proto. Devcontainer, IDE settings and task ergonomics are **M3.5**. CI is **M3**.
