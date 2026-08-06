# Contributing to FDOS

FDOS is built to be maintained for a decade. Almost every rule here exists
because something valuable is lost by default over that horizon — usually the
reasoning behind a decision, occasionally the ability to reproduce a number.

Read [`docs/constitution.md`](docs/constitution.md) first. It is the highest
authority in this repository, and it is short.

## Getting started

```sh
make doctor      # what is installed, what is missing, what to do about it
make bootstrap   # validate the toolchain, install git hooks
make verify      # run every enforcement mechanism available at this milestone
make help        # list targets
```

Or open the repository in the [devcontainer](.devcontainer/README.md) and skip
the installation entirely. `make doctor` never fails — it is a diagnostic, and a
diagnostic that exits non-zero cannot be run when things are broken.

A clean clone must pass `make verify` with no tribal knowledge. If it does not,
that is a bug in this repository, not in your machine — please report it.

`mise` is recommended but not required. `make toolchain-check` reads `mise.toml`
directly and validates whatever is on your `PATH`.

### Git hooks

`make bootstrap` installs them via `lefthook`. Pre-commit runs the fast checks —
formatting, the governance logs, a staged-content secret scan. Pre-push runs the
full `make verify`.

Hooks are a convenience, never the guarantee. `--no-verify` is fine when you
know what you are doing: CI runs the full gate regardless, so bypassing costs
you a round trip and cannot let anything through.

## Architecture before implementation

```
Question → RFC (if design exploration is needed) → ADR (decision) → implementation
```

This ordering is not negotiable (Constitution §14). Code that ships becomes the
decision, and the reasoning is never recorded afterwards.

The canonical model is decided — RFC-0001 … RFC-0007, recorded in ADR-0007 …
ADR-0012 and ADR-0022 — and it has landed as code. M6 built `libs/kernel` and
`libs/ledger` under those constraints rather than retrofitting them, and M7
added the conformance suites that keep each type's domain and wire definitions
in agreement.

The sequencing rule outlives the empty repository it was written for. A
contribution adding a **second** bounded context, a canonical type, or a message
to the published contract surface ahead of the ADR that sequences it will be
declined: not because it is bad work, but because sequencing is itself a
decision (ADR-0013). The contract surface is the sharp edge — it is consumed
outside this repository at a pinned version, so adding to it is a change to
somebody else's build, and it starts with an issue and an RFC.

## The plan-gate, and who closes it

A slice starts as a **draft pull request carrying the plan**: the objective
restated, what was read, what could not be found, the slice itself, the boundary
tests applied, and the costs accepted. No work happens while it is a draft.

**Marking it ready is the approval, and the decision to mark it is never the
session's.** An approver and an author who are the same actor make the gate
theatre on its first use.

The *decision* and the *keystroke* are separate, and only the first is
load-bearing. Where the human has no GitHub access — which is the normal case
here, since every `gh` call in this repository is made by the session — the
session may perform the transition **only on an explicit instruction naming the
pull request**, and says in that turn that it did so and on whose instruction.
What it may never do is decide that its own slice is ready.

This rule exists because the gate was designed with an approver who had no
hands. The brief said approval was the ready transition, and the reviewer
issuing that instruction could not perform it: the approval lived in a
conversation while the artifact still said draft. Two slices ran on an approval
that had no referent before anyone noticed.

The first version of this rule then made the opposite error — it forbade the
session from performing the transition at all, which in an operating model where
the session holds the only keyboard means the gate can never open. Recorded
rather than quietly rewritten, because a rule that has been wrong in both
directions is worth showing the shape of.

### A review that changes a decision is recorded in the decision

Approval alone does not make a decision non-conversational. If the reasoning
that shaped it stays in a review thread, the decision is still conversational —
now with a green artifact on top, which is worse.

So: **when review changes the shape of a decision, the artifact records that
review changed it, and carries the whole argument in its own text.** Never a
pointer to a conversation, an issue thread, or a chat log.

That includes the argument that lost, and the argument the author themselves
withdrew. A decision log holding only the winner loses exactly the information
that makes it worth keeping — the next person cannot tell a considered rejection
from an option nobody thought of, nor a position abandoned on its merits from
one that was never held.
[RFC-0010](docs/rfc/0010-the-public-surface-receives-a-claim.md) opens by
recording that review inverted its subject, and states both the argument it
dropped and the one that replaced it. That is the pattern.

**The test:** a reader who has seen none of the review must be able to
reconstruct why the decision has the shape it has, from the artifact alone. If
they cannot, the artifact is incomplete regardless of how green it is.

## Git mechanics that have cost rework

Three now, which is a pattern rather than bad luck. Each is written here because
a note in a pull request dies with the pull request, and the next person to hit
it is you in November.

**A squash-merge with `--delete-branch` closes every pull request based on that
branch.** GitHub auto-closes them, and a closed pull request can be neither
reopened nor retargeted — it must be rebuilt as a new one, losing its review
history. **Retarget the child to `main` before merging the parent**, or merge
without deleting. Verified both ways: one stacked PR was lost, the next survived
because it was retargeted first.

