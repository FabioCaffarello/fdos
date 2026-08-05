---
id: RFC-0002
title: Money, quantity and numeric representation
status: Proposed
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0002 — Money, quantity and numeric representation

## Summary

Proposes how FDOS represents monetary amounts, quantities and rates, and how
arithmetic on them behaves — in particular division, rounding and currency
mixing.

## Motivation

Binary floating point cannot represent `0.1`. Every financial system that uses
it accumulates error, and the error is not random: it is systematic, it
correlates with transaction volume, and it surfaces as reconciliation breaks
years after the code was written.

Worse for FDOS specifically: `float64` arithmetic is **order-dependent**.
`(a+b)+c ≠ a+(b+c)`. A projection that folds events in a different but equally
valid order produces a different number. That directly violates Constitution §9
— the same ledger must produce a byte-identical report.

The three failure modes this RFC exists to prevent:

1. **Representation error** — `0.1 + 0.2 ≠ 0.3`.
2. **Non-associativity** — reproducibility depends on fold order.
3. **Silent currency mixing** — adding USD to EUR and getting a number.

None is retrofittable. Changing the numeric type after events exist means every
stored amount must be re-interpreted, and the original precision is gone.

## Design

### Amounts are arbitrary-precision decimals

Proposal: `github.com/cockroachdb/apd/v3`, implementing IEEE 754-2008 decimal
arithmetic with explicit contexts.

Chosen over the alternatives because it makes **rounding and precision explicit
parameters of every inexact operation**, and signals conditions such as
`Inexact` and `Rounded` rather than discarding them. Both properties are
requirements here, not conveniences.

### Money is not a number

```
Money := { amount: Decimal, currency: CurrencyCode }
```

Addition and subtraction of `Money` with differing currencies is a **compile-time
error**, not a runtime one: the currency is part of the type's contract and
mixing is unrepresentable, not merely rejected.

Conversion is never implicit. It is an explicit operation taking an FX rate
identified by reference-dataset version (RFC-0004):

```
Convert(m Money, to CurrencyCode, rate FXRateRef, ctx RoundingContext)
    -> Explained[Money]         # RFC-0006
```

The result carries which rate, from which dataset version, at which rounding —
because a converted amount that cannot say how it was converted is not
auditable.

### Quantity is a distinct type

Quantities are not money and must not be interchangeable with it. A share count,
a bond face value and a commodity weight have different unit semantics, and
`Money × Quantity → Money` while `Money × Money` is meaningless.

```
Quantity := { amount: Decimal, unit: UnitCode }
```

Units are canonical (RFC-0005 event taxonomy governs their vocabulary). Fractional
quantities are permitted — fractional shares exist — so a quantity is never an
integer type.

### Division is the dangerous operation

Addition, subtraction and multiplication of decimals are exact. **Division is
not closed**: `1/3` has no terminating decimal representation.

Therefore every division requires an explicit `RoundingContext`:

```
RoundingContext := { precision: uint32, mode: RoundingMode }
```

There is **no default rounding context**. A division without one does not
compile. This is deliberate friction: the majority of financial calculation bugs
live in an unexamined default rounding decision, and defaults are exactly how
that decision goes unexamined.

`RoundingMode` includes at minimum `HalfEven` (banker's), `HalfUp`, `Down`,
`Up`, `Ceiling`, `Floor`. No mode is privileged — the correct choice is
jurisdiction- and instrument-specific, and picking one as "the" default would
encode a legal assumption in a library.

### Rounding is a recorded fact

Because `Inexact` is signalled rather than swallowed, a calculation knows when it
lost precision. That loss is part of the computation trace (RFC-0006), not an
invisible side effect.

This matters for auditability: "why does this total differ from the sum of its
parts by 0.01" must have an answer, and the answer is a rounding record.

### What is forbidden in domain packages

| Forbidden | Reason |
|-----------|--------|
| `float64`, `float32` | Representation error, non-associativity |
| `int` / `int64` as money | Loses currency and scale; overflows on high-precision assets |
| Implicit currency conversion | Hides a reference-data dependency |
| Division without a rounding context | Hides the most consequential decision in the calculation |
| Comparing `Money` of different currencies | Meaningless; a type error |

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| No binary floating point in domain | 2 | `go/analysis` pass banning `float32`/`float64` under domain packages (M2) |
| No cross-currency arithmetic | 1 | Currency is part of the operation's type contract |
| No implicit conversion | 1 | `Convert` requires an explicit rate reference |
| No unrounded division | 1 | Division signature requires `RoundingContext` |
| Fold order does not change results | 3 | Property test: shuffled fold order yields identical output |

The float ban is the single highest-leverage rule in FDOS. It converts
"calculations are reproducible" from an aspiration into a build error.

## Alternatives

**Integer minor units (`int64` cents).** Fast, exact, no dependency, and widely
used. Rejected: the scale is implicit and therefore easy to get wrong across
module boundaries; it cannot represent instruments needing more than a currency's
natural precision (8-decimal crypto, fractional-basis-point yields); and `int64`
overflows for high-notional amounts in low-denomination currencies. It also
still requires an explicit rounding decision on division, so it does not avoid
the hard part.

**`shopspring/decimal`.** More popular and more ergonomic. Rejected: rounding is
attached to individual method calls rather than to an explicit context, and
inexactness is not signalled. Both are exactly the properties this RFC needs.

**Rational numbers (`math/big.Rat`).** Exact under division — genuinely
attractive, since it eliminates rounding entirely during intermediate
computation. Rejected: denominators grow without bound over long fold chains,
making performance unpredictable; and money must eventually be rounded to be
paid, so rounding is deferred rather than avoided. Worth revisiting for
intermediate results in specific calculations.

**A custom decimal type.** Rejected: this is a solved problem, and getting
IEEE 754-2008 semantics right is a multi-year effort with catastrophic failure
modes.

## Prior art

The IEEE 754-2008 decimal formats exist because financial computing repeatedly
established that binary floating point is unfit for money. COBOL's `PACKED
DECIMAL` and SQL's `NUMERIC(p,s)` encode the same conclusion. The novel part
here is not the decimal type — it is refusing a default rounding context.

## Open questions

- Should `Money` carry a scale constraint per currency (JPY 0 decimals, USD 2),
  enforced at construction? It catches real errors but conflicts with
  intermediate values that legitimately need more precision.
- Does a `Rate` type distinct from `Decimal` earn its keep (interest rates,
  FX rates, ratios), or is that type proliferation?
- What is the maximum precision FDOS supports, and is it a global constant or
  per-context?

## Consequences

**Easier:** proving a report is reproducible; auditing where a cent went;
supporting assets with unusual precision.

**Harder:** every division becomes three arguments instead of one. Arithmetic is
verbose. This is the intended cost — the verbosity is where the thinking
happens.

**Impossible:** adding dollars to euros; getting a different total by summing in
a different order.
