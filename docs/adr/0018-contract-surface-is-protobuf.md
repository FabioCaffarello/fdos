---
id: ADR-0018
title: The contract surface is protobuf, and wire types are never domain types
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0018 — The contract surface is protobuf, and wire types are never domain types

## Context

Constitution §11 requires contracts to be versioned, documented and tested, and
ADR-0004 makes the open-core boundary depend on private repositories consuming
published contract modules. Until now "contract" meant an intention.

M4 makes it an artifact. The decisions below were forced by two things: what the
accepted M1.5 RFCs actually require on a wire, and what the repository's own
analysers will not permit in a domain package.

## Decision

### Protobuf, with `buf breaking` as the mechanism

The published contract surface is protobuf, configured by `buf.yaml`. The
load-bearing part is `buf breaking` against `main` with `WIRE_JSON` and `FILE`:
a contract that can change without anything failing is not a contract.

`WIRE_JSON` rather than `WIRE` because the JSON representation is what any
future OpenAPI surface will expose, and a change that is wire-safe but
JSON-breaking would still break consumers.

### Generated wire types can never be domain types

This is the decision everything else follows from, and it is not a preference —
it is what the generated code turns out to be. From the first generated file:

```go
type Money struct {
    state   protoimpl.MessageState  `protogen:"open.v1"`
    Amount  *Decimal                `protobuf:"bytes,1,opt,name=amount" json:"amount,omitempty"`
    ...
}
import ( "sync"; "unsafe"; protoimpl ... )
```

Serialisation tags, `sync`, `unsafe`, and mutable embedded state. The
`impurity` analyser rejects every one of those in a domain package, and it is
**right to**: a canonical model carrying a `json:` tag has had its wire format
decided by whoever wrote the tag, which is precisely the provider leakage
Constitution §3 forbids.

So `libs/contracts` is the wire surface. The Go kernel types — `Money` with
arithmetic, `Explained[T]` with combinators — are a separate thing arriving at
M6, and mapping between them lives in adapters.

**The drift risk is real and is not yet closed.** Two definitions of the same
concept will diverge unless something proves they do not. The obligation is a
round-trip conformance test at M6: domain → wire → domain must be the identity,
and every wire field must be reachable. Until that exists, this decision has a
gap, and stating it here is better than discovering it in a mapping bug.

### Decimals are canonical text

`Decimal` is a string, not a double and not a scaled integer.

`double` is excluded for the reason that governs all of FDOS numerics: floating
point addition is not associative, so a fold in a different order yields a
different total (ADR-0008). A scaled integer overflows for high-notional
amounts in low-denomination currencies and hides precision behind a second
field consumers ignore. Text round-trips exactly at any precision and stays
legible in a ledger someone may audit in 2031.

Trailing zeros are significant: they record the precision of the value.

### Vocabularies that private repositories extend stay open

`IdentifierAssertion.scheme` and `Quantity.unit` are strings, not enums.

An enum would make every new identifier scheme or instrument class a **public
contract release**, because a private connector supporting a new institution
cannot add an enum value. That would put the public core on the critical path of
private work, which inverts the open-core relationship.

`Confidence`, `RoundingMode`, `FactKind` and `CorrectionKind` stay closed enums:
their vocabularies are decisions, not extension points, and a new value there
*should* require review.

### Generated code is committed, and drift is checked

Private repositories import the contract module through the Go proxy (ADR-0004),
so the generated code must exist in the module the proxy serves.

Committing it also makes drift checkable: `make proto-check` regenerates into a
scratch tree and diffs. The codegen plugin is pinned to an exact version for the
same reason every GitHub Action is pinned to a commit SHA (ADR-0014), and the
check fails if it is ever loosened to a floating tag.

### The truth boundary is a schema rule

`ModelOutput` exists and is deliberately unreachable from the ledger: no `Fact`
field has that type, and `make proto-check` fails the build if one ever does.

A model may render a `DerivationRef` into prose. A model may not produce one.
That is the whole boundary — the explanation describes something that already
exists and is independently verifiable, rather than being the only account of
how a number arose (Constitution §2).

## Consequences

### Positive

- Constitution §11 has a mechanism: incompatible contract changes fail the
  build, against `main` rather than against the previous commit.
- The open-core boundary now has something concrete to be a boundary *of*.
- The §2 truth boundary is a schema property rather than a principle in a
  document.
- Every schema decision that mattered — decimal-as-text, ordinal confidence,
  open vocabularies — is recorded where a future reader will look.

### Negative

- **Two definitions of every canonical concept**, wire and domain, with no
  mechanism yet preventing drift. This is the largest cost of the decision and
  it is unpaid until the M6 conformance test.
