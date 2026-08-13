---
id: ADR-0050
title: Build output is never tracked, and a magic-byte check is what says so
status: Accepted
date: 2026-08-12
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0050 — Build output is never tracked, and a magic-byte check is what says so

> The last open item of [#79](https://github.com/FabioCaffarello/fdos/issues/79),
> which that issue flagged for a human rather than acting on: *"removing a
> tracked artifact on an inference is not a session's call."* It is no longer an
> inference.

## Context

`examples/ingest/ingest` was tracked from PR #47: **7.3 MB, `Mach-O 64-bit
executable arm64`** — one developer's local build, committed into the directory
whose job is to show a third party what conformance looks like.

Three things make it worse than untidy.

**It escapes §9 entirely.** Nothing records which source produced it, which
toolchain built it, or which platform it runs on. `make repro-check` cannot
compare it against anything, because it is not built by anything the gate
invokes. The `darwin/arm64` magic is the only evidence of its provenance, and it
is evidence that it is useless to almost everyone who clones this repository.

**It is undocumented.** `examples/README.md` carries a table of the kit's
deliverables and the binary appears in no row of it. The table reads as complete
and was not.

**It is live, not inert.** An ordinary `go build ./...` in that directory
overwrites it silently. That happened during the work that brought `examples/`
into the gate (ADR-0044) — the file turned up modified in `git status` with no
edit made to it. So it was a 7.3 MB diff waiting for whoever ran the most
obvious command in the module.

Nothing detected any of this. It was found by reading, twice, and flagged both
times.

## Decision

### 1. No compiled executable is tracked

`examples/ingest/ingest` is removed, and `go build` output in that directory is
ignored. `make build` already writes to `bin/`, which was already ignored; the
gitignore entry covers the command a developer actually runs instead.

### 2. Detection is by magic bytes

`make no-binaries-check` reads the first four bytes of every tracked file and
fails on ELF, Mach-O, Mach-O universal and PE images.

**Not by the executable bit**, because every script in `scripts/` carries it and
none of them is a compiled artifact. **Not by extension**, because a binary
committed without a suffix — which is exactly what happened here — is the case a
name-based check misses.

The `cafebabe` magic is Mach-O universal and also a Java class file. Neither
belongs in this repository, so the ambiguity costs nothing and is not worth a
second discriminator.

### 3. What this does not decide

**Nothing about binary *fixtures*.** `examples/ingest/testdata/conforming.bin` is
a serialized protobuf message and is exactly what the kit is for. It is not an
executable image and the check does not look at it. A future fixture that *is*
an executable image would fail, and the failure message says what to do: give it
a reason, because nobody has written one.

**No history rewrite.** The 7.3 MB object stays reachable in every clone made
since PR #47. Removing it from the working tree does not remove it from the
repository, and rewriting history to shrink a clone is a change to the record
that Constitution §4 exists to prevent.

## Consequences

### Positive

- A build artifact cannot be committed without the gate saying so, whatever it
  is called.
- `examples/README.md`'s table is complete, which matters more there than
  elsewhere: it is the file a third party reads to learn what the kit is.
- A `go build` in the kit no longer produces a spurious multi-megabyte diff.

### Negative

- **The clone does not get smaller.** The object is still in history and always
  will be. This stops the next one, and does nothing about this one.
- **Magic-byte detection is a heuristic with two known collisions.** `cafebabe`
  catches Java classes; a file that happens to begin with `MZ` and is not a PE
  image would be flagged. Both are acceptable here and would be wrong in a
  repository that legitimately held either.
- **The check reads every tracked file.** It is 344 files today and four bytes
  each; at a repository with binary assets it would need a different shape.
- **Nothing checks that `examples/README.md`'s table is complete.** This ADR
  makes the specific omission impossible and leaves the general one — a
  documented table drifting from a directory's contents — exactly where it was.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| No compiled executable is tracked | 3 | `make no-binaries-check`, negative-tested |
| Build output stays out of the tree | 3 | `.gitignore`, plus the check above |
| The kit's file table is complete | 6 | review |

The negative test ran in the most convincing direction available: the check was
written while the violation was still present, failed on it naming the file,
format and size, and passed once the file was removed.

## Alternatives considered

**Delete the file and add a `.gitignore` line, with no check.** The minimal
change, and it fixes the instance. Rejected: the file was flagged twice by
reading and caught by nothing, which is the definition of a rule held at rung 6.
A second one would arrive the same way.

**Use `file(1)` rather than magic bytes.** Better classification, one line of
script. Rejected as an unnecessary dependency for a check that runs first on a
clean clone — `scripts/README.md` requires nothing beyond POSIX utilities, and
`od` is one while `file`'s output format is not stable enough to parse across
platforms.

**Flag anything with the executable bit.** Simpler and catches more. Rejected
outright: it would flag every script in `scripts/`, which is most of the
enforcement mechanism this repository has.

**Rewrite history to remove the object.** It would actually shrink the clone,
which is the only real cost of the current state. Rejected: rewriting published
history is what Constitution §4 and ADR-0000 are about, and 7.3 MB is not worth
the precedent.

**Leave it, and record the gap.** Not a straw man — #79 chose it twice,
correctly, because deleting a tracked artifact on an inference is not a session's
call. Rejected now that it is not an inference: the file is a `darwin/arm64`
build, undocumented in the table that claims to list the kit, and demonstrably
overwritten by the module's most obvious command.
