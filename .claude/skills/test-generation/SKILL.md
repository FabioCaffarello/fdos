---
name: test-generation
description: Write FDOS tests that can fail for the right reason. Use when adding an enforcement mechanism, writing analyser fixtures, testing a pure domain calculation, or asserting reproducibility
---

## Workflow

1. State the property under test as a sentence. If it cannot be stated, the test
   will assert on incidental detail.
2. Write the failing case first and **run it**. A test that has never failed is
   an assertion about nothing.
3. For an enforcement mechanism: break the invariant deliberately, confirm the
   message names the rule, restore.
4. For an analyser: write a violating fixture **and** a compliant one.
5. Prefer a property over a table of examples where the code is pure.
6. Assert on the property, never on message wording or unrelated output order.

## Examples

**An analyser needs both fixtures:**
```go
// testdata/src/ctx/domain/a.go — must fire
type Rate float64 // want `nofloat: type float64`

// testdata/src/ctx/adapters/a.go — must stay silent
type Rate float64
```
```go
func TestDomainPackagesRejectBinaryFloatingPoint(t *testing.T) {
    analysistest.Run(t, analysistest.TestData(), nofloat.Analyzer, "ctx/domain")
}

// Specificity. A rule that fires on legitimate code gets switched off.
func TestAdapterPackagesAreUnaffected(t *testing.T) {
    analysistest.Run(t, analysistest.TestData(), nofloat.Analyzer, "ctx/adapters")
}
```

**Negative-testing a shell fitness function:**
```sh
perl -0pi -e 's/^status: Accepted/status: Superseded/m' docs/adr/0005-*.md
./scripts/verify-adr.sh && echo "RED-TEST FAILED" || echo "ok"
git checkout HEAD -- docs/adr/0005-*.md
```
Restore from git, not from a copy you made — a restore that silently fails
leaves the repository mutated and the next check green for the wrong reason.

**A property beats a table for pure code:**
```go
// Reproducibility is a property, not a set of cases: no permutation of the
// same events may change the result (Constitution §9).
rapid.Check(t, func(t *rapid.T) {
    events := genEvents.Draw(t, "events")
    shuffled := shuffle(events)
    if project(events) != project(shuffled) {
        t.Fatal("fold order changed the result")
    }
})
```

## Quality Bar

- The test failed before it passed. If you did not see it red, you did not test
  it.
- Every new rule has a compliant fixture. Sensitivity without specificity
  produces a rule that gets disabled.
- No assertion on message wording, timestamps, map order, or absolute paths.
- Fixtures restore cleanly; the working tree after a negative test is
  byte-identical to before it.
- Golden files change only with an explanation. An unexplained golden update is
  a reproducibility violation wearing the costume of a test fix.
- `-race` on every test run.

## Resource Strategy

- No `references/`: the testing policy lives in
  `.context/docs/testing-strategy.md` and a copy here would drift.
- `scripts/` only if a fixture becomes fragile enough to justify deterministic
  generation — at which point it probably belongs in the module's own testdata
  tooling instead.
