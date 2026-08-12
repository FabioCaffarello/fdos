---
title: The 32-area review — verdicts
status: Provisional — proposal from the 2026-08-07 architectural audit
date: 2026-08-07
---

# The 32-area review

> **Status: provisional.** Verdicts from the 2026-08-07 architectural audit,
> condensed. The full findings report — execution-verified defects with
> file:line evidence, severity rankings, priority matrix — was published as an
> artifact alongside this package. Nothing here is an accepted decision
> (ADR-0000, `AGENTS.md`).

Each verdict is one of: **Sound** (keep as is), **Sound-but-broken**
(right design, defective implementation), **Redesign** (wrong shape),
**Missing** (does not exist), **Cut** (should not exist / not now).

| # | Area | Verdict | Basis |
|---|---|---|---|
| 1 | Repository vision | **Sound-but-broken** | Principles right; 65% of ADRs meta; the vision document ADR-0013 cites never existed → [`Vision.md`](Vision.md) |
| 2 | Constitution | **Sound** | Fourteen principles correct. §15's claims are shape-validated only → [`Constitution-Amendments.md`](Constitution-Amendments.md) |
| 3 | Domain model | **Sound-but-broken** | Epistemology (identity/time/provenance) excellent; finance absent; one fact family after 16 milestones → [`Domain-Model.md`](Domain-Model.md) |
| 4 | Canonical financial model | **Sound-but-broken** | Kernel types right in intent; rounding, seed/hash injectivity, namespace defects measured; `Date`/`Rate`/`Quantize`/`Allocate` missing → [`Canonical-Financial-Model.md`](Canonical-Financial-Model.md) |
| 5 | Ubiquitous language | **Sound** | Occurrence/Observation, claim/fact, mint/resolve are real, enforced vocabulary. Gaps: `Assertion` kind undecided; `ticker` vs `symbol` ungoverned |
| 6 | Bounded contexts | **Missing (map)** | Kernel + Ledger exist and are correctly cut; Reference, Portfolio, Market Data, Corporate Actions, Analytics named nowhere but one ADR sentence → [`Domain-Model.md`](Domain-Model.md) |
| 7 | Aggregates | **Sound** | Instrument/Party/Account/LedgerStream as identity-only aggregates; positions correctly refused aggregate status (projection-only, structurally unpersistable) |
| 8 | Entities | **Missing (description)** | Entities have identity and nothing else — an instrument is a UUID with no name, currency, type, venue. Reference context owed |
| 9 | Value objects | **Sound-but-broken** | Construction-or-nothing discipline is real; exported string types (`Currency`, `Unit`, `Parameter` fields) punch holes in it; `Weakest()` launders invalid confidence |
| 10 | Domain services | **Sound** | Pure, total, clockless domain functions, analyser-enforced in intent. The analysers themselves are bypassable (area 29) |
| 11 | Commands | **Sound** | Use-case layer per context (`app.Ledger`) with explicit commands; correct. Six of seven have no production caller (area 32) |
| 12 | Events | **Sound-but-broken** | Taxonomy decision (ADR-0011) is the corpus's best; implementation collapsed three correction types into an ignored enum; no upcaster exists; `Assertion` resolved by default |
| 13 | Queries | **Missing** | No query surface, endpoint, RFC, or milestone. The system cannot answer a question. Largest single gap |
| 14 | Projections | **Sound-but-broken** | `Explained[Position]` fold is the right shape; ignores `EntitiesIdentified` (positions split across aliases), ignores two correction kinds, O(n²) load path |
| 15 | Ledger architecture | **Sound-but-broken** | Admission boundary excellent; store has four measured critical defects (string time comparison, sequence renumbering, no upcasting, unvalidated streams) → [`Ledger.md`](Ledger.md) |
| 16 | Corporate-actions engine | **Missing** | Named only as a reason identity is hard. Design proposed → [`Corporate-Actions.md`](Corporate-Actions.md) |
| 17 | Portfolio engine | **Missing** | Cross-stream projection currently unrepresentable — the flagship use case cannot be expressed → [`Financial-Engines.md`](Financial-Engines.md) |
| 18 | Fixed-income engine | **Missing** | Needs kernel `Date` + interval algebra + day-count reference data first → [`Financial-Engines.md`](Financial-Engines.md) |
| 19 | Credit intelligence | **Cut** | One hypothetical mention, no data source, no question to answer. Name reserved, nothing built |
| 20 | Risk engine | **Missing (deferred, correctly)** | Deterministic exposure decomposition first; statistical risk later, behind the model/fact boundary |
| 21 | Knowledge layer | **Missing** | `Explained[T]` has zero adopters and intermediate derivations evaporate — no store, no sink → [`Knowledge-Layer.md`](Knowledge-Layer.md) |
| 22 | Knowledge graph | **Cut (as storage), Sound (as projection)** | Reject a graph database now; accept graph-shaped projection contract + one fitness test; reconsider only when a relational read model cannot answer a real query |
| 23 | MCP architecture | **Missing (correctly, so far)** | Nothing to expose before a read model exists. Target: capabilities with mandatory as-of pairs, read-only, gated on D2 → [`MCP.md`](MCP.md) |
| 24 | API contracts | **Sound-but-broken** | Proto surface well-designed and superbly documented; no response shape for submissions; zero `reserved` fields; `Any` payloads outside the breaking gate |
| 25 | Plugin contracts | **Sound** | Two-repo split, Tier-0 matrix, four boundary tests, one-way contract flow: correct and battle-tested. Ratify D3; write the consumer compatibility policy |
| 26 | Repository structure | **Sound** | Context-module topology (ADR-0013) right; `examples/` outside the gate, `ledger-sqlite` outside `go.work`, siblings tested at stale floors → [`Repository-Architecture.md`](Repository-Architecture.md) |
| 27 | ADR strategy | **Sound-but-broken** | Append-only immutability right; deletion undetected for uncited ADRs; no "accepted-unimplemented" state; no lower tier for small decisions → [`adr-index.md`](adr-index.md) |
| 28 | RFC strategy | **Sound** | RFC-gating produced the corpus's best decisions; asymmetry (governance skips the gate) is why meta decisions outnumber domain 26:14 → [`rfc-index.md`](rfc-index.md) |
| 29 | Testing strategy | **Sound-but-broken** | Negative-test culture is real for direct forms only: analysers pass indirect violations; conformance never sees foreign messages; property generators avoid the cliffs; `temporal` has zero tests |
| 30 | Observability | **Missing** | No tracing, no metrics, no structured logging decision; named once as deferred (ADR-0018). Acceptable pre-read-surface; unacceptable past it |
| 31 | Security | **Sound-but-broken** | Supply-chain posture unusually strong (pins, SBOM, signing, honest ADR-0035); no `SECURITY.md`, no threat model in `docs/`, no PII position, D2 open while a service ships, release path escapes the CGO pin |
| 32 | Roadmap | **Redesign** | Ends at M12 with the platform's substance unscheduled; every proposed continuation milestone ends with an askable question → [`Roadmap.md`](Roadmap.md) |

## Reading order

For a newcomer deciding what to accept first: [`Vision.md`](Vision.md) →
[`Domain-Model.md`](Domain-Model.md) → [`Ledger.md`](Ledger.md) →
[`Roadmap.md`](Roadmap.md). The Phase-0 integrity items in the audit report
precede everything — they are the only items whose cost is permanent.
