---
type: doc
name: testing-strategy
description: What is tested today, and the strategy for when code exists
category: testing
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
---

# Testing Strategy

## Today

The only Go code is the toolchain in `libs/analysis`. It is tested with
`analysistest` fixtures, and the discipline applied there is the one that
carries forward to domain code.

### An analyser needs a compliant fixture, not just a violating one

Every rule has fixtures under `testdata/src/ctx/domain/` (must fire) **and**
`testdata/src/ctx/adapters/` (must stay silent).

The second is not symmetry for its own sake. An analyser that fires on
legitimate code gets switched off, and a switched-off rule enforces nothing. The
violating fixture tests sensitivity; the compliant one tests specificity, and
specificity decides whether the rule survives.

This found a real defect: the remedy for "do not range over a map" is itself a
map range —

```go
for k := range m { keys = append(keys, k) }
sort.Strings(keys)
```

— and the first version of `nondet` reported it, which would have made the rule
impossible to satisfy without a suppression comment. Only the compliant fixture
exposed that.

**The fitness functions in `scripts/` are tested against negative cases.** Each
of `toolchain-check`, `contracts-check`, `adr-check` and `constitution-check`
has had its invariant deliberately broken to confirm it fails with a useful
message.

This is not optional rigour. Two of the M0 checks had real defects that only
negative testing surfaced — one silently double-counted, one had a fragile
`set -e` interaction. Both passed the happy path.

> **A check that has never gone red is unverified.** Treat a green check with no
> negative test as an untested claim.

## When code exists (M2 onwards)

### Property-based testing for the domain

The domain is a pure functional core: same inputs, same outputs, no I/O. That is
precisely the shape property-based testing is strongest against.
`pgregory.net/rapid` is the intended harness.

Properties matter more than examples here. "This calculation is associative",
"this projection is invariant under event reordering within a timestamp", "a
correction event never mutates prior state" — these catch classes of bug that
example tests never reach.

### Golden files for reproducibility

Constitution §9 requires reports to be reproducible years later. The mechanism
is golden-file comparison: a fixed ledger plus fixed versioned reference data
must produce byte-identical output, checked in CI.

A golden file that changes without an accompanying explanation is a
reproducibility violation, not a test update.

### Reproducible-build gate

CI builds twice and diffs the binaries. Reproducibility stops being an
aspiration the day it is a red check.

### Mutation testing

`gremlins`, gated on domain packages only. Coverage says which lines ran;
mutation testing says whether the tests would notice if those lines were wrong.
For financial calculations, only the second question matters.

Likely a nightly job rather than per-PR — it is slow, and the Go tooling is
thinner than the JVM equivalent. That trade-off is unresolved.

### Deferred from M2, deliberately

The M2 plan listed a property-based testing harness and mutation testing. Both
were **deferred to M6** rather than built: there is no domain code to exercise,
and a shared test harness written against no caller is scaffold — exactly the
"playbook with no subject" problem that pruned `.context/` at M1.

`make analyze` was the acceptance criterion for M2, and it is met. Stating the
deferral here rather than quietly dropping it is the point.

### Examples are tests

Every example in `examples/` must compile and run in CI (M4). A stale example is
worse than none: it teaches an API that no longer exists and spends the trust of
the first person who tries it.

## What is not tested

Stated so nobody assumes otherwise:

- ADR immutability is enforced by review, not tooling. A git-history check is an
  M3 candidate.
- `GOWORK=off` has no enforcement until the M3 CI workflow exists. Until then
  ADR-0004 can degrade silently.
- Whether `.context/` has drifted from `docs/` is currently unchecked. The
  staleness check is an M2.5 deliverable.

Each of these sits at rung 6 and is recorded as such in Constitution §15.
