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

**M1.5 — Canonical Domain Architecture.** Six RFCs proposed, awaiting review.

There is **no Go code**, no `go.mod`, no application, no test suite and no CI
pipeline. This is deliberate, not incomplete: the canonical financial model is
an output of the M1.5 RFCs, and writing code before that design lands would
pre-judge it.

What exists is the governance and enforcement substrate everything else will be
held to. An agent asked to "add a feature" at this stage should say there is
nothing yet to add it to, and point at the roadmap.

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
| M2 | Determinism toolchain — layer boundaries, custom analysers, reproducible builds |
| M3 | CI/CD and supply chain |
| M2.5 | AI engineering — agent playbooks, prompt contracts, staleness checks |
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
