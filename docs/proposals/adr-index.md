---
title: ADR Index
status: Provisional — proposal from the 2026-08-07 architectural audit
date: 2026-08-07
---

> Navigation aid, regenerable from [`docs/adr/`](../adr/), which is
> authoritative. This index records what exists and decides nothing. The
> *Class* column (Domain / Meta / Mixed) is the 2026-08-07 audit's
> classification of each decision's subject matter, not part of any ADR.

# ADR Index

40 ADRs. Supersession chains: ADR-0001 → ADR-0006, ADR-0017 → ADR-0019,
ADR-0029 → ADR-0037. One proposed and unaccepted: ADR-0039.

| # | Title | Status | Class | Decision in one line |
|---|---|---|---|---|
| [0000](../adr/0000-record-architecture-decisions.md) | Record architecture decisions | Accepted | Meta | ADRs are an append-only, immutable log; corrections supersede, never edit |
| [0001](../adr/0001-context-management.md) | `.dotcontext` as AI knowledge directory | Superseded | Meta | Reversed one day later by ADR-0006 |
| [0002](../adr/0002-license.md) | Apache-2.0 | Accepted | Meta | Public core under Apache-2.0; private repos separately licensed |
| [0003](../adr/0003-module-path.md) | Module path | Accepted | Meta | `github.com/FabioCaffarello/fdos`, diverging from the then-repo name |
| [0004](../adr/0004-module-granularity.md) | Module granularity | Accepted | Meta | One Go module per `libs/*`; CI builds with `GOWORK=off` |
| [0005](../adr/0005-enforcement-ladder.md) | Enforcement ladder | Accepted | Meta | Six rungs; every principle enforced at the highest feasible one; climbing is mandatory |
| [0006](../adr/0006-context-directory-naming.md) | `.context` as AI knowledge directory | Accepted | Meta | Supersedes ADR-0001; tooling default wins |
| [0007](../adr/0007-internal-deterministic-identity.md) | Internal deterministic identity | Accepted | Domain | Opaque UUIDv5 entity IDs; seed is a birth certificate, never a lookup key; external IDs are assertions |
| [0008](../adr/0008-decimal-money-explicit-rounding.md) | Decimal money, explicit rounding | Accepted | Domain | Arbitrary-precision decimal; currency in the type; no division without a rounding context |
| [0009](../adr/0009-universal-bitemporality.md) | Universal bitemporality | Accepted | Domain | Every fact carries both axes; knowledge time machine-assigned; no query has a default as-of |
| [0010](../adr/0010-provenance-envelope-reference-versioning.md) | Provenance envelope, reference versioning | Accepted | Domain | Mandatory envelope; ordinal confidence; reference-dataset versions pinned per fact |
| [0011](../adr/0011-fact-taxonomy-and-upcasting.md) | Fact taxonomy and upcasting | Accepted | Domain | Occurrence vs Observation; per-type versions; stored events never migrated — upcast on read |
| [0012](../adr/0012-explained-return-type.md) | Explained return type | Accepted | Domain | Domain calculations return `Explained[T]`; combinators build the trace; no model output as a trace |
| [0013](../adr/0013-layer-structure-and-module-topology.md) | Layer structure and module topology | Accepted | Meta | Modules per bounded context; `domain`/`app`/`adapters` as packages; kernel stays minimal |
| [0014](../adr/0014-ci-runs-make-and-pins-everything.md) | CI runs make, pins everything | Accepted | Meta | CI invokes `make` only; every action SHA-pinned; hooks are convenience |
| [0015](../adr/0015-ai-engineering-policy.md) | AI engineering policy | Accepted | Meta | Agent work passes the same gate; prompt contracts declared and checked as data |
| [0016](../adr/0016-developer-experience.md) | Developer experience | Accepted | Meta | `make` is the only task runner; `AGENTS.md` the single agent entry point |
| [0017](../adr/0017-claude-export-is-versioned.md) | Claude export versioned | Superseded | Meta | Reversed by ADR-0019 |
| [0018](../adr/0018-contract-surface-is-protobuf.md) | Contract surface is protobuf | Accepted | Mixed | Protobuf with `buf breaking`; wire types never domain types; `Decimal` is canonical text |
| [0019](../adr/0019-claude-export-is-not-versioned.md) | Claude export not versioned | Accepted | Meta | Supersedes ADR-0017; `.context/` is the sole reviewed roster |
| [0020](../adr/0020-open-core-boundary-and-pull-request-workflow.md) | Open-core boundary, PR workflow | Accepted | Meta | Repo renamed `fdos`; consumer proof at release; mandatory pull requests |
| [0021](../adr/0021-purity-rules-scope.md) | Purity rules scope | Accepted | Meta | Analysers cover the kernel; test and generated files exempt |
| [0022](../adr/0022-minting-an-identity-is-a-fact.md) | Minting an identity is a fact | Accepted | Domain | Connectors emit claims; `EntityMinted` is ledgered; `HoldingObserved` is only ever derived |
| [0023](../adr/0023-ecosystem-boundary-and-one-way-contract-flow.md) | Ecosystem boundary | Accepted | Meta | Tier-0 corpus; contracts flow one way; disputes D1–D5 recorded not resolved |
| [0024](../adr/0024-contract-lifecycle-and-versioning.md) | Contract lifecycle and versioning | Accepted | Meta | Semver per module tag; breaking change is a seven-step process |
| [0025](../adr/0025-consumer-facing-surface-is-the-contracts-module.md) | Consumer surface is contracts | Accepted | Meta | Only `libs/contracts` is offered; other published modules carry no promise |
| [0026](../adr/0026-canonical-contracts-and-language-toolchains.md) | Canonical contracts | Accepted | Meta | Only canonical contracts are FDOS's; toolchain ownership not named by language |
| [0027](../adr/0027-invariant-renumbering-and-matrix-redaction.md) | Invariants E1–E9, matrix redaction | Accepted | Meta | Renumbered invariants; no provider named; E9 added (open core stands alone) |
| [0028](../adr/0028-provenance-admissibility.md) | Provenance admissibility | Accepted | Domain | `SourceRef` is `sha256:`-prefixed content hash, referent unspecified; `unmediated` interpreter sentinel |
| [0029](../adr/0029-the-public-surface-receives-a-claim.md) | Public surface receives a claim | Superseded | Domain | Claims not resolved IDs; ledger revalidates as hostile; superseded on the delivery clause by ADR-0037 |
| [0030](../adr/0030-the-submission-shape.md) | The submission shape | Accepted | Domain | A submission message that omits knowledge time; `stream` producer-supplied and unguarded (D2) |
| [0031](../adr/0031-prevc-is-the-working-agreement.md) | PREVC working agreement | Accepted | Meta | PREVC as a renaming of the existing draft-PR workflow, with a recorded reversal trigger |
| [0032](../adr/0032-blocked-work-is-registered-as-issues.md) | Blocked work as issues | Accepted | Meta | `docs/blocked.md` tombstoned; blocked work moves to labelled issues |
| [0033](../adr/0033-minting-is-an-owned-act-and-canonicalisation-is-per-scheme.md) | Minting is owned; canonicalisation per scheme | Accepted | Domain | `MintIdentity` resolves-then-refuses; folds only for schemes with issuing standards |
| [0034](../adr/0034-the-ledger-event-store.md) | The ledger event store | Accepted | Mixed | Append with expectation; store assigns sequence; store refuses non-monotonic knowledge time |
| [0035](../adr/0035-the-sqlite-driver-and-its-provenance-risk.md) | SQLite driver, provenance risk | Accepted | Meta | `modernc.org/sqlite`, `CGO_ENABLED=0`; the cannot-rederive-from-upstream risk accepted unmitigated |
| [0036](../adr/0036-knowledge-time-is-assigned-under-the-streams-write-lock.md) | Knowledge time under the write lock | Accepted | Mixed | Clock read and append serialised per stream under sharded locks; two-process case explicitly open |
| [0037](../adr/0037-delivery-includes-a-service-the-adopter-operates.md) | Delivery includes an adopter-operated service | Accepted | Mixed | Supersedes ADR-0029's delivery clause; `apps/submitd` published, FDOS operates none; loopback by default |
| [0038](../adr/0038-fdos-tracks-the-go-patch-line.md) | Track the Go patch line | Accepted | Meta | Reachable stdlib advisories fixed by moving the pin, never allowlisted |
| [0039](../adr/0039-applications-are-released-as-signed-binaries.md) | Applications released as signed binaries | **Proposed** | Meta | One signing path for apps and libraries; its own text flags releasing an ingress before D2 |
