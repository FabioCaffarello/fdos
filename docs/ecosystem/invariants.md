# Ecosystem invariants

The rules that hold across every repository in the FDOS ecosystem, not just this
one.

**Tier 0.** The block below is authored here and vendored verbatim by every
consuming repository. It is byte-identical to the corresponding section of the
engineering brief each session is given. It is amended only by an RFC in `fdos`
followed by an ADR in both repositories — never paraphrased, never improved
locally, never edited downstream.

Commentary belongs *outside* the block. Anything inside it that turns out to be
wrong is fixed by amendment, not by a helpful edit.

---

<!-- BEGIN TIER-0: ecosystem invariants — do not edit; amend by RFC + ADR in both repositories -->

**I1 — One responsibility, one repository.** If a concern has an owner, the
non-owner does not implement it, does not model it, and does not keep a
"temporary" copy of it. Duplication is not a shortcut that costs time later; it
is a fork of the truth that costs correctness immediately.

**I2 — Contracts flow one way.** `fdos` defines contracts. `fdos-connectors`
consumes them at a pinned version. There is no reverse edge. A need originating
in `fdos-connectors` becomes an RFC in `fdos`, never a type defined in
`fdos-connectors`.

**I3 — No conversational context.** Coordination happens through versioned
artifacts and GitHub. Neither session may rely on the other's memory, on a
chat transcript, or on the human relaying something verbally.

**I4 — Provenance is mandatory.** Every datum that crosses the acquisition
boundary carries its origin: source, method, fetch time, and content hash of the
raw artifact it was derived from. Data without provenance is inadmissible to the
ledger regardless of how correct it looks.

**I5 — Models explain; they are never the source of financial truth.**
(Constitution §2.) No LLM output is routed toward the ledger, the canonical
model, or any figure a user could act on. This includes "just for parsing" and
"just as a fallback".

**I6 — Determinism is checked, not asserted.** Generated artifacts are
regenerable and drift-checked. Same inputs, same outputs, on any machine, at any
time. A generator whose output nobody re-derives in CI is not a generator; it is
a one-time script with a misleading name.

**I7 — Breaking changes are a process, not an event.** Every breaking contract
change carries an RFC, a deprecation window, an N-1 compatibility period, and a
tracked migration issue in every consuming repository, opened *before* the
change merges.

**I8 — Stale documentation is a defect of the change that caused it.** Not a
follow-up ticket. Not a docs sprint. The same pull request.

<!-- END TIER-0 -->

---

## Commentary

Non-normative. This part may be edited freely.

### I2 has a sanctioned inbound channel, and it has been used

"No reverse edge" governs *types*, not *needs*. A consumer that discovers a gap
does not define the missing type; it opens an issue here, and the need travels
through the RFC process like any other.

That path has a worked example. `fdos-connectors` found that no published
message could be fully populated by a connector and raised
[fdos#10](https://github.com/FabioCaffarello/fdos/issues/10). The need became
[RFC-0007](../rfc/0007-identity-resolution-and-the-acquisition-boundary.md),
the decision became [ADR-0022](../adr/0022-minting-an-identity-is-a-fact.md),
and the resolution shipped as an additive contract release. No type was defined
downstream at any point. See [`../blocked.md`](../blocked.md) — B-007.

### I3 is the invariant this ecosystem has actually broken

Not maliciously, and not with bad results — but broken. Both repositories were
developed for months against boundary rules that existed only inside two prompts,
because this corpus did not exist. The consumer built its vendoring manifest,
its drift check and a host↔plugin wire contract with nothing canonical to check
them against.

They happen to be compatible. That is care and luck, not a mechanism, and it is
the reason [ADR-0023](../adr/0023-ecosystem-boundary-and-one-way-contract-flow.md)
exists.

### Citing a decision in the other repository

Each repository keeps its own ADR and RFC sequence, and they have already
collided. `fdos:ADR-0019` says the Claude Code export is not versioned;
`fdos-connectors:ADR-0019` decides the namespace of their plugin schema. Same
number, unrelated subjects. A bare `ADR-0019` in a document spanning both is
ambiguous and will be misread — most likely by whoever wrote it, some weeks
later.

**Cross-repository references are always qualified:** `fdos:ADR-0014`,
`fdos-connectors:ADR-0019`, `fdos:RFC-0007`.

`make context-check` understands the convention: a reference qualified with
another repository's name is skipped, because resolving it would require this
repository to contain the other's decision log. One qualified with `fdos:` is
still resolved locally, so a typo in our own name cannot silence the check.

Both behaviours are negative-tested. This is rung 3 for the half that can be
checked — that a cited local decision exists — and rung 6 for the half that
cannot: nothing here can tell whether `fdos-connectors:ADR-0019` says what a
document claims it says.

### I6 applies to this directory

[`dependencies.yaml`](dependencies.yaml) is described in the governance brief as
*generated* from issue trailers. It is currently hand-maintained, because no
issue carries a trailer yet and a generator with no input is not a generator. It
says so in its own header, and that is rung 6 by I6's own standard.
