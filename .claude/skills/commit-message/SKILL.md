---
name: commit-message
description: Write FDOS commit messages that record reasoning, trade-offs and superseded decisions. Use when committing staged changes, writing a message for a structural or governance change, or documenting a reversal
---

## Workflow

1. Review staged changes with `git diff --staged`.
2. Pick the type: `feat`, `fix`, `docs`, `chore`, `refactor`, `test`, `build`.
3. Write an imperative subject line under 50 characters, no trailing period.
4. In the body, record **why**. The diff already says what.
5. State costs and trade-offs accepted, not just benefits.
6. Name any ADR the change embodies, and any decision it supersedes.
7. If a check was added, say it was tested against negative cases.

## Examples

**Governance reversal:**
```
docs: adopt .context, superseding ADR-0001

The canonical AI knowledge directory is `.context/`, matching the tooling
default and the convention already used across sibling repositories.

Recorded as ADR-0006 superseding ADR-0001 rather than as an edit to
ADR-0001, per ADR-0000: the decision log is append-only, and a reversal
that leaves no trace reads as though the reversed choice was never made.

The `outputDir` discipline and the M1 guard that ADR-0001 required in
order to defend its own naming choice are dropped rather than built.
```

**Enforcement mechanism:**
```
feat(scripts): enforce bidirectional supersession in the ADR log

Requiring a successor to be *named* did not require it to *exist*, or to
point back. A one-directional link makes the chain traversable forwards
only, so a reader arriving at the successor cannot tell what it replaced.

Verified against five negative cases: dangling successor, dangling
predecessor, missing back-link in either direction, predecessor still
marked Accepted.

Constitution §15: §14 stays at rung 3, mechanism strengthened.
```

## Quality Bar

- Imperative mood: "add", not "added" or "adds".
- Subject under 50 characters, no trailing period.
- Body explains why; never restate the diff.
- **Costs stated.** A message listing only benefits will not be trusted on the
  one occasion the cost matters.
- A reversal names the superseding ADR explicitly. Never let a reversal read as
  though the original decision never happened.
- A new check states that it was negative-tested. If it was not, that is the
  problem to fix before committing.
- One logical change per commit.

## Resource Strategy

- None needed. This is a writing procedure, not a mechanical one; a script would
  produce messages that satisfy the format and miss the point.
