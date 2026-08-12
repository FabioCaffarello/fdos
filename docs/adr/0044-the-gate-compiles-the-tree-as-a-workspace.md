---
id: ADR-0044
title: The gate compiles the tree as a workspace, and a module under change pins what its siblings released
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0044 — The gate compiles the tree as a workspace, and a module under change pins what its siblings released

> Phase 2 of [RFC-0018](../rfc/0018-the-delivery-pipeline.md), and its open question 1 — whether the pin check fails or reports — resolved as *fails*.

## Context

[ADR-0004](0004-module-granularity.md) makes each `libs/*` an independent module
with its own tag, and `FOR_EACH_MODULE` runs every Go command with `GOWORK=off`
so siblings resolve from the proxy at *published* versions. That is deliberate
and load-bearing: it is what proves a module resolves standalone for a consumer
with no workspace, which is a consumer that exists.

It also means the gate cannot see the shape of change this repository most often
makes. While `libs/kernel` is edited, the five modules importing it compile
against the previous release from the module cache. A signature change is
compile-checked only inside its own module, and its blast radius surfaces one
tag at a time.

### This is not a prediction. It had already happened, twice.

[ADR-0041](0041-the-write-path-serialises-in-the-store.md) added `Serialise` to
`app.Store` and its release note says so plainly:

> it **breaks every implementation of `app.Store`**, including any out of tree,
> and no mechanism here can see it. `buf breaking` cannot, because no schema
> moved. `make verify` cannot […] So this release is deliberately published into
> a tree that does not build as a workspace.

**Measured while building this check**, on a clean `main` with the gate green:

```
$ cd apps/submitd && GOWORK=<root>/go.work go vet ./...
# github.com/FabioCaffarello/fdos/libs/ledger-sqlite
…/libs/ledger-sqlite@v0.2.0/store.go:453:19: cannot use (*Store)(nil) as
app.Store value: *Store does not implement app.Store (missing method Serialise)
```

And after bringing `apps/submitd` to `libs/ledger v0.8.0`, a second one in its
own test file:

```
vet: ./server_test.go:186:3: cannot use failingStore{…} as app.Store value in
argument to app.NewLedger: failingStore does not implement app.Store
```

Both were invisible to `make verify` because `apps/submitd` pinned
`libs/ledger v0.7.0`. The repository could not compile against its own current
release, and the gate was green.

`libs/ledger-sqlite` was additionally absent from `go.work` entirely, so even a
workspace build never saw its source against local siblings.

### `examples/` was outside every target

`scripts/list-modules.sh` searched `libs` and `apps`. Nothing in `make verify`
touched `examples/`, and
[ADR-0037](0037-delivery-includes-a-service-the-adopter-operates.md) carried an
enforcement row claiming *"`examples/ingest` — the kit runs in CI"*. Bringing it
into the module set found, immediately: source that is not `gofmt`-clean, a
`go.mod` that is not tidy, and a `revive` violation. None is serious. All of them
had been true for some time, in the directory whose job is to show a third party
what conformance looks like.

### Three milestones paid for the pin problem by hand

