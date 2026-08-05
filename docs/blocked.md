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

**What unblocks it:** a round-trip conformance test — domain → wire → domain
must be the identity, and every wire field must be reachable.

**RESOLVED for the kernel types (M7).** `libs/kernel-wire` maps every kernel
type and asserts two properties over generated values:

```
domain -> wire -> domain   is the identity   (nothing lost encoding)
wire   -> domain -> wire   is the identity   (nothing dropped decoding)
```

The second is what earns its keep: a codec that never reads `published_at`
passes the first forever, because the value it fails to carry was never in the
domain value it compares against. Both were negative-tested — dropping a field
and mis-mapping a rounding mode each make the suite fail.

Writing it needed exactly one addition to the kernel (`provenance.NewRef`),
which is a good signal for the encapsulation.

**RESOLVED for the ledger types (M7).** `libs/ledger-wire` maps `Fact`,
`Envelope`, `Correction` and the `HoldingObserved` payload under the same two
properties, negative-tested three ways: dropping reference bindings, mapping an
entity kind to the wrong value, and skipping the declared-type check each make
the suite fail.

`DecodeFact` checks the declared `type` and `type_version` against the unpacked
payload rather than trusting them. A fact claiming to be a `ledger.TradeSettled`
while carrying a holding observation is the shape a corrupted stream takes.

**B-003 is closed.** Every canonical concept now has a mechanism proving its two
definitions agree. What remains is scope rather than absence: a payload type
added without a conformance test would still drift, which is why
`libs/ledger-wire/README.md` makes the test part of the procedure for adding
one.

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

---

## B-005 — Dependency review on pull requests

**Blocked on:** GitHub's Dependency Graph being unavailable on this repository.

**Milestone:** M3 added `dependency-review` to `supply-chain.yml`, with a
copyleft deny-list protecting the Apache-2.0 claim in `NOTICE` (ADR-0002).

**Why it is blocked.** Every run fails with *"Dependency review is not supported
on this repository. Please ensure that Dependency graph is enabled."* It cannot
be turned on through the REST endpoints available here.

**Delivered instead.** The job is disabled rather than left permanently red — a
check that always fails trains people to ignore CI, which costs more than the
check was worth. `make vuln-check` (govulncheck, reachable vulnerabilities) runs
in the gate regardless, so dependency scanning is not absent; the PR-delta and
licence view is.

**What unblocks it:** enabling Dependency graph in repository settings, then
removing the `false &&` guard in `.github/workflows/supply-chain.yml`.

---

## B-006 — Signed commits

**Blocked on:** no SSH signing key registered with GitHub, and `gh` lacking the
`admin:ssh_signing_key` scope to add one.

**Milestone:** M5. `required_signatures` was applied in the `main` ruleset and
**removed the same day**, because it blocked every merge.

**Why it was removed.** The first pull request reached
`mergeable_state=blocked` with a green `verify` and zero required approvals, and
merged only with `--admin`. GitHub signs its own squash-merge commit, but that
was not sufficient for the rule.

A protection rule that must always be bypassed is worse than no rule: it trains
the one person who can bypass it to reach for `--admin` by reflex, and the next
rule that fires for a real reason gets the same treatment.

**What unblocks it:**

```sh
gh auth refresh -h github.com -s admin:ssh_signing_key
gh ssh-key add ~/.ssh/id_ed25519.pub --type signing
git config gpg.format ssh
git config user.signingkey ~/.ssh/id_ed25519.pub
git config commit.gpgsign true
```

Then re-add `required_signatures` to the `main` ruleset.

Constitution §6 says authorship is part of provenance. Until this is done, it is
not — and `docs/branch-protection.md` says so rather than implying coverage.

---

## B-007 — No published message is fully populatable by a connector

**Blocked on:** acceptance of [RFC-0007](rfc/0007-identity-resolution-and-the-acquisition-boundary.md).

**Raised by:** `fdos-connectors`, as [fdos#10](https://github.com/FabioCaffarello/fdos/issues/10),
against `contracts@v0.2.0`.

**The finding, verified independently here** by enumerating every published
message rather than by reading the ones under discussion. Exactly three carry
identity, and all three require it:

| Message | Requires |
|---------|----------|
| `ledger.payload.v1.HoldingObserved` | 2 × `EntityId` |
| `kernel.v1.IdentifierAssertion` | 1 × `EntityId` |
| `kernel.v1.EntitiesIdentified` | 2 × `EntityId` |

A connector knows `{scheme: "ticker", value: "PETR4"}` and cannot know an
`EntityId`. It must not mint one: ADR-0007 records that deriving from a ticker
makes the ticker the primary key, and a reused ticker then merges two
instruments silently inside an append-only ledger.

**The circularity underneath.** `IdentifierAssertion` is the shape that would
carry the claim, and it requires the identity it exists to assert. That is not
an oversight in the message — it is a **missing event**. An `EntityId` comes
into existence at some moment, and FDOS had never said what that moment is.

**DECIDED.** RFC-0007 accepted, recorded by
[ADR-0022](adr/0022-minting-an-identity-is-a-fact.md). Minting is a fact, a
connector emits a claim, and resolution is a derivation recorded in the ledger
rather than a precondition of appending.

Released as `contracts@v0.3.0`, additive: `kernel.v1.IdentifierClaim`,
`ledger.payload.v1.HoldingClaimed`, `ledger.payload.v1.EntityMinted`. No existing
message changes shape. **`fdos-connectors` B-009, C2 and C4 are unblocked** — the
hand-off shape exists.

**Built since, in four publish cycles** — the measured cost of ADR-0004, paid
again and visible in the ordering:

| | Released |
|---|---|
| `explained.FromDerivation`, so a trace cannot lose its parameters | `kernel@v0.5.0` |
| `HoldingClaimed`, `EntityMinted`, `Resolve`, `DeriveHoldingObserved`, `MintFor` | `ledger@v0.2.0` |
| Identity and claim codecs, moved to the module whose contract covers them | `kernel-wire@v0.2.0` |
| Codec and round-trip conformance for both new payloads | `ledger-wire@v0.2.0` |

**A defect found by building it.** `DeriveHoldingObserved` first built a
derivation record with three parameters and handed only its inputs and
confidence to `explained.FromObservation`, which rebuilt the record without
them. `as_of` is not recoverable from the named facts, and resolving the same
claim at a different coordinate can select a different mint — so two genuinely
different derivations would have shared one content address, with both traces
looking complete. The gap was in the kernel: every combinator took `parameters`
and the entry point did not.

**Still open:**

- Nobody notices an unresolved claim. Claims accumulate and no `HoldingObserved`
  is derived; who is told, and how, is operational and undecided.
- The claim vocabulary is an open string. `identity.NewClaim` now refuses a
  non-canonical *scheme*, closing the `"Ticker"` / `"ticker"` half at rung 1.
  The other half — `"ticker"` and `"symbol"` for the same concept — is
  vocabulary governance and no type can solve it.
- `HoldingObserved`'s provenance must be `Derived`. proto3 cannot express it and
  the Go domain type does not yet enforce it, so it stays rung 6 exactly as
  ADR-0022 recorded.
- No `IdentifierAssertion` codec. Nothing produces one yet, and adding a codec
  ahead of a producer would be a conformance test with no subject.
