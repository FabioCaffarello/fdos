---
directory: kernel
purpose: The canonical primitives every bounded context shares — identity, money, temporal coordinates, provenance and Explained[T].
owner: "@FabioCaffarello"
allowed:
  - Types RFC-0001, RFC-0002, RFC-0004 and RFC-0006 defined as universal
  - Constructors that make an invalid value unrepresentable
  - Pure functions and combinators over those types
forbidden:
  - Any import outside the kernel itself, standard library aside (ADR-0013)
  - Concepts a single bounded context could own
  - Binary floating point, wall clocks, I/O and concurrency (ADR-0021)
  - Business rules — those belong to the context that owns them
  - Persistence, wire formats or serialisation tags
---

# libs/kernel

The primitives that would otherwise be defined twice. Every bounded context
imports this module; it imports none of them, and nothing outside the standard
library (ADR-0013).

| Package | Holds |
|---------|-------|
| `identity` | `EntityId`, the claims that assert identity without asserting an entity, and the versioned per-scheme canonicalisation applied to a claim before derivation |
| `money` | Arbitrary-precision amounts, quantities, and explicit rounding contexts |
| `temporal` | Bitemporal coordinates — effective time and knowledge time |
| `provenance` | The envelope every fact carries: source, interpreter, confidence, derivation |
| `explained` | `Explained[T]` — a value that arrives with the trace that produced it |

## Why it is deliberately small

Anything one context could own is not kernel (ADR-0013). A shared kernel that
grows becomes a second place where domain language lives, and every context then
couples to every other through it — the coupling is invisible because it runs
through a module everyone already imports.

The bar for adding a package here is an RFC establishing that the concept is
universal, not that two contexts happen to want it today.

## What the analysers enforce, and why here specifically

`libs/kernel` is inside the purity perimeter: `nofloat`, `nondet` and `impurity`
treat `kernel` exactly as they treat a `domain` package (ADR-0021).

That was not always true. The pattern was `(^|/)domain(/|$)`, which
`libs/kernel/money` does not match — so the one package where binary floating
point matters most went entirely unchecked for four milestones, and a `float64`
amount would have been accepted without a word. Recorded rather than quietly
fixed, because the gap is more instructive than the fix.

Test files and generated files are exempt. The property that justifies the whole
decimal design — that folding the same amounts in a different order gives the
same total — can only be tested by shuffling, and shuffling needs `math/rand`.
A rule that makes its own justification untestable is wrong.

## Constructors, not fields

A kernel type is constructed or it does not exist. `identity.NewClaim` refuses a
non-canonical scheme so `"Ticker"` cannot enter alongside `"ticker"`;
`explained.FromDerivation` carries the parameters a trace would otherwise lose.

This is rung 1 and it is the reason the module exists in this shape. A struct
with exported fields and a validation function is a struct that will be built
without validating, in the one code path nobody tested.

## Adding a type

An RFC establishing the concept is universal, the ADR recording it, then the
type — and a codec plus round-trip conformance in `libs/kernel-wire`, because a
canonical concept with one definition on the wire and another in Go will
diverge, and B-003 is the record of what that cost.
