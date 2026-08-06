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
| Contracts (proto, schemas, generated SDKs) | `fdos` | Single source, one-way flow (I2) |
| Instrument identity resolution | `fdos` | Cross-provider concern; no single provider can decide it |
| Corporate actions | `fdos` | Domain rules, not provider quirks |
| Risk | `fdos` | Reads the canonical model |
| Knowledge graph | `fdos` | Derived from canonical entities |
| MCP surface | `fdos` | Exposes canonical model, not raw acquisitions |
| Engineering platform (make, mise, CI, supply chain) | `fdos` | Origin of the shared standard |
| Currency, rounding, precision policy | `fdos` | Ledger correctness |
| Provider plugins (`btg-bot`, `b3-bot`, …) | `fdos-connectors` | Provider-shaped, provider-lifetime |
| Provider SDK | `fdos-connectors` | Serves plugin authors only |
| Plugin runtime | `fdos-connectors` | Execution of acquisition, not of domain logic |
| Browser runtime and browser sessions | `fdos-connectors` | Acquisition mechanics |
| Provider authentication, credentials, MFA, session lifetime | `fdos-connectors` | Credentials are provider-scoped (see D2) |
| Extractors, parsers, normalizers | `fdos-connectors` | Bounded by §"Where normalisation stops" — they normalise *shape*, never *meaning* |
| Acquisition pipeline, scheduling, retries, backoff | `fdos-connectors` | Provider-facing operational concern |
| Raw artifact storage and replay | `fdos-connectors` | Provenance producer (I4) |
| Python toolchain and workspace | `fdos-connectors` | Python exists only there |

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

## Corrections pending against the charter

Recorded rather than silently absorbed, because the Tier-0 block above must stay
byte-identical to the brief it came from. Each needs the human to reconcile both
copies.

- **The "Python toolchain and workspace" row is counterfactual.** `fdos-connectors`
  is a Go workspace: a `go.work` with four Go modules and no Python anywhere. The
  row's *rule* is sound — a language toolchain that exists in only one repository
  is owned by that repository — but its stated fact is not true of the ecosystem
  that exists. Amend to name the principle rather than a language, or delete it.
- **"Contracts (proto, schemas, generated SDKs) — `fdos`" is broader than
  practice, and practice looks right.** `fdos-connectors` defines a host↔plugin
  wire contract in its own namespace, importing nothing from `fdos.*`. Read
  literally the row forbids that; read against the four tests it is plainly the
  consumer's, because a plugin runtime is theirs and `fdos` never sees those
  messages. The row should be narrowed to *canonical* contracts. See D5.

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

Provider authentication — credentials against BTG, B3, Bacen — is
`fdos-connectors`. Platform identity — who may query the ledger, who may call
the MCP surface — is `fdos`. Split explicitly before either is built.

**Status:** open, and not yet urgent: `fdos` has no query surface and no MCP
server, so the `fdos` half has no subject yet.

### D3 — Where normalisation stops

The charter grants "normalizers" to `fdos-connectors`; the section above is the
reading that keeps I1 intact. Ratify or revise it by ADR, but do not leave it
implicit.

**Status:** open, and closer to ratifiable than the others. `fdos-connectors`
has independently implemented a position consistent with the section above:
extraction and parse outputs travel as opaque bytes, emptiness must be asserted
by the provider rather than inferred from absent rows, and a parse that cannot
be trusted becomes a published *rejection* rather than a corrected value. That is
"reports faithfully, never corrects" built into a type. What is missing is the
ADR, not the behaviour.

### D4 — What a `SourceRef` must resolve to

`fdos.kernel.v1.SourceRef` is `{ string value }`, and `fdos` enforces nothing
about it. [ADR-0010](../adr/0010-provenance-envelope-reference-versioning.md)
left this open in as many words: *"Open, deliberately: whether `SourceRef` is an
opaque reference resolved privately (Open Core, Constitution §13)."*

It is no longer only open — it has been **answered downstream by construction**.
`fdos-connectors` populates it with the content address of its own acquisition
record, and requires an origin to be complete before a fact can be assembled.
Nothing in `fdos` asks for that, checks it, or knows it happened.

Why this matters more than it looks: I4 makes provenance an *admission
criterion* for the ledger — source, method, fetch time, content hash. If what the
hash addresses is specified only downstream, then the ledger's admission rule is
authored by its consumer. That is a reverse edge in substance even though there
is no reverse edge in imports, and it is invisible precisely because the Go
compiler is content.

Two defensible answers, and this document deliberately picks neither:

- **Stay opaque.** Constitution §13 says resolving a source is a private
  concern. FDOS records what it was told and audits the chain by content hash
  without knowing what the hash addresses.
- **Specify the referent.** FDOS states what a `SourceRef` must resolve to and
  what an acquisition record must contain, without knowing how it is stored.
  Consumers satisfy it; FDOS can then say what "admissible provenance" means
  rather than assuming it.

**Status:** open, and now on the critical path. It is the **gating deliverable
of M8** ([`roadmap.md`](roadmap.md)) — the milestone whose subject is accepting
an externally-produced fact, which is the exact moment this stops being free.

M8 cannot honestly start before it. Building the intake path first would
hard-code an answer to a question two repositories are supposed to ratify, and
the Go compiler would report nothing.

An RFC is asked for from `fdos-connectors`
([issue #2](https://github.com/FabioCaffarello/fdos-connectors/issues/2)),
because the implementation experience is there rather than here.

### D5 — Which contracts are "the contract surface"

[ADR-0018](../adr/0018-contract-surface-is-protobuf.md) says the contract
surface is protobuf. The matrix says `fdos` owns "contracts (proto, schemas,
generated SDKs)". Neither says whether a proto contract that is *not* canonical —
a host↔plugin boundary internal to acquisition — is covered.

Practice has answered: `fdos-connectors` owns its own, in its own namespace,
importing nothing from `fdos.*`, and the four tests agree with it. The matrix
does not.

**The second half is settled.**
[ADR-0025](../adr/0025-consumer-facing-surface-is-the-contracts-module.md)
decides that `libs/contracts` is the consumer-facing surface and that
`libs/kernel`, `libs/ledger`, `libs/kernel-wire` and `libs/ledger-wire` are
published as a consequence of ADR-0004 rather than as an offer — no
compatibility promise, and importing one couples a consumer to FDOS's internal
structure instead of its contract.

That half was entirely within FDOS's ownership of its own module topology, so it
did not need both repositories. Settling it while no external code imports those
modules was the cheap moment; afterwards it would have involved somebody else's
migration.

**Status: the first half stays open** — whether a proto contract that is *not*
canonical may be defined outside `fdos`. Practice has answered it and the four
tests agree with practice; the matrix does not. That one needs an ADR in both
repositories.
