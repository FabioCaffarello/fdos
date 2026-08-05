# AGENTS.md

Entry point for AI agents working on FDOS. Read this first, then the sources it
points at — this file is a signpost, not a substitute.

## Read before acting

1. [`docs/constitution.md`](docs/constitution.md) — fourteen principles and the
   enforcement ladder. **Highest authority in the repository.**
2. [`docs/adr/`](docs/adr/) — accepted decisions. Append-only and immutable.
3. The `README.md` front matter of any directory you change — that front matter
   is the binding contract for the directory.
4. [`CONTRIBUTING.md`](CONTRIBUTING.md) — workflow and definition of done.
5. [`.context/`](.context/README.md) — structured knowledge derived from `docs/`.

## There is no code yet, and that is deliberate

FDOS has **no Go code**, no `go.mod`, no tests, no CI pipeline and no
application. The canonical financial model is an output of the M1.5 RFCs.

If asked to "add a feature", the correct response is that there is nothing yet
to add it to, followed by a pointer to the roadmap. Do not create domain types,
module files or business rules to make progress visible: doing so settles open
RFC questions by accident, which is the most damaging thing that can be done to
this repository at its current stage.

## Commands

```sh
make bootstrap   # validate the toolchain against the pins in mise.toml
make verify      # every enforcement mechanism available at this milestone
make help        # list targets
```

There is no npm, no Jest, no TypeScript and no `dist/`. The toolchain is Go plus
`make`, pinned in `mise.toml`.

Never report work as complete without `make verify` passing. If you cannot run
it, say so rather than implying you did.

## Repository map

| Path | Contents |
|------|----------|
| `libs/` | Reusable libraries, one Go module per subdirectory. Empty until M2. |
| `apps/` | Deployable applications, composition roots only. Empty. |
| `docs/` | Constitution, ADRs, RFCs. Authoritative. |
| `deploy/` | Deployment topology. Empty. |
| `examples/` | Executable demonstrations of the public contracts. Empty until M4. |
| `scripts/` | Enforcement mechanisms, invoked through `make`. |
| `.github/` | CI workflows. Empty until M3. |
| `.context/` | Knowledge for agents, derived from `docs/`. |

Directories are empty **by design**, not by omission. Each one's `README.md`
states what may and may not live there.

## Rules that will get a change rejected

- Editing an accepted ADR to change its meaning. Reverse by superseding
  (ADR-0000); `make adr-check` validates the link in both directions.
- A structural change with no ADR.
- A new enforcement mechanism with no negative test. A check that has never gone
  red is unverified.
- Documentation left stale by the same change.
- Anything routing LLM output toward the ledger. Models explain; they are never
  the source of financial truth (Constitution §2).
- Implementing against content marked *provisional*. That converts a hypothesis
  into a decision without an ADR.

## Commits and pull requests

Conventional prefixes (`feat`, `fix`, `docs`, `chore`, `refactor`, `test`,
`build`). The body records **why**, states the costs accepted, and names any
superseded decision. The diff already says what changed.

## When you disagree

If a request contradicts an accepted ADR, say so explicitly and propose the
superseding ADR. Never quietly work around one.

If the Constitution itself is the obstacle, say that too. It is amendable
through an RFC, a version bump and an ADR. It is not ignorable.
