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

**M3 complete — CI/CD and supply chain.** Next: M2.5 (AI engineering).

There is **no domain code and no application**. The only Go in the repository is
`libs/analysis` — the four static analysers that turn Constitution principles
into build errors — plus its `cmd/fdoslint`.

This is deliberate, not incomplete. The canonical model is defined by ADR-0007 …
ADR-0012 but lands as code with the Ledger at **M6**, so that the first domain
is built under the constraints rather than retrofitted to them.

An agent asked to "add a feature" should say there is nothing yet to add it to,
and point at the roadmap. An agent asked to create `libs/kernel` or a bounded
context ahead of M6 should say the same.

What exists today, and is enforced:

```sh
make analyze   # nofloat · nondet · impurity · layering
make verify    # the full gate — 15 checks; exactly what CI runs (ADR-0014)
```

A `time.Now()` or a `float64` in a domain package fails the build, by name.
So does an unpinned GitHub Action, a rewritten ADR, a secret anywhere in
history, or a binary that does not build twice to the same digest.

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
| M3 ✅ | CI/CD and supply chain — 15 checks per PR, SHA-pinned inputs, SBOM, provenance, signing |
| **M2.5** | AI engineering — agent playbooks, prompt contracts, staleness checks |
| M3.5 | Developer experience |
| M4 | Contracts and observability — proto → buf → OpenAPI → SDK → MCP → docs |
| M5 | Open core boundary |
| M6 | First domain — the Ledger, as a vertical slice |

M1.5 produces **no code**. That is its purpose.

The six RFCs are listed in [architecture.md](./architecture.md). All are
*Accepted*, each recorded by the ADR stating what it settled (ADR-0007 …
ADR-0012). The ADRs bind; their Notes sections list the questions each decision
deliberately leaves open and the milestone that must settle them.

## Authority

[`docs/constitution.md`](../../docs/constitution.md) is the highest authority in
the repository, followed by the ADRs in [`docs/adr/`](../../docs/adr/). This file
is derived from both; where it disagrees with them, they win.
