---
directory: kernel-wire
purpose: Maps the FDOS kernel types to and from their protobuf wire form, with the conformance test that keeps the two definitions honest.
owner: "@FabioCaffarello"
allowed:
  - Encode and Decode functions for kernel types
  - Round-trip conformance tests
  - The protobuf runtime and the published contracts module
forbidden:
  - Business rules or financial calculations
  - Decoding that assigns fields instead of going through domain constructors
  - Silent defaults for a wire value with no domain equivalent
  - Any dependency the kernel itself does not have, leaking back into it
---

# libs/kernel-wire

Closes **B-003**: ADR-0018 created two definitions of every canonical concept —
the protobuf wire types and the Go domain types — and recorded that nothing
prevented them diverging. This is that mechanism.

## Why a separate module

Go resolves dependencies per module. A codec inside `libs/kernel` would put the
protobuf runtime into the graph of everyone importing `libs/kernel/money`,
making Constitution §10 true at the package level and false at the level that
decides what a consumer is actually coupled to (ADR-0013).

`libs/kernel` depends on one decimal library. It stays that way.

## The two properties

```
domain -> wire -> domain   is the identity   (nothing lost encoding)
wire   -> domain -> wire   is the identity   (nothing dropped decoding)
```

The second is the one that earns its keep. A codec that never reads
`published_at` passes the first property forever, because the value it fails to
carry was never in the domain value it compares against.

Both are property-tested over generated values, and both were **negative-tested**:
deliberately dropping `published_at` and deliberately mapping a rounding mode to
the wrong enum each make the suite fail.

## Decoding is validation

Every decode goes through the domain constructor rather than assigning fields. A
wire message is data from outside the process, and those constructors are the
only thing standing between it and an invalid value — a `Money` with a
three-letter currency that is not uppercase, a `Provenance` with no interpreter,
an interval that ends before it begins.

Enum mappings have no default that silently succeeds. A wire value with no
domain equivalent is an error, because the alternative is a rounding mode that
quietly becomes the wrong one.

## Adding a type

1. `Encode` and `Decode` in `codec.go`.
2. A round-trip property test covering **both** directions.
3. Break the codec deliberately and confirm the test fails.

A codec with only the domain-side property is a codec that will drop a field.
