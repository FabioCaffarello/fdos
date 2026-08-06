---
directory: ledger-wire
purpose: Maps the FDOS ledger and ingest types to and from protobuf, with the conformance suite that keeps the two definitions honest.
owner: "@FabioCaffarello"
allowed:
  - Encode and Decode functions for ledger facts, envelopes and payloads
  - Encode and Decode for ingest submissions, which are pre-admission shapes
  - Round-trip conformance tests
  - The protobuf runtime, the published contracts, kernel and kernel-wire modules
forbidden:
  - Business rules or financial calculations
  - Decoding that assigns fields instead of going through domain constructors
  - Trusting a declared fact type without checking the payload
  - Silent defaults for a wire value with no domain equivalent
---

# libs/ledger-wire

Closes **B-003** for the ledger types, alongside `libs/kernel-wire` for the
kernel. Together they pay the cost ADR-0018 recorded: two definitions of every
canonical concept, with nothing preventing them diverging.

## The two properties

```
domain -> wire -> domain   is the identity   (nothing lost encoding)
wire   -> domain -> wire   is the identity   (nothing dropped decoding)
```

A fact is where both matter most. If any part of the envelope fails to survive,
a fact arrives unable to say when it was true, when FDOS learned it, or where it
came from — and §6 and §7 become claims about the domain only, true right up
until something is written down.

Negative-tested. Dropping reference bindings while encoding, mapping
`KindAccount` to `PARTY`, and skipping the declared-type check each make the
suite fail.

## The declared type is checked, not trusted

A `Fact` carries both a `type` string and an `Any` payload. `DecodeFact`
unpacks the payload and rejects the fact if the two disagree.

A fact claiming to be a `ledger.TradeSettled` while carrying a holding
observation is the shape a corrupted or hostile stream takes, and it must not
reach a projection. The same check applies to `type_version`.

## Facts and payloads are different things

`fdos.ledger.v1` holds facts; `fdos.ledger.payload.v1` holds what a fact
asserts. Every message in the first must carry an `Envelope` and
`make proto-check` enforces it; a payload has none by construction, because the
fact around it carries the envelope for both.

That boundary exists because the envelope rule fired on `HoldingObserved` during
M7 and was right to. The alternative was exempting messages by name, and an
exemption by name is a rule waiting to be worked around.

## Identity codecs are not here

`EncodeEntityID` and `EncodeClaim` live in `libs/kernel-wire`. They were here
until M7, when adding the claim codec forced the question: `identity.ID` and
`identity.Claim` are kernel types and `EntityId` and `IdentifierClaim` are kernel
messages, so nothing about either was ever the ledger's. The original placement
had no technical cause — `EntityId` was in `contracts@v0.1.0`, which
`libs/kernel-wire` already pinned.

Recorded rather than quietly corrected, because the drift survived a milestone
by looking like a reasonable place for the code to be.

## Why a separate module

Go resolves dependencies per module. A codec inside `libs/ledger` would put the
protobuf runtime into the graph of everyone importing `libs/ledger/domain`,
making Constitution §10 true at the package level and false at the level that
decides what a consumer is actually coupled to (ADR-0013).

## Adding a payload type

1. A message in `fdos.ledger.payload.v1`.
2. Cases in `encodePayload` and `decodePayload` — the switches have no
   reflective fallback, so forgetting produces an error at the first encode
   rather than a silently empty `Any`.
3. A round-trip property test covering **both** directions.
4. Break the codec deliberately and confirm the test fails.
