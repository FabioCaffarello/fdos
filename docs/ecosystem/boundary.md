# The ecosystem boundary

Which repository owns which concern, how to decide a case the table does not
cover, and what is still genuinely undecided.

**Tier 0.** The responsibility matrix and the four boundary tests are authored
here and vendored verbatim by every consuming repository. Amend by RFC in `fdos`
plus an ADR in both. Ratified by
[ADR-0023](../adr/0023-ecosystem-boundary-and-one-way-contract-flow.md).

---

<!-- BEGIN TIER-0: responsibility matrix — do not edit; amend by RFC + ADR in both repositories -->

## Responsibility matrix

| Concern | Owner | Why the line falls here |
|---|---|---|
| Canonical financial model | `fdos` | Semantics of money must have exactly one definition |
| Ledger, posting rules, double-entry | `fdos` | Truth path (Constitution §2) |
| Kernel and bounded contexts | `fdos` | Domain core |
| **Canonical** contracts (proto, schemas, generated SDKs) | `fdos` | Single source, one-way flow (E2). Canonical means it defines or constrains the meaning of a financial fact; transporting one does not (ADR-0026) |
| Instrument identity resolution | `fdos` | Cross-provider concern; no single provider can decide it |
| Corporate actions | `fdos` | Domain rules, not provider quirks |
| Risk | `fdos` | Reads the canonical model |
| Knowledge graph | `fdos` | Derived from canonical entities |
| MCP surface | `fdos` | Exposes canonical model, not raw acquisitions |
| Engineering platform (make, mise, CI, supply chain) | `fdos` | Origin of the shared standard |
| Currency, rounding, precision policy | `fdos` | Ledger correctness |
| Provider plugins, one per provider | private repositories | Provider-shaped, provider-lifetime |
| Provider SDK | `fdos-connectors` | Serves plugin authors only |
| Plugin runtime | `fdos-connectors` | Execution of acquisition, not of domain logic |
| Browser runtime and browser sessions | `fdos-connectors` | Acquisition mechanics |
| Provider authentication, credentials, MFA, session lifetime | `fdos-connectors` | Credentials are provider-scoped (see D2) |
| Extractors, parsers, normalizers | `fdos-connectors` | Bounded by §"Where normalisation stops" — they normalise *shape*, never *meaning* |
| Acquisition pipeline, scheduling, retries, backoff | `fdos-connectors` | Provider-facing operational concern |
| Raw artifact storage and replay | `fdos-connectors` | Provenance producer (E4) |
| Language toolchains beyond Go | the repository that uses one | A toolchain present in one repository only is owned there. Today both repositories are Go (ADR-0026) |

## The four boundary tests

Apply these before writing anything near the line. They resolve most disputes
without escalation.

1. **The HTML test.** A provider changes its markup tomorrow. Which repository
   changes? Only `fdos-connectors`. If your change would also be needed, the
   knowledge has leaked in the wrong direction.
2. **The tax test.** Brazilian corporate-action treatment changes. Which
   repository changes? Only `fdos`. If a connector would need editing, domain
   semantics have leaked into acquisition.
3. **The offline test.** Can `fdos` be developed, built, tested and reasoned
   about with every provider unreachable and `fdos-connectors` deleted? It must
   be. Yes, always.
4. **The second-provider test.** Would this abstraction survive a second
   provider expressing the same fact differently? If not, it is provider-shaped
   and does not belong in `fdos`.

<!-- END TIER-0 -->

---

## Where normalisation stops

The single line most likely to erode. It erodes by reasonable-sounding
increments, each of which looks like a convenience.

**Normalisation of shape** — permitted downstream: decoding character sets,
parsing dates into timestamps with an explicit timezone, converting `"1.234,56"`
into a decimal with declared scale, splitting a PDF table into rows, naming
fields consistently, discarding layout.

**Normalisation of meaning** — forbidden downstream: deciding that two provider
identifiers refer to the same instrument, classifying a row as a dividend versus
a return of capital, computing cost basis, netting, currency conversion,
inferring a missing field from a business rule, deduplicating across providers,
correcting what looks like a provider error.

The last deserves emphasis. **A connector never corrects its provider.** It
reports faithfully, including contradictions, and lets `fdos` decide. A connector
that silently fixes data destroys the audit trail that makes the ledger
defensible — and it does so invisibly, which is the part that matters.

**`fdos` never learns:** HTTP, cookies, browsers, captchas, OTP, provider rate
limits, provider quirks, retry policy against a provider, or the name of any
specific provider anywhere in the domain layer. Provider identity enters `fdos`
only as opaque provenance metadata — today, `fdos.kernel.v1.SourceRef`.

## Amendments to the charter

The matrix above diverges from the founding charter in two rows, deliberately
and by the Tier-0 procedure —
[RFC-0008](../rfc/0008-narrowing-two-responsibility-matrix-rows.md) then
[ADR-0026](../adr/0026-canonical-contracts-and-language-toolchains.md).

