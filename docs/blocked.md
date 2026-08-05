# Blocked work

Work that FDOS has decided to do and cannot finish yet, with what it is waiting
on. Kept because an unrecorded block becomes an unexplained gap: the next
reader cannot tell "not done" from "deliberately not done" from "forgotten".

Each entry states the blocker, what was delivered anyway, and what unblocks it.
Nothing here is a substitute for an ADR — a decision goes in the log; this is
only the register of what that decision could not reach.

---

## B-001 — Private connector consumes the published contract module

**Blocked on:** `financial-connectors` is an empty repository.

**Milestone:** M5. This is M5's stated acceptance criterion:

> `financial-connectors` compiles against a published contract version with no
> filesystem path dependency on this repository.

**Why it is blocked, precisely.** The repository exists but has no commits, no
`go.mod`, and no plugin. It will be built from this reference architecture
rather than the other way round, so it cannot consume anything until it exists.

**Delivered instead, and why it is not a substitute.** M5 proves the *publishing*
half end to end: the contracts module is tagged, resolvable through the Go
proxy, and `make consumer-check` builds a throwaway module against the published
version with `GOWORK=off` — no workspace, no `replace`, no local path.

That proves the module is consumable. It does not prove a *private* repository
can consume it, which is the part that involves credentials, a private module
proxy path, and `GOPRIVATE`. Those are untested.

**What unblocks it:** `financial-connectors` gaining a `go.mod` and one plugin
that imports `github.com/FabioCaffarello/fdos/libs/contracts`. At that point the
conformance suite (also B-002) can run against it.

**Not impeding:** M5 completed without it. The open-core boundary is verified in
the direction this repository controls.

---

## B-002 — Plugin conformance suite

**Blocked on:** B-001, and on there being a plugin interface to conform to.

**Milestone:** M5 listed "plugin SDK skeleton + a conformance test suite private
connectors must pass".

**Why it is blocked.** A conformance suite tests that an implementation honours
an interface. There is no plugin interface: the domain ports it would express
are an M6 output (ADR-0013 puts ports in the `app` layer, which does not exist).
Writing the suite now would define the interface by accident — the same
pre-judgement M1.5 exists to prevent.

**What unblocks it:** the M6 ledger context defining its ports, plus B-001.

---

## B-003 — Two definitions of every canonical concept

**Blocked on:** the Go kernel, which is M6.

**Milestone:** M4 (ADR-0018) created the protobuf wire types. Generated Go
cannot be a domain type — it carries `json:` tags, imports `sync` and `unsafe`,
and holds mutable state, all of which the `impurity` analyser correctly rejects.

**Why it matters.** Wire and domain will diverge unless something proves they do
not. Nothing does today.

**What unblocks it:** a round-trip conformance test at M6 — domain → wire →
domain must be the identity, and every wire field must be reachable.

**Recorded in:** ADR-0018 Consequences, as the largest unpaid cost of that
decision.

---

## B-004 — Claude Code loads no agents from a fresh clone

**Blocked on:** the dotcontext export having no CLI, and Claude Code having no
setting that points at `.context/`.

**Milestone:** M2.5 / ADR-0019.

**Why it is blocked.** The export is an MCP call, so `make bootstrap` cannot run
it. Versioning the export was tried (ADR-0017) and reversed (ADR-0019): it
committed ten skills that were never in the reviewed roster.

**Mitigation, and its weakness.** `make doctor` reports the missing export. That
is rung 5 — it tells a person something and relies on them acting.

**What unblocks it:** either a CLI `bootstrap` can invoke, or confirmation that
`exportSkills` with `includeBuiltIn: false` produces a tree matching `.context/`
exactly — in which case versioning becomes correct and ADR-0019 should be
revisited.
