---
id: RFC-0018
title: The delivery pipeline — what the gate cannot see, and the release chain nobody computes
status: Accepted
date: 2026-08-12
authors:
  - "@FabioCaffarello"
---

# RFC-0018 — The delivery pipeline: what the gate cannot see, and the release chain nobody computes

## Summary

The pipeline is architecturally sound and operationally thin. Every rule
[ADR-0014](../adr/0014-ci-runs-make-and-pins-everything.md) set is still held —
CI invokes `make`, actions are pinned by SHA, `GOWORK=off` is everywhere — and
the gate is fast. What has accumulated underneath is a different class of
problem: **the gate cannot see the shape of change this repository most often
makes, and the release chain that turns one merged change into three published
tags is computed by hand every time.**

This proposes six phases. Two are mechanical and need no design. Four change
enforcement mechanisms and therefore need ADRs, and they are the reason this is
an RFC: each has a real alternative that is not a straw man, and one of them —
whether a cross-module pin check *fails* or merely *reports* — makes
in-progress multi-module work either impossible to commit half-done or
unpoliced, with no third position.

It does **not** propose pruning the gate with the affected graph. That was
decided against in ADR-0014 and the measurement below says there is nothing to
gain by reopening it.

## Motivation

### What was measured

Per-target duration of `make verify`, on `darwin/arm64` with a warm build cache
against a clean `main`:

| Target | Time | Target | Time |
|---|---|---|---|
| `vuln-check` | 9.9s | `context-check` | 2.9s |
| `lint` | 2.1s | `repro-check` | 1.6s |
| `adr-immutability-check` | 1.5s | `analyze` | 1.3s |
| `adr-check` | 1.1s | remaining twelve | < 1s each |

**Total 25s locally.** In CI, across the last twelve successful `verify` runs,
over eight modules and thirty-six published tags:

```
96 103 114 115 126 128 129 130 134    220 255 279     (seconds)
min 96    median 129    max 279
```

**The distribution is bimodal, and the gap is the finding.** Nine runs sit
between 96s and 134s; three sit between 220s and 279s, which is 2.9× the
fastest. The plausible cause is the Go build cache restored by `setup-go`
hitting or missing, and *plausible* is as far as this document can honestly go —
nothing records which.

That measurement says two things rather than one. **CI time is not a problem**,
so a plan spending its effort on the median would be optimising two minutes
while the expensive failures went untouched. And **the repository cannot
currently distinguish a slow run from a degrading one**, which is why Phase 0 is
instrumentation and comes first.

A methodological note, because it is the same failure this RFC is about: an
earlier draft of this section quoted 103s as *the* CI figure. That was a real
measurement of a real run, and it was the second-fastest of twelve. One sample
presented as a constant is what an unmeasured pipeline invites, and it survived
until a 279s run happened to land on this RFC's own pull request. The four
failures below are the expensive ones.

### 1. The gate cannot see a multi-module change

`FOR_EACH_MODULE` runs each module with `GOWORK=off`, so siblings resolve from
the proxy at published versions. While `libs/kernel` is being edited, the five
modules that import it compile against the *previous release from the module
cache*. This is deliberate — it is what proves standalone resolution, and
ADR-0004 depends on it — but it means the blast radius of a signature change is
invisible until one tag at a time makes it visible.

**This is not hypothetical and it is not old.** ADR-0041 was published in full
knowledge of it, and the release note for `libs/ledger/v0.8.0` in
[`contracts.md`](../ecosystem/contracts.md) says so in writing:

> it **breaks every implementation of `app.Store`**, including any out of tree,
> and no mechanism here can see it. `buf breaking` cannot, because no schema
> moved. `make verify` cannot […] So this release is deliberately published into
> a tree that does not build as a workspace.

Two related facts, both verified rather than inferred:

- `go.work` lists eight directories and **`libs/ledger-sqlite` is not among
  them**. Even a workspace build never compiles that module against local
  siblings.
