# Contributing to FDOS

FDOS is built to be maintained for a decade. Almost every rule here exists
because something valuable is lost by default over that horizon — usually the
reasoning behind a decision, occasionally the ability to reproduce a number.

Read [`docs/constitution.md`](docs/constitution.md) first. It is the highest
authority in this repository, and it is short.

## Getting started

```sh
make bootstrap   # validate the toolchain, install git hooks
make verify      # run every enforcement mechanism available at this milestone
make help        # list targets
```

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

**FDOS currently has no Go code, deliberately.** The canonical financial model is
an output of the M1.5 RFCs. A contribution that adds domain types, `go.mod`
files or business rules before those RFCs land will be declined — not because it
is bad work, but because it settles open questions by accident.

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
finished, and from M2 a directory README that misdescribes its module will fail
the build outright.

## Releases

Not yet automated. There are no modules to publish and no artifacts to sign.

When releases begin, they use Go's subdirectory-prefixed tags
(`libs/<name>/vX.Y.Z`, per ADR-0004) and are driven by automation landing in
**M3**, together with SBOM generation, SLSA provenance attestation and artifact
signing. Manual tagging is not an acceptable fallback — cross-module version
chains are too easy to get wrong by hand.

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
