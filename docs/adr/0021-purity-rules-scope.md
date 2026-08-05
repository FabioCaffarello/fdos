---
id: ADR-0021
title: The purity rules cover the kernel and exempt test and generated files
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0021 — The purity rules cover the kernel and exempt test and generated files

## Context

M2 built four analysers and tested them against fixtures. M6 ran them against
real domain code for the first time. Three things were wrong, and none was
visible without a subject.

## Decision

### The kernel is covered

The domain pattern was `(^|/)domain(/|$)`. `libs/kernel/money` does not match
it.

The kernel holds `Money`. The one package where binary floating point matters
most was entirely unchecked, and the analysers would have accepted a `float64`
amount without a word.

The pattern is now `(^|/)(domain|kernel)(/|$)`.

### Test files are exempt

The property that justifies the whole decimal design — that folding the same
amounts in a different order gives the same total — can only be tested by
shuffling, and shuffling needs `math/rand`. `nondet` reported the test that
proves the property it exists to protect.

**A rule that makes its own justification untestable is wrong.** The rules keep
production domain code reproducible; test files are not shipped, are not part of
any derivation, and never reach the ledger.

### Generated files are exempt

`libs/contracts/gen/fdos/kernel/v1` matches the kernel pattern, because the
proto package is `fdos.kernel.v1`. The analysers reported every generated
message for `json` tags and importing `sync`.

They were **right on the substance** — that is precisely ADR-0018's argument
that generated wire types can never be domain types, now demonstrated rather
than asserted. But nobody wrote those files. A report there can only produce a
suppression comment, and suppression comments are how a rule stops being
enforced.

Files carrying the standard `Code generated … DO NOT EDIT.` marker are skipped.
Synthesized test binaries (`<pkg>.test`) are skipped for the same reason.

## Consequences

### Positive

- The kernel is checked, which is where the float ban earns its keep.
- The rules no longer forbid the tests that prove them.
- Generated code cannot force `//nolint` into the repository.
- ADR-0018's claim about wire types is now demonstrated by a tool rather than
  argued in prose.

### Negative

- **Test files are now unchecked**, so a `time.Now()` in a test helper that a
  domain test depends on would pass. That is a real hole: a non-deterministic
  test is a test that fails randomly, and nothing here catches it. Property
  tests failing intermittently is the only signal.
- The generated-code exemption is a marker match. Anything can claim the marker,
  and hand-written code that does would be silently exempt.
- Widening the pattern to `kernel` is a path convention, not a structural fact.
  A context named `kernel-adapters` would be wrongly matched — the pattern is
  cheap and approximate where a proper module-role registry would be exact.

### Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| Kernel is pure | 2 | `nofloat`, `nondet`, `impurity` (`make analyze`) |
| Test files exempt | 2 | `scope.IsTestFile`, fixture-tested |
| Generated files exempt | 2 | `scope.IsGenerated`, fixture-tested |
| Test files are themselves deterministic | 6 | nothing; see Negative |

Each exemption has a fixture proving the rule stays silent, alongside the
existing fixtures proving it fires. Sensitivity without specificity produces a
rule that gets switched off.

## Alternatives considered

**Leave the pattern at `domain` and rename `libs/kernel` to `libs/kernel/domain`.**
Structurally cleaner — the role would be in the path rather than in a regexp.
Rejected: it makes every kernel import path a word longer for the benefit of a
pattern, and ADR-0013 deliberately gave the kernel a distinct role from a
bounded context's domain layer.

**Apply the rules to test files and inject a deterministic shuffle helper.**
Rejected: the helper would live in the kernel, be used only by tests, and exist
solely to satisfy a rule — ceremony that makes the rule look enforced while
adding nothing.

**Exclude `libs/contracts` by path instead of by generated marker.** Simpler and
exact today. Rejected as narrower: generated code appears wherever `go generate`
runs, and a path exclusion would need extending every time. The marker is the
Go-wide convention.

## Notes

Open:

- Nothing checks that a test is deterministic. A `nondet` variant scoped to
  tests, allowing seeded randomness but rejecting `time.Now()`, is feasible and
  was not built — there is no evidence of the problem yet, and a check with no
  observed failure is one nobody trusts.
- The `kernel` path convention should become a declared module role if the
  module count grows enough for the approximation to bite.
