---
type: doc
name: project-overview
description: What FDOS is, what it refuses to be, and where it currently stands
category: overview
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Project Overview

FDOS stores **immutable financial facts**, never financial state.

Every position, balance, performance metric and recommendation is derived —
reproducibly, deterministically, with full provenance — from an append-only
ledger of events. A report produced today must be regenerable years from now,
byte for byte, from the same ledger and the same versioned reference data.

That single constraint generates the entire architecture. When a design question
is open, the answer is almost always whichever option preserves reproducibility.

## Current state

**M8 complete — ingestion. A fact produced outside FDOS can now enter the
ledger.**

The module list is not repeated here. It changes, and a count in prose has a
half-life shorter than the document — read `go.work`, which cannot be wrong.

What each module is for: `libs/analysis` turns Constitution principles into
build errors; `libs/contracts` is the published protobuf surface; `libs/kernel`
the canonical types; `libs/ledger` the first bounded context; the `-wire`
modules the codecs between domain and wire, each with a round-trip suite;
`examples/ingest` the conformance kit a third-party producer runs.

Generated wire types are **not** canonical models: they carry `json:` tags,
import `sync` and `unsafe`, and hold mutable state, all of which the `impurity`
analyser correctly rejects in a domain package (ADR-0018). That is why every
canonical concept has two definitions and a conformance test proving they agree,
rather than one definition doing both jobs badly.

### What M8 shipped, and what it deliberately did not

`app.Ledger.AcceptHoldingClaim` is the entry point an external producer can
call. Every other one takes an `identity.ID`, which ADR-0007 and ADR-0022 forbid
a producer from minting — so before it, the public application surface was
structurally unusable from outside.

**Admission resolves nothing and mints nothing.** An identity that came into
existence because a stranger submitted a claim is an identity nobody chose, and
once a producer depends on that, removing it changes what the ledger does.
`UnresolvedClaims` reports which claims are waiting.

So the current gap is the mirror of the one M8 closed: **nothing mints, so a
claim cannot yet become an observation.** `Resolve`, `MintFor` and
`DeriveHoldingObserved` exist and no caller invokes them. That is M9.

There is still **no application** — `apps/` is empty, and a composition root
needs something to compose.

An agent asked to add a **second** bounded context, a canonical type, or a
message on the published contract surface should ask which ADR sequences it
first. `libs/contracts` is consumed outside this repository at a pinned version,
so adding to it changes somebody else's build.

What exists today, and is enforced:

```sh
make analyze   # nofloat · nondet · impurity · layering
make verify    # the full gate — 18 checks; exactly what CI runs (ADR-0014)
```

A `time.Now()` or a `float64` in a domain package fails the build, by name.
So does an unpinned GitHub Action, a rewritten ADR, a secret anywhere in
history, a binary that does not build twice to the same digest, or a playbook
in this directory naming a `make` target that no longer exists.

## What FDOS refuses to be

These are not preferences. Proposals violating them are rejected rather than
negotiated.

- **No mutable financial state.** Nothing overwrites a fact. Corrections are new
  events that supersede old ones.
- **No LLM in the truth path.** Models explain, summarise, prioritise and
  communicate. A model output must never be capable of entering the ledger.
- **No provider concepts in the domain.** Every external representation is
  normalised into the canonical model at the boundary. A broker's field name
  never reaches a business rule.
- **No domain dependency on infrastructure.** The domain knows nothing of
  databases, brokers, HTTP, browsers or frameworks.
- **No convenience at the cost of reproducibility.** When they conflict,
  reproducibility wins and the inconvenience is documented rather than removed.

## Open Core

The public core — engineering platform, canonical models, ledger, SDKs, APIs,
documentation, testing infrastructure — is Apache-2.0 (ADR-0002).

Authenticated providers, browser connectors and institution-specific plugins
live in separate private repositories and depend on this one only through
published, versioned contract modules. That boundary is proven by every CI run
rather than assumed (ADR-0004).

## Roadmap

| Milestone | Objective |
|-----------|-----------|
| M0 ✅ | Repository genesis — governance and enforcement substrate |
| M1 ✅ | Governance substrate — `.context`, contribution and release process |
| M1.5 ✅ | Canonical domain architecture — **RFCs only**. Six accepted (RFC-0001 … RFC-0006), recorded by ADR-0007 … ADR-0012 |
| M2 ✅ | Determinism toolchain — four analysers, reproducible builds, layer boundaries |
| M3 ✅ | CI/CD and supply chain — SHA-pinned inputs, SBOM, provenance, signing |
| M2.5 ✅ | AI engineering — prompt contracts, knowledge-base staleness checks |
| M3.5 ✅ | Developer experience — devcontainer, editor config, `make doctor` |
| M4 ✅ | Contracts — protobuf schemas, `buf breaking` gate, generated Go SDK |
| M5 ✅ | Open core boundary — published module, consumer proof, branch protection |
| M6 ✅ | First domain — the Ledger: kernel, bounded context, six principles at rung 1 |

M1.5 produces **no code**. That is its purpose.

The six RFCs are listed in [architecture.md](./architecture.md). All are
*Accepted*, each recorded by the ADR stating what it settled (ADR-0007 …
ADR-0012). The ADRs bind; their Notes sections list the questions each decision
deliberately leaves open and the milestone that must settle them.

## Authority

[`docs/constitution.md`](../../docs/constitution.md) is the highest authority in
the repository, followed by the ADRs in [`docs/adr/`](../../docs/adr/). This file
is derived from both; where it disagrees with them, they win.