- **proto3 cannot express `required`.** The rung-1 claims for §6 (Provenance)
  and §7 (Temporal Modeling) — that a fact without them is unrepresentable —
  are *not* delivered here and cannot be. The schema check enforces that ledger
  messages carry an `Envelope`, which is rung 3. Constitution §15 is corrected
  accordingly rather than left overstating what M4 achieved.
- Committed generated code makes diffs noisy and invites hand-editing. The drift
  check catches the edit; it does not prevent the temptation.
- `buf breaking` compares against `main`, so a long-lived branch can accumulate
  a break that only surfaces at merge. The merge queue (ADR-0014) mitigates
  this; it does not eliminate it.
- Remote codegen plugins require network access at generation time. Generation
  is not on the build path — the output is committed — but a contributor with no
  network cannot regenerate.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| No incompatible contract change | 3 | `buf breaking` against `main` (`make proto-check`) |
| Schemas lint and format clean | 3 | `buf lint`, `buf format -d` |
| Generated code matches the schemas | 3 | regenerate-and-diff |
| Codegen plugin pinned exactly | 3 | `make proto-check` |
| Every ledger fact carries an Envelope | 3 | `make proto-check`, FDOS-specific rule |
| No ledger message references `ModelOutput` | 3 | `make proto-check`, FDOS-specific rule |
| Wire types absent from domain packages | 2 | `impurity` analyser (serialisation tags, `sync`, `unsafe`) |
| Wire and domain definitions agree | 6 | **nothing yet** — M6 conformance test |

All six schema rules were negative-tested.

## Alternatives considered

**Generate the domain model from proto and use it directly.** By far the
simplest: one definition, no drift, no mapping layer. Rejected because the
generated type is structurally disqualified — `json:` tags, `sync`, `unsafe`,
mutable state — and the analysers enforcing Constitution §3 and §10 would have
to be weakened to admit it. Weakening the mechanism to fit the convenience is
the failure this repository is built to avoid.

**Go types as the source, proto generated from them.** Keeps one definition and
puts it in the layer that matters. Rejected: Go-to-proto generators lose the
explicit field numbering that makes `buf breaking` meaningful, and field numbers
are the thing that must never change.

**JSON Schema or Avro instead of protobuf.** Avro has better schema-evolution
semantics on paper. Rejected: `buf breaking` is a more usable gate than
anything in the Avro ecosystem, and protobuf's field-reservation discipline is
exactly the mechanism ADR-0011 depends on for upcast-on-read.

**`google.protobuf.Timestamp` versus a custom instant.** Kept the well-known
type. A custom one would let FDOS state its precision explicitly, which RFC-0003
leaves open — but at the cost of every consumer losing standard tooling.

**Closed enums for `scheme` and `unit`.** Rejected on the open-core argument
above. The cost is that the vocabulary is governed by documentation rather than
by the type, and a typo becomes a silent new scheme.

## Notes

> **Correction, same day.** This ADR was implemented with a *remote* BSR plugin,
> and a clean-clone run failed with `resource_exhausted: too many requests`.
> That made every `make verify` depend on an external, rate-limited service —
> an availability dependency on the build that ADR-0014 exists to prevent, and
> one that fails for reasons unrelated to the change.
>
> `buf.gen.yaml` now uses a local plugin,
> `go run google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.6`: pinned exactly,
> built by the project toolchain, resolved from the module cache after first
> use. Same reasoning as govulncheck in `scripts/verify-vulns.sh`. The generated
> output is byte-identical, so the contract is unchanged.
>
> The Decision below is unaffected — it requires an exactly-pinned plugin and
> never specified remote. The negative consequence about network access is now
> historical, and `make proto-check` fails if the plugin is ever moved back to
> the remote form.

Deferred from M4, deliberately, each for lack of a subject:

- **OpenAPI.** There is no service and no HTTP surface. Generating an API
  specification for endpoints that do not exist would pre-judge the M6
  application layer, which is the mistake M1.5 exists to prevent.
- **SDKs beyond Go.** No consumer in another language exists. The plugin list in
  `buf.gen.yaml` is where they land.
- **MCP tool generation.** Depends on the API surface above, and MCP is a moving
  target — the generator must stay pluggable and versioned separately.
- **Observability conventions.** There is nothing running to observe.
  Conventions defined against no service are scaffold, the same judgement that
  pruned the agent roster at M1 and deferred the property-testing harness at M2.
  They arrive with the Ledger at M6, derived from the envelope fields defined
  here.

Open questions:

- The wire/domain conformance test at M6 is the unpaid cost of this decision.
- Whether contract modules should be attested at release, or whether a Go module
  tag served by the proxy is sufficient (ADR-0014 left this open too).
- `Decimal` canonical form is documented but unvalidated. `protovalidate` could
  enforce it at rung 3; whether that is worth a runtime dependency in the
  contract module is unresolved.
