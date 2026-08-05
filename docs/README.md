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

### Current RFC set — M1.5, all Proposed

| RFC | Proposal |
|-----|----------|
| [0001](rfc/0001-identity-and-aggregate-boundaries.md) | Identity and aggregate boundaries |
| [0002](rfc/0002-money-and-numeric-representation.md) | Money, quantity and numeric representation |
| [0003](rfc/0003-bitemporal-event-model.md) | Bitemporal event model |
| [0004](rfc/0004-provenance-and-reference-data.md) | Provenance envelope and reference data versioning |
| [0005](rfc/0005-event-taxonomy-and-schema-evolution.md) | Event taxonomy and schema evolution |
| [0006](rfc/0006-explainability-as-a-return-type.md) | Explainability as a return type |

**Proposed is not decided.** None of these binds anything until it is accepted
and an ADR records what it settled — a rule `make rfc-check` enforces: an RFC
marked `Accepted` with no ADR referencing it fails the build.

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
