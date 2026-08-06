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

**E1 — One responsibility, one repository.** If a concern has an owner, the
non-owner does not implement it, does not model it, and does not keep a
"temporary" copy of it. Duplication is not a shortcut that costs time later; it
is a fork of the truth that costs correctness immediately.

**E2 — Contracts flow one way.** `fdos` defines contracts. `fdos-connectors`
consumes them at a pinned version. There is no reverse edge. A need originating
in `fdos-connectors` becomes an RFC in `fdos`, never a type defined in
`fdos-connectors`.

**E3 — No conversational context.** Coordination happens through versioned
artifacts and GitHub. Neither session may rely on the other's memory, on a
chat transcript, or on the human relaying something verbally.

**E4 — Provenance is mandatory.** Every datum that crosses the acquisition
boundary carries its origin: source, method, fetch time, and content hash of the
raw artifact it was derived from. Data without provenance is inadmissible to the
ledger regardless of how correct it looks.

**E5 — Models explain; they are never the source of financial truth.**
(Constitution §2.) No LLM output is routed toward the ledger, the canonical
model, or any figure a user could act on. This includes "just for parsing" and
"just as a fallback".

**E6 — Determinism is checked, not asserted.** Generated artifacts are
regenerable and drift-checked. Same inputs, same outputs, on any machine, at any
time. A generator whose output nobody re-derives in CI is not a generator; it is
a one-time script with a misleading name.

**E7 — Breaking changes are a process, not an event.** Every breaking contract
change carries an RFC, a deprecation window, an N-1 compatibility period, and a
tracked migration issue in every consuming repository, opened *before* the
change merges.

**E8 — Stale documentation is a defect of the change that caused it.** Not a
follow-up ticket. Not a docs sprint. The same pull request.

**E9 — The open core must be usable alone.** `fdos` must build, test, run and
deliver value with the private repository absent, unlicensed, and unbuildable.
This is a product requirement, not only an architectural test: if the only way
to get data in is private, the open core is a demonstration rather than a
platform.

<!-- END TIER-0 -->

---

## Commentary

Non-normative. This part may be edited freely.

### The renumbering, and where the old numbers still stand

These were `I1`–`I8` through `ecosystem/v0.2.0`. They became `E1`–`E9` at
`v0.3.0` because the downstream Integration Charter carries its own `I-1`…`I-10`,
and two unrelated rule sets sharing one prefix is a misreading waiting to happen
— most likely by whoever wrote it, weeks later.

The mapping is the identity: `I1`→`E1` … `I8`→`E8`. `E9` is new.

**The old numbers survive in documents that may not be edited.** `fdos:ADR-0023`,
`fdos:ADR-0024` and `fdos:ADR-0026` cite `I1 I2 I3 I4 I6 I7`, and an accepted ADR
is superseded rather than rewritten (`fdos:ADR-0000`). Each carries a banner
pointing here. Reading a bare `I2` in an FDOS decision predating `v0.3.0` means
`E2`; reading one in the Charter means the Charter's own `I-2`, which is a
different rule. That ambiguity is exactly what the renumbering removes going
forward and cannot remove going backward.

`ecosystem/v0.1.0` and `v0.2.0` remain tagged with the old numbering. Consumers
pinned there are not wrong; they are pinned.

### E2 has a sanctioned inbound channel, and it has been used

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

### E3 is the invariant this ecosystem has actually broken

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

### E9 has a path, and it stops short of a guarantee

`E9` was admitted unmet at `v0.3.0` and sat at rung 6. It now has a public
ingestion path that requires nothing private: a submission message in
`fdos.ingest.v1`, an admission entry point that resolves and mints nothing, and
a conformance kit in `examples/ingest`.

**Its rung is 5, not higher, and the distinction matters.** The kit compiles and
runs in CI, and its fixtures are compared, so *the kit* is checked at rung 3.
`E9` itself is not: nothing detects a would-be adopter who never finds the kit,
or who reads it and writes a producer anyway that nobody runs. It is
documentation that executes, which is the strongest thing an invariant about
*usability* can be.

What would climb it is not another check here. It is a producer outside this
programme submitting something, which is evidence rather than enforcement.

### E6 applies to this directory

[`dependencies.yaml`](dependencies.yaml) is described in the governance brief as
*generated* from issue trailers. It is currently hand-maintained, because no
issue carries a trailer yet and a generator with no input is not a generator. It
says so in its own header, and that is rung 6 by E6's own standard.
