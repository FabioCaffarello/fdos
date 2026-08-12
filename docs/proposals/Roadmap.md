---
title: Roadmap continuation — M12a through M19
status: Provisional — proposal from the 2026-08-07 architectural audit
date: 2026-08-07
---

> **Provisional.** This document is a proposal produced by the 2026-08-07
> architectural audit. It is not accepted. Nothing may be implemented against
> it until an RFC and ADR accept the relevant part (ADR-0000,
> [AGENTS.md](../../AGENTS.md)). Where this document conflicts with an
> accepted ADR, the ADR governs until superseded.
>
> [`docs/ecosystem/roadmap.md`](../ecosystem/roadmap.md) is the authoritative
> milestone record. It ends at M12 (consumer enablement) with no successor.
> This document proposes the continuation.

# Roadmap continuation

Two standing rules frame every row:

- **The release gate.** No ADR-0039 signed binaries, and no new external
  contract consumers, before M12a completes and D2 is decided (M17 at the
  latest; earlier if an adopter appears). ADR-0039's own open-questions
  section asks for exactly this pause.
- **The allocation rule.** At most one meta/governance milestone per three
  domain milestones (see
  [Engineering-Principles.md](Engineering-Principles.md) §7). Every row
  below is a domain milestone except M12a, which is the debt the meta
  apparatus itself accumulated.

Each exit criterion is a question the system can answer afterwards that it
could not answer before (§4 of the same document).

| Milestone | Theme | Exit criterion (askable question) | Decision gates (must precede) | Audit items absorbed |
|---|---|---|---|---|
| **M12a** | **Foundation integrity** | "Is every encoding that will outlive us injective, ordered, and tested?" — all ten critical defects closed | A short encoding RFC (derivation pre-image, identity seed, namespace, storage time encoding — one document, they version together) | **All P0**: string-compared knowledge times; COUNT(*)-derived sequences and Load renumbering; derivation-address and identity-seed injectivity + the DNS-namespace constant; exact-context traps; `Quantize`; stream-name validation; the idempotency natural key; `reserved` policy; temporal package tests |
| **M13** | **The first answer** | "What is my position in instrument X, as of (effective, knowledge), aliases resolved, and why?" — a claim submitted over the wire becomes a queryable, explained position | RFC: the query surface (what may be asked, at what coordinates, at what cost); RFC: who mints (M9's unmet objective); ADR: correction redesign (replacement values, superseding refs, correction-of-correction) | P1: query surface; `EntitiesIdentified` traversal in `ProjectPosition`; the mint loop closed by an owned caller; upcaster implementation with the promised round-trip property test; the derivation store decision (build the sink or withdraw §8's rung-1 claim); the submission *response* contract |
| **M14** | **Reference** | "What is this instrument — name, currency, type, venue — and which dataset version says so?" | RFC: reference data (the context, dataset versioning, publication; what `ReferenceBinding` actually pins) | P1: reference-data RFC; first versioned dataset existing in some form; Constitution §9's second leg becomes real |
| **M15** | **Money flows** | "What is my position worth, in which currency, under which FX rate and rounding — and where did every number come from?" | RFC: price facts and valuation methods; the double-entry decision (accounting ledger or epistemic fact store — decided either way, in writing) | P1: first real exercise of the money kernel (currently zero call sites); `Rate`/price types; valuation as `Explained` derivations against pinned reference versions |
| **M16** | **Scale honesty** | "Does a 500-line statement ingest as one epistemic event, and does a million-fact stream project in bounded time?" | RFC: knowledge time for batches (ADR-0036's declined alternative E, promoted to its own decision **before** any batch endpoint); RFC: snapshots and incremental load (reopening ADR-0034's deferral, which was priced on a linear cost model the audit measured as quadratic) | P1: O(n²) read path; batch provenance (ADR-0010's own open item); as-of queries pushed to SQL |
| **M17** | **D2 and release** | "Who may write to this stream, who may read this ledger — and can an adopter run submitd off-loopback with a stated authorisation model?" | ADR: D2 (platform identity; ADR-0037 §5 already names the probable shape and asks for the ADR); then, and only then, ADR-0039 proceeds | P0 release gate lifts; the eight-times-deferred question closes; `SECURITY.md`, threat model, PII position land with it |
| **M18** | **Corporate actions** | "A split happened — is my position still right, and can I see the action that explains the change?" | RFC: corporate actions (action types, effects as generated Occurrences with derivations, identity continuity through `EntitiesIdentified`) | The Occurrence vocabulary grows beyond observations; the case internal identity was built for (ADR-0007's own motivating example) is finally modelled |
| **M19** | **MCP read surface** | "Can an AI agent ask for an explained position — and nothing else?" | RFC: MCP architecture (capabilities not raw rows; every tool takes explicit as-of coordinates; gated on D2 and the query surface) | The AI layer gets its minimum architecture: renderers and orchestrators consuming `Explained` values, with `ModelOutput` structurally unreachable from facts, exactly as the contract already enforces |

## Sequencing rationale

M12a precedes everything because every item in it is silent-corruption class
and permanently unfixable once real data exists — and no production data
exists today. That window is the cheapest engineering the project will ever
buy.

M13 before M14/M15 because the loop (submit → mint → resolve → project →
explain) is the platform's spine; reference data and valuation attach to a
working spine rather than to a diagram of one.

M16 before M18 because corporate actions arrive in batches (an issuer's
action fans out across every affected account), and batch knowledge-time
semantics must be decided before the first fan-out writes distorted history.

M17 sits where it does as a *latest* bound, not an earliest: D2 may be
decided any time earlier, and must be if any adopter materialises. The only
hard rule is its ordering before ADR-0039's binaries ship.