**A tag captures the tree at a commit, not your working directory.** Tag after
the change is merged and `main` is pulled, never before.

**Correcting an ADR that has not merged requires amending, not a follow-up
commit.** `make adr-immutability-check` compares each ADR against the commit
that introduced it, including a commit on your own branch — so a fix in a second
commit reads as rewriting the record. Amend the introducing commit; the check is
right, and the ADR was never published.

## When you need an ADR

Any change to:

- repository structure or directory contracts
- module boundaries or the public contract surface
- the toolchain or pinned versions
- enforcement mechanisms
- the Constitution itself

Use [`docs/adr/template.md`](docs/adr/template.md). An ADR is not finished until
it has a genuine negative-consequences section, alternatives with the specific
reason each lost, and an enforcement section naming its ladder rung.

An ADR listing no costs has been advocated for, not thought about.

For decisions needing exploration first, write an RFC
([`docs/rfc/template.md`](docs/rfc/template.md)). An accepted RFC is followed by
the ADRs recording what it settled.

## The decision log is append-only

An accepted ADR is **never edited to change its meaning** (ADR-0000). To reverse
a decision:

1. Write a new ADR with `supersedes: [ADR-NNNN]`.
2. Set the old ADR's status to `Superseded`, add `superseded_by: [ADR-MMMM]`,
   leave its original text **unaltered**, and add a banner pointing forward.

`make adr-check` validates both directions of the link. ADR-0001 → ADR-0006 is
the worked example in the repository.

Typo fixes are fine. Changing what a decision *says* is not.

## The enforcement ladder

Every principle is enforced at the highest feasible mechanism (ADR-0005):

```
type system > static analysis > CI > automated review > documentation > human discipline
```

Human discipline is the last line of defence, never the first. When you have a
choice, prefer the mechanism that fails earlier. A convention a linter could
catch should become a linter.

Constitution §15 records where each principle currently sits. If your change
moves one up a rung, update that table — `make constitution-check` ensures the
table stays complete, but only you can record the climb.

## Adding an enforcement mechanism

1. Write the script in `scripts/`, with a header naming the principle it
   enforces and its ladder rung.
2. Add a `make` target; wire it into `verify`.
3. **Test it against negative cases.** Break the invariant deliberately, confirm
   the check fails with a useful message, restore.
4. Update Constitution §15.

Step 3 is required. Two of the original checks had real defects that only
negative testing surfaced — both passed the happy path.

> A check that has never gone red is unverified.

## Commit messages

Record **why**. The diff already says what.

State costs and trade-offs accepted, not only benefits. Name the ADR the change
embodies and any decision it supersedes. If you added a check, say it was
negative-tested.

Conventional prefixes: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`,
`build`, `perf`, `ci`, `revert`. Imperative mood, no trailing period. Keep the
subject under 50 characters; the `commit-msg` hook hard-fails above 72 and warns
between the two.

The hook checks the subject only. That the body records **why**, and states the
costs accepted, is not something a regexp can check — it stays a review
obligation.

## Definition of done

- `make verify` passes
- Structural changes carry an ADR
- New enforcement mechanisms have negative tests
- Constitution §15 reflects reality
- Documentation updated **in the same change**

Documentation is production code. A change leaving `docs/` stale is not
finished — `make context-check` fails on a reference to a `make` target, script,
link or ADR that does not exist.

It cannot catch a paragraph that is merely wrong. Three statements in this file
were stale when M3.5 audited it, all of them passing every check. That residue
is why review still matters.

## Releases

Automated since M3. Pushing a tag matching `libs/<name>/vX.Y.Z` (Go's
subdirectory-prefixed convention, per ADR-0004) runs `make verify` again on the
tagged commit, then builds, generates an SPDX SBOM, attests build provenance and
signs the checksum manifest with keyless `cosign`.

Go *libraries* need no artifact — a module release is a tag, served by the
proxy. The only binary released today is `fdoslint`, so that a consumer can
verify the tool gating their code was built from the source it claims.

Manual tagging is not an acceptable fallback: cross-module version chains are
too easy to get wrong by hand.

## Working with AI agents

FDOS is developed with substantial AI assistance.
[`.context/`](.context/README.md) holds structured knowledge for agents, derived
from `docs/`.

Two rules:

- **`docs/` is authoritative.** Where `.context/` disagrees, `docs/` wins and the
  disagreement is a bug in the derivation.
- **A playbook must have a subject.** Documentation for roles, tools or layers
  that do not exist is worse than none: an agent will act on it, and nothing
  reports that it was wrong.

## Open Core

The public core is Apache-2.0 (ADR-0002). Contributions are accepted under the
same licence.

Private connectors live in separate repositories and depend on this one only
through published, versioned contract modules. If a change would require a
private repository to depend on anything else, it is a design error — say so
rather than working around it.
