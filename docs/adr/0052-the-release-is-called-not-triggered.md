---
id: ADR-0052
title: The release is called rather than triggered, so publication is one dispatch again
status: Accepted
date: 2026-08-13
deciders:
  - "@FabioCaffarello"
supersedes:
  - ADR-0051
superseded_by: []
---

# ADR-0052 — The release is called rather than triggered, so publication is one dispatch again

> Phase 4's shape, third attempt. [ADR-0046](0046-publishing-a-tag-is-a-dispatched-act.md)
> assumed a tag push would start the release and it did not;
> [ADR-0051](0051-a-tag-pushed-by-a-workflow-triggers-nothing.md) made the
> release dispatchable and accepted two human acts, on the explicit condition
> that `workflow_call` was "the right answer once the path has published
> something without incident". It has.

## Context

ADR-0051 established the constraint and left the remedy conditional:

> GitHub suppresses workflow runs for events created with `GITHUB_TOKEN` […] A
> tag pushed by a workflow is such an event.

and, rejecting `workflow_call` at the time:

> Not taken for now on evidence rather than principle: this is the fourth
> consecutive defect in this path, three of them from adding indirection […] It
> remains the right answer once the path has published something without
> incident.

**`libs/ledger-wire/v0.5.0` published without incident.** Every step of
`release.yml` succeeded, and the result was verified as an adopter would: the
manifest describes the bytes, the provenance attestation verifies against
`.github/workflows/release.yml` in this repository, and the release carries the
module's own zip and an SBOM named after the module.

So the condition ADR-0051 named is met, and the cost it accepted is now the
thing worth removing: two human acts, the second of which nothing enforces.

## Decision

### 1. `release.yml` is callable

It keeps `push` and `workflow_dispatch`, and gains `workflow_call` with the same
`tag` input. One job, three ways in. Every step already names the tag through
`inputs.tag || github.ref_name`, so nothing else changes.

### 2. `release-tag.yml` calls it, in the same run

A second job, `needs: tag` and gated on the same explicit `publish == 'yes'`.
**A call is not an event**, so the `GITHUB_TOKEN` guard that suppressed the push
trigger does not apply to it.

The permissions the release needs — `contents: write`, `id-token: write`,
`attestations: write` — are declared on the calling job rather than inherited
broadly, so the tag job keeps the narrower set it had.

### 3. A green dispatch now means the release happened

Under ADR-0051 a green `release-tag` run meant *a tag exists and something
should be run next*. It now means the artifacts were built, signed, attested and
published. That is the property worth having: B-008 is what happens when a
release path's success signal does not mean the release succeeded.

### 4. What this does not decide

**`workflow_dispatch` on `release.yml` stays.** It is what recovered
`libs/ledger-wire/v0.5.0` from ADR-0051's state, and it is the only way to
rebuild a release for a tag that already exists.

**The `push` trigger stays.** A tag pushed from a maintainer's own credentials
still triggers it — that path works and is how `libs/kernel-wire/v0.3.0` reached
`release.yml` at all.

## Consequences

### Positive

- Publication is one human act again, which is what ADR-0046 designed and never
  achieved.
- The second act cannot be forgotten, because there is no second act. ADR-0051's
  rung-5 printed instruction is gone.
- A green run means a published release rather than a started one.

### Negative

- **This is the third shape of Phase 4 in two days**, and the first two were
  each believed correct when written. The evidence that this one is right is one
  clean release; the evidence the previous two were wrong was also one release
  each.
- **The call is exercised only by publishing.** Dispatching with `publish=no`
  skips the release job entirely, so a malformed call surfaces on the next real
  release. That is the same trap ADR-0039 named, in the one place this
  repository keeps stepping into it.
- **Permissions are declared in two places now** — the tag job's and the calling
  job's. A future step needing a permission in the wrong one fails at the point
  it is used, which is late.
- **A failure in the release job leaves a pushed tag**, exactly as before. The
  difference is that the failure is attached to the same run as the tag, rather
  than to a run nobody dispatched.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| A tag is followed by its release | 3 | `release-tag.yml`'s `release` job, `needs: tag` |
| Publication is a human act | 3 | one `workflow_dispatch`, `publish` typed as `yes` |
| A release can be rebuilt for any tag | 3 | `release.yml` `workflow_dispatch` |
| The call itself works | 6 | performing a release. Nothing else runs it |

The last row is unchanged from ADR-0039's assessment and is the reason this ADR
records what it costs rather than only what it fixes.

## Alternatives considered

**Keep two dispatches.** ADR-0051's shape, and it works — it published a release
today. Rejected because the second dispatch is enforced by a printed line, and
the state it prevents (a tag nothing builds) is the state that produced B-008.

**A personal access token so the push triggers workflows.** Restores ADR-0046's
original design exactly, with no call and no second job. Rejected for the third
time on the same grounds: it is a credential nobody reviews, granted to the
build.

**Merge tagging and releasing into one job.** No call, no `needs`, no permission
split. Rejected again: it would put `contents: write` in the same job that
builds and signs, and confining that grant is what ADR-0046 spent its design on.

## Notes

- The workflow was validated by parsing it and by `make verify`; the *call* is
  not exercisable without publishing. The next release is what proves it, and
  the one after that is what proves this ADR did not become the fourth shape.
