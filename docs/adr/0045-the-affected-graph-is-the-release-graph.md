---
id: ADR-0045
title: The affected graph is the release graph, and the registry is checked against the tags
status: Superseded
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by:
  - ADR-0046
---

# ADR-0045 — The affected graph is the release graph, and the registry is checked against the tags

> **Superseded by [ADR-0046](0046-publishing-a-tag-is-a-dispatched-act.md).**
> Rule G1 below — "every module row names its module's newest tag" — makes the
> release ritual it was written to protect unperformable: the registry update
> belongs in the commit being tagged, so during that pull request the row names
> a version no tag has yet. Everything else here is restated unchanged in the
> successor, which also decides the publication half.

## Context

Two problems that look separate and are one.

### The affected graph was built and never harvested

[ADR-0004](0004-module-granularity.md) chose `make` over Nx and accepted losing
affected-graph builds. `scripts/affected-modules.sh` was written as the
compensation, and it is called by **no automation at all** — not CI, not
`lefthook.yml`, not `scripts/doctor.sh`. ADR-0014 and
[ADR-0016](0016-developer-experience.md) both anticipate it "driving a separate
CI job"; no such job existed.

### The release chain was computed by hand, thirty-six times

Every `libs/*` is an independent module with its own tag, so a change spanning
two of them is really two or three coordinated releases in a specific order. The
order was worked out by hand each time, and got rediscovered rather than
remembered: M9 Track A, M10 and the M11 gate each found a version chain when the
gate went red ([#65](https://github.com/FabioCaffarello/fdos/issues/65)).

`CONTRIBUTING.md` has said this was unacceptable the whole time:

> Manual tagging is not an acceptable fallback: cross-module version chains are
> too easy to get wrong by hand.

**They are the same question.** "Which modules did this change affect" and
"which modules now need a tag" are one graph traversal asked twice — the second
is the first, ordered topologically and filtered to what has unreleased changes.

### The registry drifted because nothing compared it to anything

[ADR-0024](0024-contract-lifecycle-and-versioning.md) calls
`docs/ecosystem/contracts.md` part of the interface rather than documentation
about it. It went stale for four milestones, and the document now says so about
itself: *"the mechanism is rung 6 and this is what rung 6 costs."*

It was stale again when this was written, measured rather than assumed:

- `libs/kernel` listed at `v0.8.0`; newest tag `v0.9.0`.
- the governance corpus listed at `ecosystem/v0.3.0`; newest tag
  `ecosystem/v0.3.1`, which `fdos-connectors` vendors against.
- `libs/contracts v0.6.0` — tagged, pinned by two modules — with no row in the
  version history at all.

Three drifts, in a document a consumer reads to learn what exists, four days
after the last one was fixed by hand.

## Decision

### 1. `make release-plan` computes the chain the change implies

The affected set, restricted to modules whose source differs from their own
newest tag, topologically sorted over the first-party dependency graph. Output
is the ordered sequence of tags, the pin bumps between them, and the registry
rows the result will require.

**It plans; it does not publish.** Choosing a version number is a judgement about
compatibility that no script can make, so every version it prints for a new
release is a placeholder. Pushing a tag is a publication and is not this
command's business — that is a separate decision, deliberately not taken here.

**The ordering is the part worth automating**, because getting it wrong is what
costs a second release to correct.

### 2. `make registry-check` compares the registry to the tags

Four rules: every module row names its module's newest tag; every published
module is described somewhere; the corpus row names the newest `ecosystem` tag;
every published `libs/contracts` tag has a version-history row.

**The per-package `version` column is deliberately not checked.** It records the
version in which a package last changed, which is a historical fact no tag can
confirm — checking it against the newest tag would make the check demand a wrong
answer. Stating that boundary is better than checking three of four columns and
implying the fourth.

### 3. A preflight job runs the affected modules, and is not a gate

ADR-0014 rejected pruning `verify` by affectedness and named where the speed
belongs: *"Speed belongs in a separate job."* This is that job.

**It runs a strict subset of what `verify` runs, over a subset of the modules it
covers**, so it cannot go red while the gate is green. That property is the
design, not an accident: a non-required check that can fail on its own is one
people learn to ignore, and this repository already disabled `dependency-review`
for exactly that reason. `verify` does not depend on it either — making the gate
wait on an advisory job would make the advisory job a gate.

`verify` remains the only required status check ([ADR-0020](0020-open-core-boundary-and-pull-request-workflow.md)).

### 4. The affected graph is surfaced where people look

`scripts/doctor.sh` prints it, and the pull-request template asks for the release
chain. A mechanism nobody invokes is a mechanism that does not exist, which is
what this ADR is mostly about.

### 5. What this does not decide

**Nothing about publishing.** No workflow gains permission to create a tag, and
no version number is chosen by a machine.

**No automatic rewriting of the registry.** `release-plan` prints the rows and
`registry-check` prints the corrected row on failure; neither edits the file. A
script that rewrites a document containing prose will eventually eat a
paragraph, and this one contains the reasoning for every version it lists.

**No opinion on whether `apps/*` and `examples/*` should be tagged.** They are
placed at the end of the chain as consumers that publish nothing, which is what
they are today.

## Consequences

### Positive

- The release chain is computed once instead of rediscovered per milestone.
- The registry cannot silently disagree with the tags. Three live drifts were
  found by turning it on, one of them a version pinned by two modules and
  described nowhere.
- `CONTRIBUTING.md`'s standing claim about manual tagging is closer to true: the
  mechanical half is a command, and the judgement half is still a person's.
- The affected graph earns its keep for the first time since ADR-0004.

### Negative

- **`release-plan` is advisory and nothing enforces that it was run.** The
  pull-request template asks; that is rung 5. The enforcement lives in
  `pin-check` and `registry-check`, which catch the *consequences* of skipping
  it rather than the skipping.
- **The preflight job costs a second runner on every pull request**, for
  information the gate produces anyway a minute later. The trade is fast failure
  on the common case; it is a real recurring cost for a convenience.
- **`registry-check` can only decide what is mechanical.** The prose beneath
  each table — which is where the actual meaning of a release lives — stays
  unchecked, and a wrong paragraph passes.
- **Three registry rows were repaired in the same change that added the check.**
  As in ADR-0044, that makes the diff larger than the mechanism, and it is the
  evidence the mechanism was overdue.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| The registry names the newest tag of every published module | 3 | `make registry-check` G1/G2, negative-tested |
| The corpus row names the newest `ecosystem` tag | 3 | `make registry-check` G3, negative-tested |
| Every published contracts version is described | 3 | `make registry-check` G4, negative-tested |
| The release chain is known before it is performed | 5 | `make release-plan`, asked for by the pull-request template |
| Affected modules fail fast | 4 | the `preflight` job — advisory, and cannot fail alone |

## Alternatives considered

**Make `release-plan` also push the tags.** The obvious next step, and it is
where [#104](https://github.com/FabioCaffarello/fdos/issues/104) goes. Rejected
here on sequencing: publication needs its
own decision about permissions and about who presses the button, and bundling it
into a planning command would make the plan a publication by accident.

**Have `registry-check` rewrite the table instead of failing.** Removes the
manual step entirely. Rejected: the document is mostly prose explaining what each
version means, generated content and prose in one file drift into each other, and
the failure message already contains the exact row to paste.

**Check the per-package `version` column too.** More coverage. Rejected because
the column does not mean what a mechanical check would assume, and a check that
demands a wrong answer is worse than no check on that column.

**Make the preflight job required.** It would fail faster *and* block. Rejected:
that is a narrower gate by another name, which ADR-0014 decided against and the
gate's measured cost — a 114s median — gives no reason to revisit.

**Derive the affected set from `go list` alone rather than a git diff.** More
precise. Rejected as unnecessary: the script is deliberately conservative — a
change to anything shared marks every module affected — and over-reporting costs
time while under-reporting ships a break. That trade is already recorded in the
script and is right for an advisory tool.

## Notes

- `libs/analysis` has never been tagged, so `release-plan` lists it as "never
  released" whenever it is affected. That is accurate and slightly odd: it is the
  module `fdoslint` is built from, and `release.yml` triggers on `libs/*/v*`.
  Recorded in ADR-0044's notes as well, still not acted on.
- The plan places `apps/*` and `examples/*` after every library, as consumers
  that publish nothing. If ADR-0039 is accepted and applications start carrying
  tags, that placement needs revisiting rather than extending.