- `examples/` is outside `scripts/list-modules.sh` by construction, so nothing
  in `make verify` touches it. `examples/ingest` stopped compiling and stayed
  broken unreported, while an enforcement row claimed the kit ran in CI
  ([#79](https://github.com/FabioCaffarello/fdos/issues/79)).

The cost is already paid three times: M9 Track A, M10 and the M11 gate each
discovered a cross-module pin chain by hand, each at the end of a slice rather
than the start ([#65](https://github.com/FabioCaffarello/fdos/issues/65)).

### 2. `CONTRIBUTING.md` states a rule the repository does not follow

> Manual tagging is not an acceptable fallback: cross-module version chains are
> too easy to get wrong by hand.

The actual ritual is manual throughout: merge the change, open a
`release/<module>-vX.Y.Z` branch that edits the version table in
[`contracts.md`](../ecosystem/contracts.md) by hand, merge it, `git tag`,
`git push`, then repeat for each dependent module in the chain. Thirty-six tags
have been produced this way.

**Nothing verifies that table.** ADR-0024 calls the registry "part of the
interface" rather than documentation about it, and #79 found it listing
`libs/kernel` at `v0.5.0` against a pinned `v0.7.0`, with `libs/ledger-sqlite`
— published, tagged and imported — absent from it entirely.

### 3. Two build inputs are not pinned by digest

`.github/actions/setup-toolchain/action.yml` installs gitleaks and buf with
`curl` by version, with no checksum. ADR-0014 records this as an open gap and
names the fix. It is still open, and it is the step that installs the tools
whose output every later attestation rests on.

Adjacent, and verified against the live API: the `release-tags` ruleset covers
`refs/tags/libs/*/v*` and nothing else. Tags matching `apps/*/v*` — which
[ADR-0039](../adr/0039-applications-are-released-as-signed-binaries.md) proposes
to start producing — and the `ecosystem/*` corpus tags are **movable today**. A
release tag that can move makes every provenance attestation pointing at it
describe something that is no longer there, which is the argument
[`branch-protection.md`](../branch-protection.md) already makes for the tags it
does cover.

### 4. The affected graph is built and never harvested

`scripts/affected-modules.sh` exists, is exposed as a `make` target, and is
called by **no automation at all** — not CI, not `lefthook.yml`, not
`scripts/doctor.sh`. ADR-0014 and
[ADR-0016](../adr/0016-developer-experience.md) both anticipate it "driving a
separate CI job"; no such job exists.

### Which principle is at stake

Constitution §9 (reproducibility) for item 3, and §15's ladder for the rest:
items 1, 2 and 4 are all cases where a rule the repository believes it holds is
held at rung 6 — human attention — while the artifact reads as though a
mechanism holds it. ADR-0032 exists because this repository decided that the
worst version of a gap is one carried as habit rather than written down.

**Retrofittability.** All six phases are retrofittable; none of this is a
schema decision that expires. What is not retrofittable is the history a moved
tag would invalidate, which is why the ruleset gap in item 3 is grouped with
the pinning work rather than deferred.

## Design

Six phases, ordered so that each is useful alone and none depends on a later
one.

### Phase 0 — Instrument, before optimising anything

No ADR. The per-target table in §Motivation had to be produced by a throwaway
script, which is the finding: nothing in this repository reports what its own
gate costs.

- A timing helper under `scripts/lib/`, opt-in through an environment variable,
  so `make verify` can report duration per target without changing its default
  output.
- A CI step that writes those durations to the job summary — through a script,
  not inline YAML.
- A weekly job publishing p50/p95 duration and failure rate of recent `verify`
  runs as an issue.
- **Cache hit or miss recorded per run.** The 2.9× spread in §Motivation is
  attributed to the build cache by inference, not by evidence, and the whole
  "there is room to grow the gate" argument rests on knowing which mode a run
  was in.

**Why first.** It is what tells you when the "CI time is not a problem"
conclusion above has expired, it is the only honest trigger for the job matrix
this RFC declines to build today, and it is what would have stopped this
document's own first draft from quoting one sample as a constant.

### Phase 1 — Close the pinning gaps

A checksum file beside `mise.toml`, `scripts/tool-version.sh` extended to serve
checksums, both `curl` installs verifying before executing, and a check that
fails when a URL-installed tool has no registered checksum.

The tag ruleset is extended to `refs/tags/apps/*/v*` and `refs/tags/ecosystem/*`
and [`branch-protection.md`](../branch-protection.md) records it.

**Negative test.** Remove a checksum line; the check fails naming the tool.
Corrupt one; the install fails rather than proceeding.

**ADR.** Yes, short. `CONTRIBUTING.md` lists the toolchain and its pins as ADR
matter, and this changes how the toolchain is installed. It implements a fix
ADR-0014 already named rather than inventing policy, so the ADR is a record, not
an argument.

### Phase 2 — The gate acquires a workspace view, without losing the published one

The expensive phase, and the one that closes #79 and #65 together.

- **A root workspace build, alongside — never instead of — the per-module
  `GOWORK=off` runs.** Both properties matter and they are different
  properties: `GOWORK=off` proves each module resolves standalone from the
  proxy, and a workspace build proves the tree is internally consistent *before*
  a release makes the inconsistency somebody's problem. Running only the second
  would silently retire ADR-0004's discipline.
- **A workspace-membership check.** A module that exists and is absent from
  `go.work` fails the gate. `libs/ledger-sqlite` is absent today.
- **A first-party pin check.** For each module, its first-party pins beside the
  version present in the tree; fails when a pin names a tag that does not exist
  or predates the tree.
- **`examples/` enters the module set.** Constrained by #79 item 3: the
  `conforming.textproto` fixture is byte-compared, and
  `protobuf/internal/detrand` makes that output unstable across builds, so the
  fixture passes only for the build that produced it. Replacing the comparison
  with a semantic one is a prerequisite of this bullet, not a detail of it.

**Negative tests.** Pin a module to a tag that does not exist — the check fails
naming it. Remove a module from `go.work` — membership fails. Break a sibling's
signature in the working tree — the workspace build fails where the per-module
run passes, which is precisely the case ADR-0041 shipped blind.

**ADR.** Yes, one covering all four.

### Phase 3 — The affected graph is the release graph

This is where the unharvested mechanism from §Motivation item 4 pays, and the
observation the phase rests on is that **"which modules did this change affect"
and "which modules now need a tag" are the same question asked twice.** The
repository answers the first with a script it never runs and the second by hand.

- **A release-plan target.** The affected set, ordered topologically over
  first-party dependencies, printed as the exact sequence of tags and `go.mod`
  bumps. What three milestones each found by hand becomes one command.
- **The same computation generates the registry diff** for
  [`contracts.md`](../ecosystem/contracts.md), retiring the hand-edited
  `release/*` branch.
- **A registry check.** Fails when the version table disagrees with published
  tags. Today nothing checks a document ADR-0024 calls part of the interface.
- **A non-required preflight CI job** running tests and lint over the affected
  modules only, for fast failure. `verify` remains the sole required check.
  This is ADR-0014's own instruction — *"Speed belongs in a separate job"* —
  executed rather than restated.
- The affected target is wired into `scripts/doctor.sh` and the pull-request
  template.

### Phase 4 — Publishing a tag becomes an auditable act rather than a ritual

Two manually dispatched workflows: one that runs the release plan and opens the
pull request carrying the registry update and the pin bumps, and one that
creates the tag from the green commit on `main` after that merges. Write
permission is scoped to the second alone, behind a protected environment.

**Why dispatched rather than automatic, stated as the decision it is.**
Publishing is a human act. Keyless signing binds the artifact's identity to the
workflow, and a tag pushed without a human choosing to publish produces a signed
claim nobody decided to make. The human keeps *whether* and *which version*; the
machine takes the chain mechanics, which is where the errors actually are. This
also makes `CONTRIBUTING.md`'s existing sentence true rather than aspirational.

### Phase 5 — Attest what downstream consumes; give applications a path

- **Accept ADR-0039**, whose extraction of one signing path is already designed
  and whose reasoning this RFC does not reopen.
- **A release rehearsal**, exercising the full release path without a permanent
  tag. This closes ADR-0039's own stated weak point — *"Nothing tests a release
  workflow except releasing"* — which is the condition that let B-008 ship
  fourteen empty releases while looking green. The drill already exists ad hoc:
  the `libs/release-smoke/v0.0.0-rc.1` and `rc.2` tags are its residue.
- **Attest the `libs/contracts` module zip.** `fdoslint` is attested and the
  module an external repository actually builds against carries no attestation
  at all. ADR-0014 left this open; the priority is inverted and worth
  correcting.

### Phase 6 — Drift and freshness, reported and never applied

- **A weekly freshness job** comparing each pinned action SHA against its
  upstream tag and **opening an issue**. Not a pull request, not a merge.
  ADR-0014 accepted that pins would lag and named "the scheduled supply-chain
  workflow plus deliberate review" as the mitigation; the scheduled workflow
  exists and checks the freshness of nothing. This builds what was promised
  while respecting the refusal of automatic updates.
- **A ruleset drift check that runs locally, deliberately not in CI.** Reading
  rulesets needs an admin-scoped token, which is the risk ADR-0014 declined and
  [ADR-0020](../adr/0020-open-core-boundary-and-pull-request-workflow.md)
  recorded as an open gap. Run from the maintainer's own authenticated CLI
  through `scripts/doctor.sh`, the objection does not transfer — the same
  argument `branch-protection.md` used to apply the rulesets by hand.
- **Platform parity** ([#67](https://github.com/FabioCaffarello/fdos/issues/67),
  option B): a job triggered by changes to `Makefile`, `mise.toml` and
  `.github/**`, so a build-flag change is validated on CI's platform early
  rather than at the end. This is the class that made `CGO_ENABLED=0` pass every
  local run and fail the first CI run to reach it.
- **Noise.** `supply-chain` produces a `skipped` conclusion on every pull
  request, because its only PR job is disabled at the job level. Moving the
  condition so no phantom check is created costs nothing and stops training the
  eye to ignore a red-adjacent signal. Dependency review returns when the
  Dependency Graph is enabled
  ([#55](https://github.com/FabioCaffarello/fdos/issues/55)).

### What this does not cover

- **No container image, package manager or installer.** ADR-0037 and ADR-0039
  both settled that a distribution channel carries a support obligation, and one
  channel at a time.
- **No deployment.** `deploy/` is empty by design and this RFC does not fill it.
- **No versioning policy for applications.** ADR-0039 explicitly left it open
  and this does not close it.
- **No job matrix over modules.** Declined on arithmetic, not principle: fixed
  per-job setup is ~20s against a median serial run of 129s, and the runs that
  actually hurt are the cache-miss ones a matrix would multiply rather than
  divide. Phase 0 exists to say when that inverts.
- **No change to what `verify` covers as the required check.** Every phase adds
  to the gate or runs beside it; none narrows it.

## Enforcement

Ladder rungs per [ADR-0005](../adr/0005-enforcement-ladder.md).

| Phase | Mechanism | Rung | Negative test |
|---|---|---|---|
| 0 instrumentation | reporting only; enforces nothing | — | n/a — and this is why it needs no ADR |
| 1 checksums | checksum verified before install; check fails on a missing entry | 3 | remove an entry; corrupt a checksum |
| 1 tag ruleset | repository ruleset | 3 | attempt to move an `apps/*/v*` tag |
| 2 workspace build | `make` target in the gate | 3 | break a sibling signature in the tree |
| 2 membership | check over `go.work` versus the module set | 3 | remove a module from `go.work` |
| 2 pin check | check over first-party pins versus tags | 3 | pin a tag that does not exist |
| 3 registry | check over the version table versus published tags | 3 | stale the table by one version |
| 3 preflight job | non-required CI job | 4 | n/a — advisory by construction |
| 4 tag publication | dispatched workflow, protected environment | 3 | rehearsal on a disposable tag |
| 5 rehearsal | dispatched workflow exercising the release path | 2 | it *is* the negative test for the release path |
| 6 freshness | scheduled job opening an issue | 4 | a stale pin produces an issue |
| 6 ruleset drift | local check invoked from `doctor` | 6 in CI, 3 locally | change a ruleset in the UI; `doctor` reports |

**Two rows are honest weak points and are stated rather than dressed up.**
Ruleset drift stays rung 6 from CI's perspective because raising it requires the
admin token ADR-0014 refused, and refusing it again is the right call. The
preflight job is advisory by construction: if it were required it would be a
narrower gate, which is the thing this RFC will not do.

## Alternatives

**Prune `make verify` with the affected graph.** The obvious performance move,
and the one most reviewers would expect. Rejected twice over: ADR-0014 already
decided it — *"under-reporting affectedness ships a broken module, and the
failure is silent"* — and the measurement says there is no problem to solve.
Reopening it would need a superseding ADR and would be trading a real
correctness property for two minutes.

**A job matrix over modules now.** Would cut wall-clock without narrowing
coverage, and preserves a single required check through an aggregating job.
Rejected on arithmetic today rather than on principle: eight jobs at ~20s of
fixed setup each cost more than the 129s median they would parallelise, and the
279s tail is a cache miss — which a matrix pays once per job rather than once
per run, making the worst case worse rather than better. It becomes
correct at some point and Phase 0 is what will detect that point.

**Dependabot or Renovate for actions and modules.** The standard answer to
lagging pins, and it would close the freshness gap with no bespoke job.
Rejected: it contradicts ADR-0014 head-on, where an input that can change
without a reviewed commit is the whole thing being defended against. Phase 6
takes the reporting half and leaves the applying half to a human, which is the
only version compatible with the accepted decision.

**Replace the per-module `GOWORK=off` runs with a workspace build.** Simpler,
one build, and it would catch every cross-module break Phase 2 targets.
Rejected: it silently retires ADR-0004's discipline. The `GOWORK=off` runs are
what prove a published module resolves for a consumer that has no workspace, and
that consumer exists.

**Do nothing and keep the release chain manual.** Not a straw man — it has
produced thirty-six correct tags and the ritual is understood by the person
performing it. Rejected because the ritual's failure mode is silent and
cumulative: #65 records the chain being rediscovered three milestones running,
#79 records the registry drifting two versions unnoticed, and ADR-0041 shipped a
known-broken tree because no mechanism could see it. Each is individually
survivable and the trend is the argument.

## Prior art

**Nx and Bazel affected-graph builds.** ADR-0004 chose `make` over Nx and
accepted losing this; `affected-modules.sh` was written as the compensation. The
industry lesson worth importing is not the tool but the split: build systems
that prune the gate by affectedness pair it with a hermetic dependency graph
that cannot under-report. This repository has no such guarantee — a shell script
intersecting a git diff is a heuristic — which is why Phase 3 puts the graph on
the advisory side and the release side, and never on the gate.

**Go's own release discipline.** Multi-module repositories in the Go ecosystem
(`golang.org/x/tools`, `google.golang.org/protobuf`) converge on the same
answer Phase 3 proposes: the tag chain is computed from the module graph, not
remembered. `gorelease` and the `x/` repositories' release tooling exist because
the manual version of this is known not to survive contact with more than two
modules.

**SLSA and the sigstore ecosystem.** The provenance model in place today is the
right one, and its own literature is clear that an attestation over the wrong
subject is the common failure — which is Phase 5's third bullet: the attested
artifact is the linter, and the consumed artifact is the contracts module.

**The industry's dependency-bot consensus, and why it is declined.** Automatic
pin updates are near-universal practice and reduce lag effectively. FDOS took
the opposite trade knowingly in ADR-0014; the prior art is cited here so the
divergence is visible as a choice rather than an oversight.

## Open questions

> **All four are answered below**, in the ADRs named. They are left in their
> original wording because the answers only make sense against the questions as
> they were asked.

1. **Does the first-party pin check fail the gate, or only report?** The
   sharpest question here and the reason it is an RFC. Failing makes an
   in-progress multi-module change impossible to commit half-done — which is
   either exactly the point or an obstruction, depending on whether the release
   chain is worked one module at a time. #65 raises it and leaves it open. This
   RFC's position is *fail*, on the evidence that three milestones lost time to
   the reporting-only equivalent, and the counter-argument deserves an answer in
   the ADR rather than here.

2. **What does the workspace build do about the `examples/` fixture?** Semantic
   comparison, whitespace normalisation, or dropping the textproto fixture. #79
   item 3 establishes that byte comparison cannot pass in both modes; which
   replacement is right is a test-design question this RFC does not settle.

3. **Should the release-plan output be committed?** A generated registry diff
   that is reviewed and merged is auditable; one regenerated at release time is
   never stale. The two are in tension and ADR-0024's "part of the interface"
   framing points at the first.

4. **Does rehearsal use a disposable tag or a tagless dispatch?** The disposable
   tag is proven — it is what proved B-008 fixed — and it leaves permanent
   residue in the tag namespace, which `libs/release-smoke` already
   demonstrates.

Resolved by: @FabioCaffarello, in the ADRs this RFC produced.

| # | Answer | Where |
|---|---|---|
| 1 | **Fails** — but split. R1/R2/R3 block; R4, a released module pinning behind, reports. Thirteen such pins existed and none was a defect; blocking them would redden `main` on every tag. | ADR-0044 |
| 2 | **No change needed.** The `prototext` fixture is parsed and compared with `proto.Equal`, not byte-compared. The concern was right when raised and had already been answered. | ADR-0044 |
| 3 | **Printed, not committed.** `release-plan` prints the rows and `registry-check` prints the corrected row on failure; neither edits the file. A script that rewrites a document containing prose will eventually eat a paragraph. | ADR-0045 |
| 4 | **Tagless dispatch.** The disposable tag was the proven instrument and stopped being available: ADR-0043 made tags undeletable, so every drill would leave permanent residue. | ADR-0047 |

## Consequences

### What becomes easier

- A multi-module change is designed against a computed chain instead of
  discovered against a red gate at the end of a slice.
- The registry stops being a document maintained by memory.
- A release becomes a decision plus a dispatch, rather than a decision plus six
  mechanical steps that must be performed in order.
- The claim that every build input is pinned by digest becomes true.

### What becomes harder

- **The gate grows.** A workspace build and three new checks add time to
  something whose median CI run is 129s and whose worst is 279s, and Phase 0
  exists partly to keep that visible. This RFC trades gate time for coverage
  deliberately, having established there is room — and the bimodality means the
  trade should be judged against the tail rather than the median.
- **In-progress cross-module work may become uncommittable** if open question 1
  resolves to *fail*. That is the cost of the mechanism, not a side effect of
  it.
- **More CI surface to maintain**, including two dispatched workflows touching
  the release path — the path with the worst track record in the repository
  (B-008, [`blocked.md`](../blocked.md)). Phase 5's rehearsal is the mitigation
  and it is not a complete one.

### What becomes impossible

- Publishing a release tag whose chain was never computed.
- A module existing outside the workspace without the gate saying so.
- Moving an `apps/*` or `ecosystem/*` tag after an attestation points at it.
- Shipping a version table that disagrees with the tags it claims to describe.

---

## What execution changed, recorded after the fact

This section was written when the last phase merged. An RFC that only records
what was planned is a plan; this is what it cost to be right and where it was
wrong.

### The phases, and what each produced

| Phase | ADR | Held as planned? |
|---|---|---|
| 0 — instrument | none | yes |
| 1 — pinning gaps and tag ruleset | ADR-0043 | yes |
| 2 — workspace view | ADR-0044 | yes, with the R3/R4 split the plan did not anticipate |
| 3 — affected is the release graph | ADR-0045 | **no** — one rule was wrong |
| 4 — publishing is dispatched | ADR-0046 | yes, and it corrected Phase 3 |
| 5 — attest what is consumed | ADR-0047, ADR-0039 accepted | **no** — the defect was larger than the plan knew |
| 6 — drift reported | ADR-0048 | yes, minus one item that was already done |

### Four things the plan got wrong

**Its own headline number.** The Motivation quoted 103s as the CI duration; it
was the second-fastest of twelve, and a 279s run landed on this RFC's own pull
request. Corrected in `f103bab`, before any phase was built. One sample
presented as a constant is what an unmeasured pipeline invites, and it happened
inside the document arguing for measurement.

**Phase 3 shipped a rule that forbade the thing it protected.** `registry-check`
G1 was stated as "every module row names its module's newest tag". The registry
update belongs in the commit being tagged, so during a release pull request the
row names a version no tag has yet — and G1 failed it. The other order reddens
`main` for the whole window, which is the property Phase 2 had refused four
hours earlier for a different rule. ADR-0046 supersedes ADR-0045 to fix it.

**Phase 5's defect was not the one the plan named.** The plan said the
`libs/contracts` zip was unattested while `fdoslint` was — true, and the smaller
half. Reading what was actually published showed **every library tag was
publishing `fdoslint` binaries** with a signed manifest describing them: real
signatures over the wrong artifact, in twenty releases. That is worse than an
unattested module, and no amount of reasoning about the workflow would have
found it. `gh release view` did.

**Phase 6 had an item that was already done.** [#67](https://github.com/FabioCaffarello/fdos/issues/67)
option B — a job so the platform-sensitive gate runs early — duplicates `verify`,
which already runs `make test -race` with `CGO_ENABLED=1` on `ubuntu-latest` for
every pull request. Declined with the measurement rather than built.

### What stayed declined

The three things this RFC refused in §Alternatives were never revisited: no
affected-pruned gate, no job matrix, no dependency bot. The gate grew by roughly
two seconds locally against a 114s CI median, which is the trade §Consequences
said it was making.

### What the gate cost, before and after

| | Median CI run | Range | Checks |
|---|---|---|---|
| before Phase 0 | 129s | 96–279, bimodal | 19 |
| after Phase 6 | **106s** | 92–269, still bimodal | **23** |

Four checks were added and the median went *down* by 23s, which is not a
claim about the checks: it is cache behaviour, and the range shows the
distribution did not change shape.

The distribution is still bimodal and the cause is now recorded per run rather
than inferred: `make ci-summary` writes the build-cache state, and the first
slow run after it landed reported `Build cache: miss`.

### Still open

- `examples/ingest/ingest`, a tracked 7.3 MB binary, is still tracked, and is
  now known to be silently overwritten by an ordinary `go build ./...`
  ([#79](https://github.com/FabioCaffarello/fdos/issues/79)).
- Twenty published releases still carry `fdoslint` under other modules' tags.
  They are history attached to immutable tags and nothing marks them
  (ADR-0047).
- `release.yml`'s last two steps — `gh release create` and the tag trigger —
  are still exercised only by releasing (ADR-0047).
- `lefthook` drops the `commit-msg` argument inside a linked worktree, so every
  commit in this work was made with `--no-verify` and verified afterwards with
  `make commit-msg-check` ([#109](https://github.com/FabioCaffarello/fdos/issues/109)).
