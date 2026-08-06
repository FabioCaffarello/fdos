---
id: RFC-0008
title: Narrowing two responsibility-matrix rows
status: Accepted
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0008 — Narrowing two responsibility-matrix rows

> **Accepted**, recorded by
> [ADR-0026](../adr/0026-canonical-contracts-and-language-toolchains.md).


> **Banner — disclosure redaction (`ecosystem/v0.3.0`).** This RFC names
> identifiers belonging to the private repository. They are retained here
> because they are the *subject* of the decision and removing them would make
> the reasoning unreadable; current documents no longer repeat them. The
> boundary rule and what remains permanently published are recorded in
> [`../disclosure.md`](../disclosure.md).

## Why this needs an RFC at all

The responsibility matrix in
[`docs/ecosystem/boundary.md`](../ecosystem/boundary.md) is **Tier 0**: authored
here, vendored verbatim downstream, and amendable only by an RFC in `fdos`
followed by an ADR in both repositories. That procedure exists so a Tier-0 row
cannot be improved by whoever happens to be editing.

Two rows shipped at `ecosystem/v0.1.0` with **known** defects, listed under
*Corrections pending* rather than silently fixed, precisely because the
procedure forbids fixing them in place. This is that procedure.

## Row 1 — "Contracts (proto, schemas, generated SDKs) — `fdos`"

### What is wrong with it

Read literally, the row assigns *every* protobuf schema in the ecosystem to
`fdos`. `fdos-connectors` publishes `libs/plugin-api`, carrying
`fdosconn.plugin.v1` — the wire contract between its plugin host and a plugin.
The row forbids it.

Practice and the boundary tests disagree with the row. Both sides reached that
conclusion independently:

- This repository, drafting the corpus, wrote that the row "is broader than
  practice, and practice looks right… it should be narrowed to *canonical*
  contracts."
- `fdos-connectors` filed [fdos#25](https://github.com/FabioCaffarello/fdos/issues/25)
  with the four tests applied to its schema, and proposed the same narrowing.

Two independent analyses converging on the same wording is the strongest signal
available in a two-repository ecosystem where neither side can see the other
work.

### The evidence, restated

Applying the four boundary tests to `fdosconn.plugin.v1`:

| Test | Result |
|---|---|
| **HTML** — provider changes markup | Only `fdos-connectors` changes. The schema carries `Artifact`, `Fragments`, `ParseResult` — acquisition shapes with no canonical meaning |
| **Tax** — corporate-action treatment changes | Only `fdos` changes. The schema *cannot express* a corporate action: its `Capability` enum has one value, and adding one requires `fdos` to publish a payload first |
| **Offline** — is `fdos` developable with it deleted | Yes, entirely. It imports nothing from `fdos.*`, nothing in `fdos` imports it, and a claim payload crosses as `google.protobuf.Any` — a type URL and bytes |
| **Second provider** | Survives. `Fragments` and `ParseResult` carry opaque bytes specifically so no provider's structure enters the contract |

It also **defines no financial concept**. Every payload it transports is one
`fdos` published, and the consumer's SDK looks the type up in a registry of
published `fdos.ledger.payload.v1` messages before assembling anything.

### The proposal

> | **Canonical** contracts (proto, schemas, generated SDKs) | `fdos` | Single source, one-way flow (I2) |

### The operational test, because "canonical" must not be a matter of taste

A contract is **canonical** when it *defines or constrains the meaning of a
financial fact*. Transporting a fact whose type and shape another repository
defined is not defining it.

Two mechanical consequences, both already true:

- No package outside `fdos` may sit under `fdos.*`. `fdos` publishes only
  `fdos.*`; `fdos-connectors` enforces `fdosconn.*` in its own `proto-check`
  (`fdos-connectors:ADR-0019`). An import path is the most-read claim of
  ownership there is.
- A non-canonical contract may carry canonical payloads but may not declare
  them. `Any` plus a registry of published messages is the permitted shape; a
  locally-defined message describing a holding, a trade or an instrument is not.

I2 is untouched by this. Nothing in `fdos-connectors` defines or redefines a
canonical type, and the reverse edge stays closed.

## Row 2 — "Python toolchain and workspace — `fdos-connectors` — Python exists only there"

### What is wrong with it

The stated fact is false. `fdos-connectors` is a Go workspace: a `go.work` with
four Go modules and no Python anywhere in the ecosystem. The row was inherited
from a charter written before either repository existed.

The *rule* underneath it is sound — a language toolchain present in only one
repository is owned by that repository — but a Tier-0 row asserting a
counterfactual teaches every reader something false about the ecosystem, and
Tier 0 is the tier nobody is allowed to correct downstream.

### The proposal

> | Language toolchains beyond Go | the repository that uses one | A toolchain present in one repository only is owned there. Today both repositories are Go |

This keeps the rule, drops the false claim, and stops naming a language that
would have to be re-litigated if the ecosystem ever adds a different one.

## Consequences

- The corpus becomes `ecosystem/v0.2.0`. `fdos-connectors` re-pins and re-syncs;
  under its vendoring ADR that is a reviewed act rather than an automatic one.
- The narrow, expiring exception `fdos-connectors` recorded for these two rows
  can retire. Retiring it is theirs to do — this RFC does not close their ADR.
- `libs/plugin-api` becomes explicitly legitimate rather than tolerated. It was
  never in doubt on the evidence; it was in doubt against the text.
- **D5's first half closes. D1, D2 and D3 stay open**, and this RFC touches none
  of them.

## Alternatives considered

**Leave both rows and rely on the exception.** Rejected. An exception in the
consumer's decision log against a row in the producer's Tier-0 corpus means the
authoritative text stays wrong and the correction lives downstream — the exact
inversion Tier 0 exists to prevent. It also expires by its own terms, so
declining to act would strand it.

**Narrow row 1 by naming `fdosconn.plugin.v1` as permitted.** Rejected: it
settles one schema rather than a boundary, and the next non-canonical contract
reopens the question. The test belongs in the rule.

**Move `plugin-api` into `fdos`.** Rejected, and it is the option the
unamended row demands. It fails the offline test in the direction that matters:
`fdos` would carry a schema describing plugin-host mechanics it has no use for,
whose shape is driven entirely by a runtime it does not own. The HTML test is
near-miss too — the schema would not change for a markup change, but it would
change for a *runtime* change, and the runtime is theirs.

**Delete row 2 rather than rewrite it.** Rejected narrowly. The rule is real and
will matter the first time either repository adds a non-Go toolchain; deleting
it would leave that unowned.
