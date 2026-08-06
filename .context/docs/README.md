# Documentation Index

Repository knowledge for AI agents working on FDOS.

**This directory is derived from [`docs/`](../../docs/), never the reverse.**
Where the two disagree, `docs/` wins and the disagreement is a bug in the
derivation.

Start with the project overview — in particular the part recording which
milestones have landed, and which parts of the repository are still empty by
design rather than by omission.

## Guides

| Guide | Contents |
|-------|----------|
| [Project Overview](./project-overview.md) | What FDOS is, what it refuses to be, current milestone, roadmap |
| [Architecture](./architecture.md) | Repository structure, intended layering, determinism constraints |
| [Data Flow](./data-flow.md) | Intended source → ledger → view pipeline. **Design intent, not implementation** |
| [Development Workflow](./development-workflow.md) | RFC → ADR → implementation, supersession, definition of done |
| [Glossary](./glossary.md) | Governance terms (binding) and domain terms (provisional) |
| [Testing Strategy](./testing-strategy.md) | Negative testing today; property-based and golden-file testing from M2 |
| [Security](./security.md) | Threat model, secret policy, supply chain plan, current gaps |
| [Tooling](./tooling.md) | Toolchain pins, `make` targets, enforcement scripts |

## Authority

1. [`docs/constitution.md`](../../docs/constitution.md) — fourteen principles and
   the enforcement ladder. Highest authority.
2. [`docs/adr/`](../../docs/adr/) — accepted decisions. Append-only, immutable.
3. Directory `README.md` front matter — binding contract per directory.
4. This directory — derived.

## Reading these documents

Anything marked *provisional* is design intent, not decision. The canonical
financial model, identifiers, aggregate boundaries, event taxonomy, bitemporal
scope, reference-data versioning and explainability are all **M1.5 RFC outputs**
and are not yet settled.

Implementing against provisional content converts a hypothesis into a decision
without an ADR. That is the most common way to damage this repository at its
current stage.
