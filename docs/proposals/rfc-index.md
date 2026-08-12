---
title: RFC Index
status: Provisional — proposal from the 2026-08-07 architectural audit
date: 2026-08-07
---

> Navigation aid, regenerable from [`docs/rfc/`](../rfc/), which is
> authoritative. This index records what exists and decides nothing. The
> *Open questions worth tracking* column collects each RFC's own surviving
> deferrals, as extracted by the 2026-08-07 audit — items an RFC named and no
> later decision has closed.

# RFC Index

15 RFCs, all Accepted. The RFC gate applies to domain decisions; governance
decisions go straight to ADR — which is why this log reads more
domain-focused than the ADR log.

| # | Title | Recorded by | Proposal in one line | Open questions worth tracking |
|---|---|---|---|---|
| [0001](../rfc/0001-identity-and-aggregate-boundaries.md) | Identity and aggregate boundaries | ADR-0007 | Opaque deterministic `EntityID`; positions are projections, never aggregates | Is the entity-kind list closed? Does the canonicalisation algorithm version belong in the namespace? Does `Account` belong in the public core? |
| [0002](../rfc/0002-money-and-numeric-representation.md) | Money and numeric representation | ADR-0008 | `apd` decimals; currency in the type; no default rounding context | Per-currency scale constraints; a distinct `Rate` type; maximum supported precision — all three now load-bearing for valuation |
| [0003](../rfc/0003-bitemporal-event-model.md) | Bitemporal event model | ADR-0009 | Two axes, universal; total order `(effective_from, knowledge_time, sequence)` | `Instant` precision (interacts with the ordering tiebreaker — the audit's string-comparison defect landed exactly here); open-start intervals; projection semantics of corrections-to-corrections; one as-of pair or one per reference dataset |
| [0004](../rfc/0004-provenance-and-reference-data.md) | Provenance and reference data | ADR-0010 | Mandatory envelope; ordinal confidence; versioned bitemporal reference datasets | Batch/envelope size for high-frequency facts (blocks market data); who publishes reference datasets and their trust model; `Disputed` as level or separate assertion |
| [0005](../rfc/0005-event-taxonomy-and-schema-evolution.md) | Event taxonomy and schema evolution | ADR-0011 | Occurrence vs Observation; additive-only per-type schemas; upcast-on-read | The `Assertion` third kind (resolved by default when `EntityMinted` shipped as an Occurrence); upcaster chain composition across majors; fact-type retirement |
| [0006](../rfc/0006-explainability-as-a-return-type.md) | Explainability as a return type | ADR-0012 | `Explained[T]` with combinators; LLMs may render a trace, never produce one | The precise scope boundary ("calculations producing financial values"); combinators vs codegen; trace storage volume; whether pruning violates Constitution §6 — plus the audit's finding that no derivation store exists |
| [0007](../rfc/0007-identity-resolution-and-the-acquisition-boundary.md) | Identity resolution and the acquisition boundary | ADR-0022 | Connectors emit unresolved claims; minting is a ledgered fact; resolution reads the ledger | What reports claims that never resolve, and to whom; the `EntitiesIdentified` producer path (still unused, untested, uncodec'd); how a consumer asks *how* inferred |
| [0008](../rfc/0008-narrowing-two-responsibility-matrix-rows.md) | Narrowing two matrix rows | ADR-0026 | "Contracts" → "canonical contracts"; toolchain ownership by use, not language | — |
| [0009](../rfc/0009-renumbering-invariants-and-redacting-the-matrix.md) | Renumbering invariants, redacting the matrix | ADR-0027 | I→E renumbering; provider redaction; E9 added, admitted unmet | — |
| [0010](../rfc/0010-the-public-surface-receives-a-claim.md) | The public surface receives a claim | ADR-0029 | Claims not resolved identities; a library constructs, only the ledger gates | Delivery clause superseded by ADR-0037; the D2 questions it deferred remain open |
| [0011](../rfc/0011-provenance-admissibility.md) | Provenance admissibility | ADR-0028 | `SourceRef` as content hash with unspecified referent; honest `unmediated` interpreter | Whether a rejection carries a `SourceRef` (its own RFC — the rejections-are-not-facts silence the audit flagged); signed attestation over the acquisition record (blocked on D2's trust root) |
| [0012](../rfc/0012-the-submission-shape.md) | What a producer submits | ADR-0030 | A submission message omitting knowledge time | Transport settled by ADR-0037; the response shape is still undefined (audit: half the ingress interaction is not in the contract); E9 still unmet by this alone |
| [0013](../rfc/0013-minting-is-an-owned-act-and-canonicalisation-is-per-scheme.md) | Minting is an owned act | ADR-0033 | `MintIdentity` resolves-then-refuses; folds only where an issuing standard exists | Who is entitled to mint (answerable when D2 is); what notices claims accumulating unresolved — `UnresolvedClaims` makes it askable, nothing asks |
| [0014](../rfc/0014-the-ledger-event-store.md) | The ledger event store | ADR-0034 | `Append` with expectation; store-assigned sequence; SQLite via pure-Go driver | Whether `Load` stays whole-stream (deferred on a linear cost model; the audit measured quadratic — reopen); the `ErrStaleRead` retry contract; clock skew before a second writer deploys |
| [0015](../rfc/0015-the-submission-service-and-the-admission-race.md) | The submission service and the admission race | ADR-0036, ADR-0037 | Clock read and append serialised per stream; minimal HTTP transport, loopback by default | D2 (who may write to a named stream); the narrowed-but-open retry contract for `MintIdentity`/`CorrectFact`; shard count as a recorded constant |
