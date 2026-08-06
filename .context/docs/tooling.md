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
| lefthook | 2.1.8 | M3 |
| gitleaks | 8.30.0 | M3 |
| govulncheck | v1.6.0 | M3 (via `go run`) |
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
| `make doctor` | Diagnose this working copy and name the fix for each problem |
| `make proto-check` | Contract surface: lint, format, breaking, pinning, drift, FDOS schema rules |
| `make proto-gen` | Regenerate Go from the proto schemas |
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
| `verify-proto.sh` | §2, §6, §7, §11 — the contract surface cannot change silently | 3 |
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

## The dotcontext harness: what is bound, and what was declined

The harness offers sessions, traces, sensors, artifacts, task contracts,
handoffs, replays, failure datasets, policy rules and a PREVC workflow. Sensors
and policy were bound in M9; **M9.5 bound the PREVC workflow, sessions and task
contracts as well** (ADR-0031), when the milestone calibration program gave the
runtime a subject. What remains declined is recorded rather than left looking
unnoticed.

### Bound, because they have a subject

**PREVC, sessions and task contracts** — since M9.5. PREVC was evaluated in M9
and declined: it was a second description of how work proceeds, and two process
descriptions is the drifted-copy problem this repository names for `CLAUDE.md`
and `AGENTS.md`. ADR-0031 reversed that the only way the objection allows — by
making PREVC the single description. `CONTRIBUTING.md` now names the working
agreement's stages with the PREVC letters; each milestone session opens with
`workflow-init`, carries a task contract whose acceptance criteria are the
gate's, and records `make verify` runs as the `verify` sensor. The declination
and the argument that lost are preserved in the ADR. Where the harness's
built-in PREVC skills and `CONTRIBUTING.md` disagree, `CONTRIBUTING.md` wins.

**Sensors** — `.context/config/sensors.json`. Two, and the file argues its own
smallness: `make verify` runs nineteen checks and enumerating them here would be
a second copy of a list the Makefile already holds. A sensor exists where there
is a distinct *execution context*, not a distinct check — which is why
`secrets-check-staged` is the second one, since scanning staged content and
scanning the tree are different questions.

**Policy** — `.context/config/policy.json`. It corresponds to the
`needs/human-decision` label the pull-request workflow already requires, and M9
extended it to the paths that had none: `docs/ecosystem/**` (Tier 0, vendored
downstream), `docs/rfc/**`, the published contract surface, and the disclosure
register.

### Declined, with reasons

**Handoffs, replays and failure datasets.** Machinery for multi-agent work.
One session works this repository at a time, and a handoff between an agent
and itself records nothing a pull request does not. They become interesting
when there is a second actor; there is not. (Task contracts moved out of this
list in M9.5: a contract carrying the gate's acceptance criteria and required
sensors is what makes calibration sessions comparable.)

**Tracking session state.** Sessions and traces are runtime state, correctly
untracked. The durable record of what happened here is still the decision log
and the pull request bodies, both of which outlive any session; the calibration
log summarizes what a session measured, and the raw state stays local.

### Defects measured while adopting the workflow (M9.5 pilot)

`plan link` rejects `required_sensors` on execution phases in every documented
format, while still half-registering the link — after which `approvePlan`'s two
code paths disagree about whether a plan is linked at all. The P→R and R→E
gates were passed with an explicit, trace-recorded `force`. Until fixed
upstream, the phase gates are honour-system plus traces. To be reported against
dotcontext alongside the `dryRun` defect below.

### A caution, learned expensively

`sync` accepts `dryRun` and **does not honour it**. A preview call wrote 102
files across four new top-level directories and turned the gate red; the count
it reported as `filesCreated` was a description of what it had already done, not
a projection. Assume the flag does nothing, and arrange to be able to undo any
call. Recorded in `docs/blocked.md` under B-004.

## Developer environment

`.devcontainer/` gives a zero-configuration path: it installs `mise`, and `mise`
installs the toolchain from `mise.toml`. **No version is declared there** —
devcontainer features for Go were rejected because a feature declares its own
(ADR-0016).

The container is *not* a build input. CI does not use it and nothing built in it
is released or attested to, which is the only reason its image is tag-pinned and
its installer arrives by `curl | sh`. Both would be unacceptable in
`.github/workflows/`.

`.vscode/settings.json` is committed, restricted to settings that mirror what
`make` already enforces — `goimports` local prefix, `-race`, `staticcheck`.
Theme, font and keybindings stay in user settings. `.idea/` is not committed:
JetBrains mixes shared configuration with machine-local paths in the same files.

`make doctor` diagnoses a working copy and **never fails**. `toolchain-check`
answers "is this correct" and exits non-zero; `doctor` answers "why does this not
work on my machine", a question only asked when something is already broken.

## No second task runner

There is no Taskfile, no Just, no npm scripts, and no abbreviated fast-path
target. `make` is the only entry point, and CI runs it (ADR-0014).

The full gate measured **9.2 seconds** at M3.5 — `test` 3.2s, `vuln-check` 1.2s,
`context-check` 1.1s, everything else under 0.6s. That does not justify a second
path, and a fast target that omits checks becomes the one people run.

Revisit above roughly 60 seconds, and the fix is then
`scripts/affected-modules.sh` driving a separate CI job — not a narrower local
gate.

## Not yet present

Everything pinned in `mise.toml` is now in use.
