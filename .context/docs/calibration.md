---
type: doc
name: calibration
description: The milestone calibration program — session prompts, the harness bootstrap, and the feedback loop that tunes both
category: workflow
generated: 2026-08-06
status: filled
scaffoldVersion: "2.0.0"
---

# The calibration program

Each milestone runs in a dedicated session driven by one prompt file, under the
PREVC working agreement (ADR-0031). The prompt is the controlled input; the
calibration log is the recorded output; the harness runtime is the
instrumentation between them. This document is the canonical home of the
protocol and the template. The instances — the prompt files themselves and the
log — live in `_prompts/`, which is local and deliberately unversioned: the
prompts are operator input, not repository knowledge, and versioning them here
would make every prompt tweak a commit to a repository that is not about
prompts.

One consequence, measured in the M9.5 pilot: `verify-directory-contracts.sh`
and `verify-doc-references.sh` walk the **filesystem**, not the git index. A
gitignored `_prompts/` still needs a valid front-matter README contract
locally, and its files must not cite ADR or RFC numbers that do not exist.

## The loop

1. The human opens a fresh session and pastes `_prompts/<milestone>.md`.
2. The session checks the gate PR is **ready** (never marks it so), runs the
   harness bootstrap below, and works the gate slice by slice through PREVC.
3. `make verify` runs per slice and is recorded as the `verify` sensor on the
   session; the session ends with a checkpoint and `completeSession`.
4. The session drafts a calibration log entry; the human's edit of that entry
   is the verdict.
5. Before the next session, the next prompt is adjusted from the log — and
   where a friction has a possible mechanism (a sensor, a policy rule, a
   check), it becomes a mechanism rather than prompt text. The enforcement
   ladder applies to prompts too.

Metrics per session: `verify` failures; human corrections; PREVC phases that
did not match reality; policy triggers; drift found by `context-check` and
`agent-contract-check`.

## The harness bootstrap

```
workflow-init({ name: "<milestone-slug>" })
harness createSession({ name, metadata: { milestone, gatePr } })
workflow-manage defineTask({ taskTitle, acceptanceCriteria: <the gate's>,
                             requiredSensors: ["verify"] })
… work by phases, workflow-advance at transitions …
harness recordSensor after every make verify
workflow-manage checkpoint · harness completeSession
```

Known upstream defects (measured in the pilot, reported against
`vinilana/dotcontext`): `sync` `dryRun` writes (the B-004 incident, issue
[#54](https://github.com/FabioCaffarello/fdos/issues/54)); `plan link` rejects
`required_sensors` in every documented format while half-registering the link,
so phase gates may need an explicit, trace-recorded `force`. Never call
`sync export*` from a session.

## The prompt template

Every prompt file carries, in order: **Mission** (one sentence, milestone and
gate PR links) · **Gate check** (if the PR is draft, refine the plan only) ·
**must_read** (the `AGENTS.md` order, then milestone-specific decisions) ·
**must_not** (no `sync export*`; never mark a PR ready; supersede, never edit;
nothing LLM-ward of the ledger; respect policy) · **Harness bootstrap** (the
sequence above) · **Slices** (the gate's list with definitions of done) ·
**Evidence required** (`make verify` stated plainly, sensors recorded, the PR
body carries the argument) · **Feedback** (draft the calibration entry).