- **"Contracts" → "Canonical contracts."** Read literally the original forbade
  `fdos-connectors` from publishing the wire contract between its own plugin
  host and its plugins, which the four tests plainly assign to it. Both
  repositories identified this independently before either had seen the other's
  reasoning.
- **"Python toolchain and workspace" → "Language toolchains beyond Go."** The
  original asserted that Python exists in `fdos-connectors`. It does not; that
  repository is a Go workspace with four Go modules. The rule underneath was
  sound and is kept without naming a language.

Both rows shipped knowingly wrong at `ecosystem/v0.1.0` and were listed as
defects rather than fixed in place, because Tier 0 forbids fixing a vendored row
by editing it. They were corrected because they were written down as errors.

## Disputed items

Ambiguities that are **not settled by whoever writes code first**. Each needs an
ADR in both repositories before either implements against it.

### D1 — Browser runtime provenance

`fdos-connectors` owns browser sessions, but a Browser-as-a-Service platform
already exists as a separate product (`synbas`). Build, vendor, or consume as an
external service? This decides whether anti-detect concerns sit inside the
ecosystem boundary or outside it.

**Status:** open. RFC in `fdos-connectors`, ADR in both.

### D2 — "Authentication" is two concerns wearing one word

Provider authentication — credentials against any external provider — is the
private side's. Platform identity — who may write a fact to a stream, who may
query the ledger, who may call the MCP surface — is `fdos`. Split explicitly
before either is built.

**Status:** open, registered as
[fdos#64](https://github.com/FabioCaffarello/fdos/issues/64).

**This entry previously said the `fdos` half had "no subject yet"**, on the
grounds that there was no query surface and no MCP server. That reasoning was
overtaken rather than wrong: the subject arrived through the **write** side,
which the entry was not watching.
[ADR-0030](../adr/0030-the-submission-shape.md) published a submission message
carrying a producer-supplied `stream` name and recorded, as a cost, that nothing
validates who may write to a named stream — assigning that question here. Its
enforcement table carries the row at *none — D2 is open*.

The correction is kept rather than overwritten because the failure mode is worth
seeing: a disputed item was assessed as not-yet-urgent against one surface, and
became urgent through a different one, in a decision that named D2 explicitly
while doing it.

### D3 — Where normalisation stops

The charter grants "normalizers" to `fdos-connectors`; the section above is the
reading that keeps E1 intact. Ratify or revise it by ADR, but do not leave it
implicit.

**Status:** open, and closer to ratifiable than the others. `fdos-connectors`
has independently implemented a position consistent with the section above:
extraction and parse outputs travel as opaque bytes, emptiness must be asserted
by the provider rather than inferred from absent rows, and a parse that cannot
be trusted becomes a published *rejection* rather than a corrected value. That is
"reports faithfully, never corrects" built into a type. What is missing is the
ADR, not the behaviour.

### D4 — What a `SourceRef` must resolve to — CLOSED

**Settled by [ADR-0028](../adr/0028-provenance-admissibility.md)**, via
[RFC-0011](../rfc/0011-provenance-admissibility.md).

A `SourceRef` is an algorithm-prefixed content hash with an **unspecified
referent**. FDOS never dereferences it, so Constitution §13 and the offline test
hold exactly as before; what changed is that the *form* is known, which opacity
never required it not to be.

The distinction that resolves it: **opaque is about resolution, unconstrained is
about form.** ADR-0010 conflated them because at the time there was no producer
to constrain.

The field is renamed `value` → `content_hash` in the same migration window, on
the reasoning that a form check catches accident rather than intent — which
leaves the *identifier* carrying most of the enforcement weight.

**Status: closed.** D1, D2 and D3 remain open.

### D5 — Which contracts are "the contract surface" — CLOSED

**Both halves are now settled.**

The FDOS-owned half — whether `libs/kernel`, `libs/ledger` and the `-wire`
modules are offered to consumers — is decided by
[ADR-0025](../adr/0025-consumer-facing-surface-is-the-contracts-module.md): they
are published as a consequence of ADR-0004, carry no compatibility promise, and
`libs/contracts` is the only offered surface.

The disputed half — whether a proto contract that is *not* canonical may be
defined outside `fdos` — is decided by
[ADR-0026](../adr/0026-canonical-contracts-and-language-toolchains.md) via
[RFC-0008](../rfc/0008-narrowing-two-responsibility-matrix-rows.md). It may, and
the matrix row above now says so. Evidence and the four tests applied to the private side's host-plugin schema
are in [fdos#25](https://github.com/FabioCaffarello/fdos/issues/25).

**Status: closed at `ecosystem/v0.2.0`.** D1, D2 and D3 remain open; D4 remains
M8's gating deliverable.