M9 Track A, M10 and the M11 gate each discovered a cross-module version chain
when the gate went red rather than when the change was designed
([#65](https://github.com/FabioCaffarello/fdos/issues/65)).

## Decision

### 1. The gate compiles the tree as a workspace, alongside the per-module runs

`make workspace-check` compiles every module against its siblings' source.
**It does not replace the `GOWORK=off` runs.** Both properties matter and they
are different properties: one proves a module resolves standalone from the
proxy, the other proves the tree is internally consistent before a release makes
the inconsistency somebody else's problem. Running only the second would
silently retire ADR-0004's discipline.

### 2. It runs `go vet`, not `go build`

Test files are where interface assertions and doubles live, and `go build` does
not compile them. The `failingStore` defect above is invisible to `go build` and
caught by `go vet`. Choosing the cheaper command would have produced a check
that passed over the exact defect it was written for.

### 3. It sets `GOWORK` to an explicit path, and proves the workspace is live

`verify.yml` exports `GOWORK: "off"` for the whole workflow and `mise.toml` sets
it for developers. A check that inherited the environment would pass while
testing nothing, in CI, permanently.

So the path is explicit, and the check then **resolves a first-party dependency
and requires the answer to be inside the repository** before trusting any later
result. Negative-tested: with the explicit path removed and `GOWORK=off` in the
environment, the guard reports the module-cache path and fails rather than
reporting a vacuous pass.

### 4. `make pin-check` fails on three rules and reports a fourth

| | Rule | Disposition |
|---|---|---|
| R1 | a first-party pin names a tag that exists | **fails** |
| R2 | a pin is not newer than the newest tag | **fails** |
| R3 | a module with unreleased changes pins its siblings at their newest tag | **fails** |
| R4 | a released, unchanged module pinning behind | reports |

**R4 does not fail, and that is measured rather than cautious.** Thirteen pins
across five modules were behind their dependency's newest tag, and none was a
defect: a module legitimately stays on what it was released against until
somebody bumps it. Making that blocking would turn `main` red the instant any
module is tagged — before the follow-up bump could merge — and would make the
release sequence ADR-0041 documents impossible to perform.

**R3 does fail**, and it is the answer to
[#65](https://github.com/FabioCaffarello/fdos/issues/65)'s open question. It
fires only on modules with changes not yet released: the ones somebody is
working on. Tagging a dependency does not redden the gate; *editing* a module
whose pins are stale does. That is exactly the case that cost three milestones,
and it is caught when the change is designed rather than when the gate fails.

Indirect requirements are exempt from R2, R3 and R4: their versions are chosen by
minimal version selection across the whole graph rather than by whoever wrote the
`go.mod`. R1 still applies — an indirect pin naming a tag that does not exist is
a build resolved off nothing.

### 5. `examples/` is a module like any other

`scripts/list-modules.sh` searches `examples` as well. The kit is now formatted,
tidy, lint-clean, vetted, tested, analysed, scanned and built reproducibly,
because it claims to be the thing a third party copies.

### 6. What this does not decide

**Nothing about `go.work`'s role for editors.** It remains what ADR-0004 said it
is; this adds a second reader of it.

**No opinion on whether `main` should ever carry an unreleased cross-module
change.** R3 makes the pins explicit at the moment of change; it does not require
a tag to exist before the change lands.

**The tracked binary `examples/ingest/ingest` is not removed here.**
[#79](https://github.com/FabioCaffarello/fdos/issues/79) flagged it for a human
and that has not changed — though it is now known to be overwritten silently by
an ordinary `go build ./...` in that directory, which is one more reason it
should not be tracked.

## Consequences

### Positive

- A cross-module API break fails the gate on the commit that causes it, instead
  of on a later tag in a different module.
- The repository can no longer be green while unable to compile against its own
  most recent release.
- A module that exists outside `go.work` is now a failure, in both directions.
- The conformance kit is inside the gate, so the enforcement row that claimed it
  ran in CI is true.
- A cross-module change states its release chain in the diff that makes it.

### Negative

- **The gate grew.** `workspace-check` vets nine modules a second time, in a
  different resolution mode. This is deliberate — the gate was measured at a 114s
  median and a 279s tail before this landed, so there was room
  ([#101](https://github.com/FabioCaffarello/fdos/issues/101)) — and it is the
  largest single addition the gate has taken.
- **R3 makes some in-progress work uncommittable.** Editing a module with stale
  pins now requires bumping them in the same change. That is the mechanism
  rather than a side effect, and #65 left it open precisely because it is a cost
  and not only a benefit.
- **`go.work` is now load-bearing for the gate**, not only for editors. A
  malformed one fails the build, where before it affected nobody's CI.
- **Two resolution modes mean two ways to be broken**, and a failure now has to
  be read for which mode it came from. The messages name the mode; that is
  documentation, not a mechanism.
- **This change had to fix what it found**: nine pins bumped across three
  modules, a test double given a method, a lint violation, a formatting fix and a
  tidy. A check that arrives with a repair attached is a check whose absence was
  costing something, but it also means this diff is larger than the mechanism it
  adds.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| The tree compiles against its own source | 3 | `make workspace-check`, negative-tested |
| Every module is in `go.work`, and every entry is a module | 3 | same check, both directions, negative-tested |
| The workspace check is not vacuous | 3 | the check's own resolution probe, negative-tested |
| A first-party pin names a published version | 3 | `make pin-check` R1, negative-tested |
| A pin does not exceed the newest tag | 3 | `make pin-check` R2 |
| A changed module pins current | 3 | `make pin-check` R3, negative-tested |
| A released module's pin skew is visible | 5 | `make pin-check` R4 — reported, not enforced |

## Alternatives considered

**Replace the `GOWORK=off` runs with a workspace build.** One mode, one set of
results, and it would catch everything this catches. Rejected: it silently
retires ADR-0004's discipline. The `GOWORK=off` runs are what prove a published
module resolves for a consumer with no workspace, and that consumer exists and
builds against `libs/contracts` today.

**Make R4 blocking, as [#65](https://github.com/FabioCaffarello/fdos/issues/65)
asked.** The strictest reading, and it would eliminate skew entirely. Rejected on
the measurement: thirteen instances today, none of them defects, and it would
redden `main` on every tag until a follow-up merged. R3 gets the benefit — pins
are current where someone is working — without the standing cost.

**Use `go build` rather than `go vet` for the workspace pass.** Faster and more
obviously "a build". Rejected because it does not compile test files, and the
defect that motivated the check lives in one.

**Leave `examples/` out and check it separately.** It is a demonstration, not a
library, so a lighter treatment is arguable. Rejected: it is the artifact a third
party copies, and ADR-0037 already claimed it ran in CI. A separate, lighter
check would be a second definition of what "verified" means.

**A `replace` directive instead of a workspace.** Would give local resolution
without `go.work`. Rejected outright: a `replace` in a published module is the
one thing ADR-0020's consumer proof exists to catch.

## Notes

- [#79](https://github.com/FabioCaffarello/fdos/issues/79) item 3 predicted that
  a byte-compared `prototext` fixture could not pass in both resolution modes. It
  does not reproduce: the fixture is parsed and compared with `proto.Equal`
  rather than byte-compared, and the test says why in a comment. The concern was
  correct when raised and had already been answered.
- `libs/analysis` has no release tag at all, so R3 treats it as always-unreleased.
  It has no first-party dependencies, so nothing follows from that today. That
  `release.yml` triggers on `libs/*/v*` and `fdoslint` has never been tagged is a
  separate observation, recorded here rather than acted on.
