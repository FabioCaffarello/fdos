---
directory: examples
purpose: Executable examples demonstrating FDOS contracts and SDK usage.
owner: "@FabioCaffarello"
allowed:
  - Runnable programs demonstrating public contracts and SDKs
  - Synthetic, obviously-fictitious financial data
  - Reference plugin implementations for the Plugin SDK
forbidden:
  - Real financial data, account identifiers or institution credentials
  - Code that does not compile or run
  - Examples excluded from CI verification
  - Behaviour not achievable through the public contract surface
---

# examples

Executable demonstrations of the FDOS public surface.

## `ingest/` — the conformance kit

The first occupant, and the deliverable `E9` was waiting for: a producer outside
FDOS now has a worked example, a way to check its own output, and fixtures to
compare bytes against.

| File | What it is |
|---|---|
| `producer.go` | A worked producer. Imports **only** `libs/contracts` — a producer that imports `libs/kernel` or `libs/ledger` is depending on internals that carry no compatibility promise (ADR-0025) |
| `conform.go` | `Check` — reports whether a serialized submission would be admitted |
| `conform_test.go` | Every way a submission is refused, with the reason |
| `testdata/` | The conforming submission, serialized and as text, so a producer in another language can compare bytes |

### The kit restates no rules

`Check` does not describe admission — it **runs** it, against a throwaway
in-memory ledger. A kit that re-implemented the rules would drift from them, and
the drift has a direction: the kit passes what admission rejects, and a producer
learns the truth from a rejection in production.

Because the rules are shared rather than copied, they cannot disagree. Disabling
the source grammar in `libs/kernel` turns the kit's refusal tests red, which is
the test that the sharing is real.

### It is not permission

The ledger revalidates every submission it receives, assuming nothing about what
the caller ran — a producer can link a modified build of anything published here
(ADR-0029). **Passing the kit is evidence about your submission, never a
commitment from the ledger.**

## Examples are tests

Every example must compile and run in CI. An example that has drifted out of
date is worse than no example: it teaches an API that no longer exists and
consumes the trust of the first person who tries it.

This makes examples a second, independent check on the contract surface. If an
example becomes awkward to write, the contract is awkward to use, and that is
information worth having before external consumers discover it.

## Public surface only

Examples may only use what an external consumer could use. Reaching into
internal packages to make an example work is a signal that the public contract
is incomplete — the fix belongs in the contract, not in the example.

This constraint is what makes `examples/` an honest test of Constitution §11.

## Data

All data is synthetic and unmistakably fictitious. No real institution,
identifier or holding appears here, including in comments and fixture filenames.
