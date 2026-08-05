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
| `make analyze` | Domain purity and layer boundaries (the FDOS analysers) |
| `make repro-check` | Every command builds byte-reproducibly |
| `make tidy-check` | `go.mod`/`go.sum` are tidy in every module |
| `make fmt-check` / `fmt` | Go source is canonically formatted |
| `make vet` / `lint` / `test` / `build` | Standard Go targets, run per module |
| `make adr-immutability-check` | No accepted ADR has been rewritten since its introducing commit |
| `make action-pinning-check` | Every GitHub Action is pinned to a full commit SHA |
| `make secrets-check` | Full git history scanned for secrets (`gitleaks`) |
| `make vuln-check` | No known vulnerability reachable from FDOS code (`govulncheck`) |
| `make hooks` | Install the git hooks (`lefthook`) |
| `make affected` | Print the modules a change affects |
| `make clean` | Remove build output |

**`GOWORK=off` on every Go target.** This is the load-bearing half of ADR-0004:
it forces module resolution through published versions instead of local
workspace paths. Without it the open-core boundary silently stops being
verified, with nothing to indicate that it has stopped. `go.work` exists only
for editor convenience.

`GOFLAGS=-mod=readonly` is exported by the Makefile, so it holds whether or not
`mise` is installed. An implicit `go mod tidy` during a build is a silent,
unreviewed change to the dependency graph.

## Enforcement scripts

| Script | Enforces | Rung |
|--------|----------|------|
| `toolchain-check.sh` | §9 — pinned, reproducible toolchain | 3 |
| `verify-directory-contracts.sh` | §10 — declared architectural boundaries | 2–3 |
| `verify-adr.sh` | §14 — append-only decision log | 3 |
| `verify-rfc.sh` | §14 — design is decided before it is built | 3 |
| `verify-constitution-coverage.sh` | ADR-0005 — the ladder table stays honest | 3 |
| `run-analyzers.sh` | §2, §3, §10, §11 — domain purity and layering | 2 |
| `verify-reproducible-build.sh` | §9 — builds are byte-reproducible | 3 |
| `verify-tidy.sh` | §9 — the dependency graph is reviewed, not resolved at build time | 3 |
| `verify-gofmt.sh` | §9 — identical source for every checkout | 3 |
| `verify-adr-immutability.sh` | §14, ADR-0000 — the decision log is not rewritten | 3 |
| `verify-action-pinning.sh` | §9, ADR-0014 — build inputs are identified by digest | 3 |
| `verify-secrets.sh` | §13, §14 — no secret in history | 3 |
| `verify-vulns.sh` | §14 — no reachable vulnerability | 3 |
| `verify-commit-message.sh` | §14 — commit subject convention | 4 |
| `tool-version.sh` | shared helper — the single parser for `mise.toml` pins | — |
| `affected-modules.sh` | shared helper — the Nx compensation (ADR-0004) | — |
| `list-modules.sh` | shared helper (ADR-0004 makes commands per-module) | — |
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

## CI and hooks

CI invokes `make` targets and contains no logic of its own (ADR-0014). The
narrow exception is tool installation, and even there the **version comes from
`mise.toml`** through `scripts/tool-version.sh` — CI never declares a version of
its own, so the pins developers use and the pins CI uses cannot diverge.

Every GitHub Action is pinned to a full commit SHA. A tag can be moved under the
repository with no commit here, which is an unreviewed third party with write
access to the build.

`lefthook.yml` runs a fast subset pre-commit and the full `make verify`
pre-push. Hooks call the same `make` targets, and are explicitly bypassable: CI
re-runs everything, so `--no-verify` costs a round trip and cannot let anything
through.

## Not yet present

`buf` is pinned but unused — there is no proto yet (M4). Devcontainer and IDE
settings are **M3.5**.
