---
id: ADR-0008
title: Money is arbitrary-precision decimal and no division has a default rounding
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0008 — Money is arbitrary-precision decimal and no division has a default rounding

Records the acceptance of [RFC-0002](../rfc/0002-money-and-numeric-representation.md).

## Context

Binary floating point cannot represent `0.1`, and its arithmetic is
order-dependent: `(a+b)+c ≠ a+(b+c)`. A projection folding events in a
different but equally valid order would produce a different number, violating
Constitution §9 directly. The failure modes — representation error,
non-associativity, silent currency mixing — are systematic, correlate with
volume, and surface as reconciliation breaks years later.

None is retrofittable: changing the numeric type after events exist means
re-interpreting every stored amount, and the original precision is gone.

Serves Constitution §2 (determinism) and §9 (reproducibility).

## Decision

FDOS represents amounts as arbitrary-precision decimals using
`github.com/cockroachdb/apd/v3` (IEEE 754-2008 decimal arithmetic), chosen
because rounding and precision are explicit parameters of every inexact
operation and conditions such as `Inexact` are signalled rather than discarded.

`Money` is `{amount: Decimal, currency: CurrencyCode}`. Cross-currency addition,
subtraction and comparison are unrepresentable — a compile-time error, not a
runtime rejection. Conversion is always explicit, takes an FX rate pinned by
reference-dataset version (ADR-0010) and a rounding context, and returns
`Explained[Money]` (ADR-0012).

`Quantity` is `{amount: Decimal, unit: UnitCode}`, distinct from `Money` and
never an integer type.

Every division requires an explicit `RoundingContext` `{precision, mode}`.
**There is no default rounding context** and no privileged mode — the correct
choice is jurisdiction- and instrument-specific, and a default is exactly how
that decision goes unexamined. Precision loss is signalled and recorded in the
computation trace.

`float32`/`float64`, integer-as-money, implicit conversion and unrounded
division are forbidden in domain packages.

## Consequences

### Positive

- Reproducibility becomes provable: shuffled fold order yields identical output.
- "Why does this total differ from the sum of its parts by 0.01" always has an
  answer — a rounding record.
- Assets with unusual precision (8-decimal crypto, fractional basis points) are
  representable.

### Negative

- Every division takes three arguments. Arithmetic is verbose — deliberately,
  because the verbosity is where the thinking happens — but the friction is
  real and permanent.
- The most critical arithmetic in the system now depends on a third-party
  library. Its version becomes part of what a reproduction must pin
  (ADR-0010's "code is the third leg"); a semantics bug in it is baked into
  recorded traces.

### Enforcement

Today: rung 5 — there is no code. From M2: rung 1 for currency mixing, implicit
conversion and unrounded division (type contracts); rung 2 for the float ban
(`go/analysis` pass over domain packages) — the single highest-leverage rule in
FDOS. From M6: rung 3, property test that shuffled fold order yields identical
output.

## Alternatives considered

- **Integer minor units (`int64` cents)** — implicit scale, overflow on
  high-notional/low-denomination amounts, and it still needs the rounding
  decision, so it avoids nothing hard.
- **`shopspring/decimal`** — rounding per method call, inexactness not
  signalled: precisely the properties needed are the ones missing.
- **Rationals (`big.Rat`)** — exact under division but unbounded denominators;
  rounding is deferred, not avoided. Worth revisiting for intermediates.
- **A custom decimal type** — a solved problem with catastrophic failure modes.

Full exploration in RFC-0002.

## Notes

Open, deliberately: per-currency scale constraints at construction; whether a
distinct `Rate` type earns its keep; maximum supported precision and whether it
is global or per-context. To be settled by the first calculation designs in M6.
