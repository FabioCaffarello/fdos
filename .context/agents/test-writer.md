---
type: agent
name: Test Writer
description: Write tests that can fail for the right reason, and fixtures that prove a rule stays silent
agentType: test-writer
phases: [E, V]
generated: 2026-08-05
status: filled
scaffoldVersion: "2.0.0"
must_read:
  - .context/docs/testing-strategy.md
  - libs/analysis/README.md
  - docs/constitution.md
must_not:
  - Add an enforcement mechanism without a test that makes it fail
  - Write a fixture that proves sensitivity without one that proves specificity
  - Assert on incidental output instead of the property under test
  - Report a test as passing without having run it
  - Restore a fixture with `git checkout` when the file is untracked
evidence:
  - A failing run before the fix and a passing run after
  - For a new rule, both a violating and a compliant fixture
---

# Test Writer

Restored at M2.5. Before M2 there was no Go code and this playbook had no
subject; it was removed at M1 rather than left to describe a repository that did
not exist.

## Available skills

| Skill | Description |
|-------|-------------|
| [test-generation](./../skills/test-generation/SKILL.md) | Write tests that can fail for the right reason |

## The rule that matters most here

> A check that has never gone red is unverified.

Every enforcement mechanism in this repository has been broken deliberately to
confirm it fails with a useful message. Two of the original four had real
defects that only that exercise surfaced — one double-counted silently, one had
a fragile `set -e` interaction. Both passed the happy path.

When you add a check, break it. When you review one, ask whether anyone has.

## Sensitivity is half the job

An analyser fixture proving a rule **fires** on bad code is the easy half. The
fixture proving it stays **silent** on good code is what decides whether the
rule survives, because a rule that reports legitimate code gets switched off,
and a switched-off rule enforces nothing.

This found a genuine defect in `nondet`. The remedy for "do not range over a
map" is itself a map range:

```go
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
```

The first version reported that loop, which would have made the rule impossible
to satisfy without a suppression comment. Only the compliant fixture exposed it.

Every analyser therefore has fixtures under both
`testdata/src/ctx/domain/` (must fire) and `testdata/src/ctx/adapters/`
(must stay silent).

## Restoring after a negative test

Break the invariant, confirm the failure, restore — and **verify the restore**.

`git checkout HEAD -- <path>` reverts a tracked file but *deletes* an untracked
one. A restore helper that does not distinguish the two will silently destroy
new work while reporting success. This happened during M2.5: the helper deleted
this very file, and `make context-check` caught it only because the roster still
linked here.

Prefer copying to a scratch location and back, and diff afterwards:

```sh
cp "$f" /tmp/f.bak
# ... mutate, assert failure ...
cp /tmp/f.bak "$f"
diff /tmp/f.bak "$f" || echo "RESTORE FAILED"
```

## When domain code arrives (M6)

Two techniques become primary, and neither is optional for financial
calculations:

**Property-based testing** (`pgregory.net/rapid`). The domain is a pure
functional core, which is exactly the shape properties are strongest against.
Prefer "this projection is invariant under event reordering within a timestamp"
over a table of examples — the property catches a class of bug that examples
never reach.

**Golden files for reproducibility.** A fixed ledger plus fixed reference data
must produce byte-identical output. A golden file that changes without an
explanation is a reproducibility violation, not a test update.

## Do not test the mechanism you are inside

Tests assert on the property, not on the wording of a message or the order of
unrelated output. A test coupled to incidental detail fails on every unrelated
change, and a suite that cries wolf gets skipped.
