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

## Status: empty by design

`examples/` contains nothing at M0. There is no public contract surface to
demonstrate until M4.

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
