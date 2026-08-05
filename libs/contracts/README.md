---
directory: contracts
purpose: The published contract surface — protobuf schemas and the Go code generated from them.
owner: "@FabioCaffarello"
allowed:
  - Protobuf schemas under proto/
  - Generated Go under gen/, committed and drift-checked
  - Schema documentation as proto comments
forbidden:
  - Hand-edited generated code
  - Business logic or financial calculations
  - Domain types — generated wire types are not canonical models (ADR-0017)
  - Any message reachable from a Fact that carries model output
  - Dependencies beyond the protobuf runtime
---

# libs/contracts

The wire surface private repositories consume (ADR-0004, ADR-0017).

```sh
make proto-check   # lint, format, breaking, pinning, drift, FDOS schema rules
make proto-gen     # regenerate after changing a schema
```

## These are not domain types

The generated Go looks like this:

```go
type Money struct {
    state  protoimpl.MessageState  `protogen:"open.v1"`
    Amount *Decimal                `protobuf:"bytes,1,opt,..." json:"amount,omitempty"`
}
import ( "sync"; "unsafe"; ... )
```

Serialisation tags, `sync`, `unsafe`, mutable embedded state. The `impurity`
analyser rejects every one of those in a domain package, and it is right to: a
canonical model carrying a `json:` tag has had its wire format decided for it,
which is the provider leakage Constitution §3 forbids.

The Go kernel types — `Money` with arithmetic, `Explained[T]` with combinators —
are a separate thing arriving at **M6**. Mapping between the two lives in
adapters.

**This costs a drift risk that is not yet closed.** Two definitions of the same
concept diverge unless something proves they do not. The obligation is a
round-trip conformance test at M6. Until it exists, the gap is real.

## Layout

| Path | Contents |
|------|----------|
| `proto/fdos/kernel/v1/` | Cross-domain primitives: identity, money, temporal, provenance, derivation |
| `proto/fdos/ledger/v1/` | The fact envelope, fact kinds, corrections |
| `gen/` | Generated Go. Committed, never hand-edited |

## Schema decisions worth knowing before editing

**`Decimal` is canonical text.** Not a `double` — floating-point addition is not
associative, so a fold in a different order yields a different total, which
breaks Constitution §9 independently of precision. Not a scaled integer — it
overflows for high-notional amounts in low-denomination currencies. Trailing
zeros are significant.

**`Confidence` is an ordinal enum, never a number.** A numeric confidence
implies a calibration FDOS does not have, and invites arithmetic nobody can
defend.

**`scheme` and `unit` are open strings; the other vocabularies are closed
enums.** An enum for identifier schemes would make every new institution a
private connector supports into a *public* contract release, putting the public
core on the critical path of private work. `Confidence`, `RoundingMode`,
`FactKind` and `CorrectionKind` stay closed: those vocabularies are decisions,
and a new value should require review.

**`ModelOutput` is unreachable from the ledger.** No `Fact` field has that type,
and `make proto-check` fails if one ever does. A model may render a
`DerivationRef` into prose; it may not produce one (Constitution §2).

## Changing a schema

1. Edit the `.proto`.
2. `make proto-gen`, and review the generated diff.
3. `make proto-check` — it will refuse a wire- or JSON-incompatible change
   against `main`.

Additive changes only within a major version. Field numbers are never reused;
removed fields are reserved permanently. Anything else is a new major version,
and both versions remain readable forever because both exist in the ledger
forever (ADR-0011).

## What proto3 cannot do

There is no `required`. The schema cannot make an envelope-less fact
unrepresentable, so `make proto-check` enforces it one rung lower: every ledger
message must carry an `Envelope`.

The rung-1 guarantee for provenance and bitemporality — that omitting them is
*inexpressible* — belongs to the Go kernel constructors at M6. Constitution §15
records this honestly rather than claiming M4 delivered it.
