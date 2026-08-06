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

## The plan-gate

**A slice starts as a draft pull request carrying the plan**: the objective
restated, what was read, what could not be found, the slice, the boundary tests
applied, and the costs accepted. No work happens while it is a draft.

**Marking it ready is the approval, and the decision to mark it is never the
session's.** An author who is also the approver makes the gate theatre on its
first use. Where the human has no GitHub access — the normal case here — the
session may perform the transition **only on an explicit instruction naming the
pull request**, and says in that turn that it did so and on whose instruction.

**A review that changes a decision is recorded in the decision.** If the
reasoning stays in a review thread, the decision is still conversational, now
with a green artifact on top. The artifact carries the whole argument in its own
text — including the argument that lost and the one the author withdrew. The
test: a reader who saw none of the review can reconstruct why the decision has
its shape, from the artifact alone.

## Pull requests

Since M5, `main` is protected: direct pushes are impossible and every change
goes through a pull request with a green `verify` (ADR-0020).

`.github/pull_request_template.md` carries what the gate cannot check — whether
an ADR is required, whether a new mechanism was negative-tested, whether §15
moved, and whether documentation was updated in the same change.

Required approvals is 0 because a solo repository with one required approval
cannot merge anything. It rises to 1 with a second maintainer.

Work that cannot be finished is registered as a GitHub issue carrying the
`status/blocked` label (ADR-0032) rather than silently omitted: an unrecorded
block becomes an unexplained gap. `docs/blocked.md` is the frozen index of the
pre-M9.5 entries, B-001 … B-009, and their identifiers stay citable.

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
| `rfc-check` | The RFC set is well-formed, and an `Accepted` RFC produced the ADRs recording it |
| `constitution-check` | Every Constitution principle appears in the §15 enforcement table |
| `analyze` | Domain purity and layer boundaries (`nofloat`, `nondet`, `impurity`, `layering`) |
| `repro-check` | Every command builds byte-reproducibly |
| `adr-immutability-check` | No accepted ADR rewritten since its introducing commit |
| `action-pinning-check` | Every GitHub Action pinned to a full commit SHA |
| `secrets-check`, `vuln-check` | Full-history secret scan; reachable vulnerabilities |
| `tidy-check`, `fmt-check`, `vet`, `lint`, `test` | Standard Go hygiene, per module, with `GOWORK=off` |

CI runs `make verify` and nothing else (ADR-0014), so a green local run is a
meaningful prediction of a green pipeline. Git hooks (`lefthook`) run the fast
subset on commit and the full gate on push; they are bypassable because CI
re-runs everything regardless.

A clean clone must pass `make verify` with no tribal knowledge. If it does not,
that is a repository bug.

## Adding an enforcement mechanism

Every mechanism ships with a **negative test** — the invariant broken
deliberately, the check failing with a useful message, the tree restored — and
answers the **execution-context question** in its pull request and its ADR:

> Where does this check actually run? What can it observe there that differs
> from the fixture? What would it report if its subject were simply absent?
>
> **Silence is the answer that should worry you.**

That question has caught real gaps: a rule stated with no execution site, a
check that globbed the wrong directory, and an ADR that understated the
enforcement already available to it.

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
