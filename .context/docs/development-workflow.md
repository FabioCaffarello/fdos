---
type: doc
name: development-workflow
description: How work is proposed, decided, reviewed and landed in FDOS
category: workflow
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Development Workflow

## Architecture before implementation

FDOS reviews architecture before code exists (Constitution §14). The order is
not negotiable and is the reason M1.5 produces RFCs and no code at all.

```
Question → RFC (if design exploration needed) → ADR (decision) → implementation
```

Skipping straight to implementation is the single most common way to damage this
repository, because code that ships becomes the decision, and the reasoning is
never recorded.

## When an ADR is required

Any change to:

- repository structure or directory contracts
- module boundaries or the public contract surface
- the toolchain or pinned versions
- enforcement mechanisms
- the Constitution itself

Write an **ADR** when the decision is clear. Write an **RFC** first when it needs
exploration; an accepted RFC is followed by the ADRs recording what it settled.

## The decision log is append-only

An accepted ADR is never edited to change its meaning (ADR-0000). A decision
that proves wrong is superseded:

1. New ADR: `supersedes: [ADR-NNNN]`, status `Accepted`.
2. Old ADR: status → `Superseded`, `superseded_by: [ADR-MMMM]`, original text
   left **unaltered**, banner added pointing at the successor.

`make adr-check` validates both directions of the link and rejects a predecessor
still marked `Accepted`. ADR-0001 → ADR-0006 is the worked example in the repo.

Fixing a typo is fine. Changing what a decision *says* is not.

## Verification

```sh
make bootstrap   # validate toolchain against mise.toml pins
make verify      # every enforcement mechanism available at this milestone
make help        # list targets
```

`make verify` currently runs:

| Check | Enforces |
|-------|----------|
| `toolchain-check` | Installed tools match the `mise.toml` pins |
| `contracts-check` | Every directory declares a valid architectural contract |
| `adr-check` | The decision log is well-formed and supersession is bidirectional |
| `constitution-check` | Every Constitution principle appears in the §15 enforcement table |

A clean clone must pass `make verify` with no tribal knowledge. If it does not,
that is a repository bug.

## Adding an enforcement mechanism

1. Write the script in `scripts/`, with a header naming the principle it
   enforces and its ladder rung.
2. Add a `make` target and wire it into `verify`.
3. **Test it against negative cases.** A check that has never gone red is
   unverified. Break the invariant deliberately, confirm the check fails with a
   useful message, restore.
4. Update the enforcement table in `docs/constitution.md` §15.

Step 3 is not optional and is not ceremony. Two of the M0 fitness functions had
real defects that only negative testing surfaced.

## Commits

Commit messages record **reasoning**, not just change. A reader in five years
needs to know why, and the diff already tells them what.

State trade-offs and costs accepted. If a decision was reversed, say so and name
the superseding ADR.

## Definition of done

- `make verify` passes
- Any structural change has an ADR
- Any new enforcement mechanism has negative tests
- Constitution §15 reflects reality
- Documentation updated in the same change, not afterwards

Documentation is production code. A change that leaves `docs/` stale is not
finished.
