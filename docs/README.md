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
  - Knowledge intended primarily for AI agents (belongs in .dotcontext/)
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

## ADR or RFC?

Write an **ADR** when the decision is clear and needs recording. Write an **RFC**
when the decision requires exploration before it can be made; an accepted RFC is
followed by the ADRs that record what it settled.

Any change to repository structure, module boundaries, the public contract
surface, the toolchain, enforcement mechanisms, or the Constitution requires an
ADR.

## Relationship to `.dotcontext/`

`docs/` is written for humans and is the authoritative record. `.dotcontext/` is
structured knowledge for AI agents and is *derived from* this directory — never
the reverse. Where the two disagree, `docs/` wins, and the disagreement is a bug
in the derivation.
