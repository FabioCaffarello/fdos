---
id: ADR-0053
title: fdoslint is not distributed, and the consumer its release was for does not exist
status: Accepted
date: 2026-08-13
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0053 — `fdoslint` is not distributed, and the consumer its release was for does not exist

> Settles a question three ADRs left open in their notes:
> [ADR-0044](0044-the-gate-compiles-the-tree-as-a-workspace.md),
> [ADR-0045](0045-the-affected-graph-is-the-release-graph.md) and
> [ADR-0046](0046-publishing-a-tag-is-a-dispatched-act.md) each recorded that
> `libs/analysis` has no tag and moved on.

## Context

[ADR-0047](0047-a-release-carries-what-the-module-publishes.md) made a release
carry what its module publishes. Before it, `release.yml` hardcoded `fdoslint`
and triggered on `libs/*/v*`, so **twenty library releases carried a linter's
binaries** under other modules' tags.

Fixing that had a consequence neither ADR-0047 nor the work that followed it
stated: **`fdoslint` is now distributed nowhere.** `libs/analysis` has never been
tagged, so no release carries it at all.

That matters because ADR-0014 gave a reason for releasing it, and
`release.yml` carried the same sentence for months:

> The only artifact FDOS produces today is `fdoslint` […] It is released so that
> a consumer can verify the tool gating their code was built from the source it
> claims.

So the question is not "should we tag `libs/analysis`". It is **who is the
consumer that sentence describes**.

### Measured rather than assumed

**The gate does not use a released binary.** `scripts/run-analyzers.sh` builds it
from source into a temporary directory on every run:

```sh
BIN="$(mktemp -d)/fdoslint"
( cd "${ROOT}/${ANALYSIS_MODULE}" && GOWORK=off "$GO" build -trimpath -o "$BIN" ./cmd/fdoslint )
```

**Nothing outside this repository references it.** Not
[`boundary.md`](../ecosystem/boundary.md), which defines what crosses between
the two repositories; not [`contracts.md`](../ecosystem/contracts.md), which
already says so directly:

> `libs/analysis` is not published at all: it is tooling, and nothing outside
> this repository has reason to link it.

**The purity rules are scoped to this repository.**
[ADR-0021](0021-purity-rules-scope.md) settles what they cover, and it is code
here.

So: the binary gated nobody's code but this repository's, and this repository
builds it from source. **The consumer in ADR-0014's sentence did not exist when
it was written and does not exist now.**

## Decision

### 1. `fdoslint` is not distributed, and that is written down

`libs/analysis` stays untagged. No release carries the binary, and the registry
says so about the binary as well as the module.

This is not a new restriction — it is the removal of a claim. The twenty
releases that carried `fdoslint` were shipping an artifact with no consumer, and
attaching provenance to it answered a question nobody was asking.

### 2. ADR-0014's rationale is corrected here rather than repeated

ADR-0014 is immutable and its sentence stands as written. What this ADR records
is that the sentence described a consumer that did not exist — so the reasoning
it supports ("the attestation is worth exactly as much as the weakest input")
survives entirely, while the example it used does not.

Nothing about action pinning, digests or `make verify` changes. The correction
is to one worked example in one paragraph.

### 3. What would reverse this

If the purity rules ever gate code outside this repository, the sentence becomes
true and a release becomes meaningful. The live form of that question is the
plugin conformance suite —
[#53](https://github.com/FabioCaffarello/fdos/issues/53), which ADR-0021 and
B-002 both leave open. **That decision comes first**; a release follows it rather
than anticipating it.

Nothing in the machinery blocks it. `release-tag` and `release-artifacts` will
release `libs/analysis` the day someone decides to, and
[#134](https://github.com/FabioCaffarello/fdos/pull/134) made a module's first
release possible.

### 4. What this does not decide

**Nothing about the twenty existing releases.** ADR-0047 already decided they
are history attached to immutable tags. This adds that what they carried had no
consumer, which makes them less wrong than they looked and no more correctable.

## Consequences

### Positive

- A question three ADRs deferred is answered, with the evidence in one place.
- The repository stops implying it distributes a tool it does not.
- `libs/analysis` staying untagged is now a decision rather than an omission,
  which is the difference between "not done" and "deliberately not done" that
  `docs/blocked.md` exists to preserve.

### Negative

- **A future contributor may read ADR-0014's sentence first.** It is immutable
  and prominent, and this correction is one ADR further along. Nothing links
  from there to here, because nothing can.
- **`fdoslint` has no verifiable distribution**, so if it ever acquires an
  outside user, that user starts from source with no attestation — which is
  where every consumer of every Go tool starts, and is why this was worth
  checking rather than assuming.
- **The purity rules remain enforceable only from inside.** A third party who
  wanted to check their code against them would build the tool themselves. That
  is the honest state and #53 is where it changes.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| `fdoslint` is built from source by the gate | 3 | `scripts/run-analyzers.sh`, which builds it every run |
| `libs/analysis` is not published | 6 | this decision and the registry. Nothing prevents a tag |

The second row is deliberate. A check that refused to release `libs/analysis`
would have to be removed the day #53 decides otherwise, and a mechanism whose
purpose is to be deleted is worse than a sentence.

## Alternatives considered

**Tag `libs/analysis` and release `fdoslint` properly.** It would restore the
distribution ADR-0047 removed, and the machinery is ready. Rejected on the
evidence: the release would exist for a consumer nobody can name, and the
registry says the opposite in a document ADR-0024 calls part of the interface.
Publishing to contradict your own registry is worse than not publishing.

**Say nothing and let `libs/analysis` stay untagged.** The status quo, and it
costs nothing today. Rejected because three ADRs have now recorded the same
observation and moved on, which is how something becomes folklore instead of a
decision.

**Release it as an application under `apps/`.** ADR-0013 considered exactly this
split — `libs/analysis` plus `apps/fdoslint` — and rejected it. Nothing here
reopens that.

**Keep publishing it under library tags.** Not a straw man only because it is
what happened for twenty releases. Rejected by ADR-0047 on the grounds that a
signature over the wrong artifact is worse than none, and that reasoning is
unaffected by whether the artifact has a consumer.
