---
directory: docs
purpose: Governance, decisions and durable engineering knowledge for humans.
owner: "@FabioCaffarello"
allowed:
  - The Engineering Constitution
  - Architecture Decision Records (docs/adr/)
  - Requests for Comments (docs/rfc/)
  - Architecture, security, performance and operational documentation
  - Diagrams and reference material supporting the above
forbidden:
  - Generated API reference (produced from contracts at M4, never hand-written)
  - Knowledge intended primarily for AI agents (belongs in .context/)
  - Anything that duplicates rather than references a decision
---

# docs

Documentation is production code (Constitution §14). This directory holds the
material that outlives any particular implementation: what FDOS decided, why,
and under what constraints.

## Structure

| Path | Contents |
|------|----------|
| `constitution.md` | The principles that govern FDOS. Highest authority in the repository. |
| `adr/` | Architecture Decision Records. Append-only, immutable (ADR-0000). |
| `rfc/` | Requests for Comments — decisions requiring design exploration. |
| `ecosystem/` | The boundary between FDOS and its consuming repositories (ADR-0023). |
| `blocked.md` | Work decided and not finished, with what it is waiting on. |
| `disclosure.md` | What this public repository has revealed about the private side, and what of it is permanent. |

### `ecosystem/`

Authoritative for **both** repositories, not just this one. `invariants.md` and
the responsibility matrix in `boundary.md` are Tier 0: authored here, vendored
verbatim downstream, amended only by an RFC here plus an ADR in both. The
delimiting markers in those files are what make "verbatim" checkable.

`contracts.md` is the published-contract registry and is part of the interface —
a version missing from it is undiscoverable by the only channel the other
repository may use. `labels.md` records the issue taxonomy both repositories
share, so it is reviewable as a file rather than only as GitHub configuration.

### Current RFC set — M1.5, all Accepted

Each acceptance is recorded by the ADR that states what the RFC settled — a
rule `make rfc-check` enforces: an RFC marked `Accepted` with no ADR
referencing it fails the build.

| RFC | Proposal | Decided by |
|-----|----------|------------|
| [0001](rfc/0001-identity-and-aggregate-boundaries.md) | Identity and aggregate boundaries | [ADR-0007](adr/0007-internal-deterministic-identity.md) |
| [0002](rfc/0002-money-and-numeric-representation.md) | Money, quantity and numeric representation | [ADR-0008](adr/0008-decimal-money-explicit-rounding.md) |
| [0003](rfc/0003-bitemporal-event-model.md) | Bitemporal event model | [ADR-0009](adr/0009-universal-bitemporality.md) |
| [0004](rfc/0004-provenance-and-reference-data.md) | Provenance envelope and reference data versioning | [ADR-0010](adr/0010-provenance-envelope-reference-versioning.md) |
| [0005](rfc/0005-event-taxonomy-and-schema-evolution.md) | Event taxonomy and schema evolution | [ADR-0011](adr/0011-fact-taxonomy-and-upcasting.md) |
| [0006](rfc/0006-explainability-as-a-return-type.md) | Explainability as a return type | [ADR-0012](adr/0012-explained-return-type.md) |

## ADR or RFC?

Write an **ADR** when the decision is clear and needs recording. Write an **RFC**
when the decision requires exploration before it can be made; an accepted RFC is
followed by the ADRs that record what it settled.

Any change to repository structure, module boundaries, the public contract
surface, the toolchain, enforcement mechanisms, or the Constitution requires an
ADR.

## Relationship to `.context/`

`docs/` is written for humans and is the authoritative record. `.context/` is
structured knowledge for AI agents and is *derived from* this directory — never
the reverse. Where the two disagree, `docs/` wins, and the disagreement is a bug
in the derivation.
