---
id: ADR-0031
title: PREVC is the working agreement, and the harness executes it
status: Accepted
date: 2026-08-06
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0031 — PREVC is the working agreement, and the harness executes it

## Context

This repository has a working agreement, refined through M8 and M9 and written
in `CONTRIBUTING.md`: a slice starts as a draft pull request carrying the plan,
marking it ready is the approval and never the session's decision, every
mechanism ships a negative test, and `make verify` is the whole gate.

The dotcontext harness offers a five-phase workflow — PREVC: Plan, Review,
Execute, Verify, Confirm — with runtime state, phase gates, per-session sensor
records and task contracts. M9 evaluated it and **declined it**, with the
reason recorded in `.context/docs/tooling.md` and in the closing note of
`.context/config/sensors.json`: PREVC would be a second description of how work
proceeds, and two process descriptions is the drifted-copy problem this
repository already names for `CLAUDE.md` and `AGENTS.md`. The M9 gate (PR #48)
closed that evaluation by leaving the judgement where it belonged: *"it is a
judgement about how this repository works, which makes it yours more than
mine."*

Two things changed since that declination, and they are the honest content of
this reversal:

1. **The human decided.** On 2026-08-06, asked directly with the M9 argument
   restated, @FabioCaffarello chose adoption-as-replacement over keeping the
   declination and over running PREVC beside the agreement.
2. **The runtime acquired a subject.** M9.5 starts a calibration program: each
   milestone runs in a dedicated session driven by a controlled prompt, and
   the harness runtime — workflow phase state, session traces, sensor records,
   task contracts — is the instrumentation that makes those sessions
   comparable. The M9 evaluation declined machinery that had nothing to
   instrument; the program is the subject it found missing.

The declination itself was recorded in `.context/`, not in an ADR, so there is
nothing to supersede — but the record of it must survive this change, because
erasing a declination reads as though it never happened. Constitution §14
(architecture before implementation) is why this arrives as a decision before
`CONTRIBUTING.md` changes shape.

## Decision

FDOS adopts PREVC as the working agreement, **as a renaming of the agreement it
already has, executed on the dotcontext harness** — not as a process beside it.

- **P — Plan** is the draft pull request carrying the plan.
- **R — Review** is the human review of that plan; marking the pull request
  ready is the approval, and the keystroke rule in `CONTRIBUTING.md` is
  unchanged.
- **E — Execute** is the slices, on the gate's branch.
- **V — Verify** is `make verify` per slice, recorded as the `verify` sensor
  on the milestone's harness session.
- **C — Confirm** is the pull request ready for a human to merge, and the
  harness session checkpointed and completed.

`CONTRIBUTING.md` is rewritten in this same change so that exactly one
description of the process exists, with the PREVC letters naming its stages. A
milestone session opens with `workflow-init` and closes its harness session;
phase state lives under `.context/runtime/` and stays untracked. Where the
harness's built-in PREVC skills and `CONTRIBUTING.md` disagree,
`CONTRIBUTING.md` wins, and the disagreement is a bug to report upstream.

ADR-0015's deliberately open question — whether prompt contracts should carry a
`phases` obligation tied to PREVC, or whether that is ceremony — is closed:
they carry it. The letters now name phases of the working agreement itself, so
the field has a referent, and the sensor catalog regains its `phases` markers
for the same reason.

## Consequences

### Positive

- One vocabulary across `CONTRIBUTING.md`, the prompt contracts, the sensor
  catalog and the harness runtime. The drifted-copy objection is answered the
  only way it can be: by there being one copy.
- Execution evidence becomes queryable per milestone: which phases ran, what
  the sensors recorded, where a session stopped. The calibration log gets a
  stable frame to compare sessions against.
- The plan-gate loses nothing. Its two hardest-won rules — ready is the
  approval, and the keystroke is the session's only on explicit instruction —
  are P and R's gate, verbatim.

### Negative

- **This reverses a decision one milestone old**, and the M9 argument was
  sound. If the calibration program stops, the workflow reverts to ceremony —
  the playbook with no subject arrives after all. The trigger to supersede
  this ADR: two consecutive milestone sessions whose workflow state records
  nothing a pull request body does not already record.
- **The first execution measured real defects in the machinery being
  adopted.** `plan link` rejects `required_sensors` on execution phases in
  every documented format while still half-registering the link, after which
  two code paths disagree about whether a plan is linked at all; the P→R and
  R→E gates were passed with an explicit, trace-recorded `force`. Until those
  are fixed upstream, the phase gates are honour-system plus traces, and this
  ADR is honest about adopting them in that state.
- The built-in dotcontext skills describe a generic PREVC. That is a new drift
  surface between a vendored description and this repository's, managed by the
  authority rule above rather than by a mechanism.

### Enforcement

Documentation and human discipline (rungs 5–6), with one mechanical element:
the `verify` sensor recorded per session is a machine record that V actually
ran. What would climb the ladder: a check that a merged gate pull request has a
matching completed harness session. Deliberately not built now — session state
is untracked runtime, and inventing an export format to check it would be a
mechanism ahead of its subject.

## Alternatives considered

**Keep the declination.** Lost because the calibration program needs the
workflow runtime as instrumentation, and the human — asked with the M9
argument in front of them — decided the trade the other way now that the
runtime has a subject.

**Adopt PREVC beside the working agreement.** The exact drifted-copy failure
M9 named. Lost then, and lost now for the same reason; nothing about the
calibration program makes two descriptions safer.

**Instrument without renaming — sessions and sensors, but no PREVC phases.**
The closest contender. Lost because phase state is what the runtime keys its
gates, skills and progress records on: instrumenting while refusing the
vocabulary would leave every recorded phase mislabelled relative to the
process description, which is drift by construction rather than by accident.

## Notes

- First execution: M9.5 itself, the pilot calibration session.
- The upstream defects measured during adoption (`plan link` validation, link
  state inconsistency, and `sync --dryRun` writing — the last already recorded
  under B-004) are to be reported against dotcontext as part of M9.5.
- The declination this ADR reverses remains readable: in this ADR's Context,
  and in `.context/docs/tooling.md`'s history of what was bound and when.
