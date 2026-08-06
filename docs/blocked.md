# Blocked work

> **This register is frozen (ADR-0032).** Blocked work is now registered as
> GitHub issues carrying the `status/blocked` label, using the taxonomy in
> [`docs/ecosystem/labels.md`](ecosystem/labels.md). The entries below are
> permanent history: the `B-NNN` identifiers stay citable, nothing may be
> added here, and the open entries are annotated with the issue that now
> tracks them.
>
> | Entry | State at freeze (2026-08-06) | Live record |
> |---|---|---|
> | B-001 | Resolved; private-path resolution unobservable from here | — |
> | B-002 | Open | [#53](https://github.com/FabioCaffarello/fdos/issues/53) |
> | B-003 | Closed | — |
> | B-004 | Open | [#54](https://github.com/FabioCaffarello/fdos/issues/54) |
> | B-005 | Open | [#55](https://github.com/FabioCaffarello/fdos/issues/55) |
> | B-006 | Open | [#56](https://github.com/FabioCaffarello/fdos/issues/56) |
> | B-007 | Decided; five open items | [#57](https://github.com/FabioCaffarello/fdos/issues/57) |
> | B-008 | Resolved | — |
> | B-009 | Publishing half done; consumer half unobservable from here | — |

Work that FDOS has decided to do and cannot finish yet, with what it is waiting
on. Kept because an unrecorded block becomes an unexplained gap: the next
reader cannot tell "not done" from "deliberately not done" from "forgotten".

Each entry states the blocker, what was delivered anyway, and what unblocks it.
Nothing here is a substitute for an ADR — a decision goes in the log; this is
only the register of what that decision could not reach.

---

## B-001 — Private connector consumes the published contract module

**Milestone:** M5. This was M5's stated acceptance criterion:

> the private connector repository compiles against a published contract version
> with no filesystem path dependency on this repository.

**RESOLVED.** The consumer is no longer empty. Two of its modules require
`github.com/FabioCaffarello/fdos/libs/contracts v0.3.0`, and neither those
`go.mod` files nor its `go.work` carries a `replace` directive. That is the
criterion, met. The module names are the private side's and are not repeated
here (`fdos:docs/disclosure.md`).

**Verified how, and how far.** By reading the consumer's committed `go.mod`,
`go.sum` and `go.work` through the GitHub API — the artifact channel, which is
the only coordination channel between the two repositories. FDOS has not built
that repository and cannot: it is private, and depending on anything inside it
would be the reverse edge the open-core boundary exists to prevent.

**What is still untested, and is now the whole of what remains.** Resolution
through a *private* module path — credentials, `GOPRIVATE`, a private proxy.
That is exercised by the consumer's own CI, and whether it passes there is not
observable from here. This repository's half stays proven by `make
consumer-check`, which builds a throwaway module against the published version
with `GOWORK=off` — no workspace, no `replace`, no local path.

**A naming correction.** This entry, `README.md` and ADR-0020 called the consumer
`financial-connectors`. It has since been renamed `fdos-connectors`; GitHub
redirects the old name, which is why nothing broke and nobody noticed. ADR-0020
is immutable and keeps the old name — that is the decision log working as
designed, not a defect in it.

---

## B-002 — Plugin conformance suite

> Now tracked as [#53](https://github.com/FabioCaffarello/fdos/issues/53).

**Blocked on:** an undecided ownership question. No longer on absence.

**Milestone:** M5 listed "plugin SDK skeleton + a conformance test suite private
connectors must pass".

**Why the original reasoning no longer holds.** It said there was no interface to
conform to, because the ports would be an M6 output and the `app` layer did not
exist (ADR-0013). M6 shipped and `libs/ledger/app` exists. Separately, the
consumer has defined a host↔plugin wire contract of its own, in its own
namespace, importing nothing from `fdos.*`.

**What is actually undecided.** Whether a plugin conformance suite is an FDOS
deliverable at all. A plugin conforms to a *plugin runtime*, and the runtime is
the consumer's; what a plugin owes FDOS is a well-formed fact, whose shape
`libs/ledger-wire` already checks. Both readings are defensible and there is no
written ecosystem boundary to settle them against — which is the real blocker,
and a more useful thing to record than the absence that used to be.

**What unblocks it:** the ecosystem boundary written down and ratified, giving
this question an owner instead of an assumption.

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

**Held closed through `fdos.ingest.v1` (M8).** The submission message was a new
concept with two representations — the wire shape a producer sends and the
admission command FDOS acts on — and for exactly one slice nothing proved they
agreed. `libs/ledger-wire` now maps them under the same two properties,
negative-tested two ways: a decoder that ignores reference bindings, and an
encoder that forgets the stream name.

The first is the useful one. Dropping references on decode is caught by the
**wire → command → wire** direction and would also have been caught the other
way here — but the direction that only reads is the one that stays silent when a
field is never learned at all, which is why both are asserted rather than one.

**B-003 is closed.** Every canonical concept now has a mechanism proving its two
definitions agree. What remains is scope rather than absence: a payload type
added without a conformance test would still drift, which is why
`libs/ledger-wire/README.md` makes the test part of the procedure for adding
one.

**Recorded in:** ADR-0018 Consequences, as the largest unpaid cost of that
decision.

---

## B-004 — Claude Code loads no agents from a fresh clone

> Now tracked as [#54](https://github.com/FabioCaffarello/fdos/issues/54).

**Blocked on:** the dotcontext export having no CLI. **The other half is now
answered, and the answer is no.**

**Milestone:** M2.5 / ADR-0019.

**Why it is blocked.** The export is an MCP call, so `make bootstrap` cannot run
it. Versioning the export was tried (ADR-0017) and reversed (ADR-0019): it
committed ten skills that were never in the reviewed roster.

**Mitigation, and its weakness.** `make doctor` reports the missing export. That
is rung 5 — it tells a person something and relies on them acting.

**The second unblocking path is closed, by measurement (M9).** This entry asked
for *"confirmation that `exportSkills` with `includeBuiltIn: false` produces a
tree matching `.context/` exactly — in which case versioning becomes correct and
ADR-0019 should be revisited."*

A `dryRun` export answers it:

| | |
|---|---|
| Skills in the reviewed roster (`.context/skills/`) | **7** |
| Skills exported with `includeBuiltIn: false` | **17** |
| Difference | **10** |

**The same ten.** `includeBuiltIn: false` excludes the tool's built-ins and
still adds three general skills plus the five PREVC workflow skills and two
tooling ones — none of which this repository reviewed. ADR-0019 was right, and
is now right by measurement rather than by memory. **It is not revisited.**

**A hazard found while measuring — and found the hard way, because `dryRun`
writes.**

The measurement above was taken with `dryRun: true`. It created **102 files
across four new top-level directories** — `.agents/`, `.codex/`, `.gemini/`,
`.windsurf/` — plus `.github/skills/`. The count it reported as `filesCreated`
was not a projection; it was a description of what it had already done.

`make contracts-check` caught it immediately: four directories with no README
front matter, and the gate went red. The tree was restored by hand and the gate
is green again.

Two things follow, and the second is the one that matters:

- Only `.claude/` is gitignored. The other targets are not, so the export dirties
  the working tree in a way the gate refuses — the export is not merely unhelpful
  here, it is a **broken build waiting for someone to run it.**
- **A `dryRun` that writes is worse than no `dryRun`**, because it invites
  exactly the "let me just preview this" that produced the mess. Anyone reaching
  for this tool in this repository should assume the flag does nothing and
  arrange to be able to undo the call.

**What unblocks it, and it is now only one thing:** a CLI that `make bootstrap`
can invoke, exporting the reviewed roster and nothing else.

## B-005 — Dependency review on pull requests

> Now tracked as [#55](https://github.com/FabioCaffarello/fdos/issues/55).

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

> Now tracked as [#56](https://github.com/FabioCaffarello/fdos/issues/56).

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

> Decided (RFC-0007 → ADR-0022); the open items are now tracked as
> [#57](https://github.com/FabioCaffarello/fdos/issues/57).

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

- ~~Nobody notices an unresolved claim.~~ **Closed.** `app.Ledger.UnresolvedClaims`
  answers which admitted claims resolve to no identity, at a required as-of.
  Asking does not resolve anything — minting on inspection would make the act of
  looking change the ledger, and an identity would come into existence because
  somebody ran a report. Tested by asking twice and requiring the same answer.

  What replaced it is narrower and undecided: **who mints, when, and on whose
  authority.** Admission deliberately cannot, so a claim waits until something
  with an owner acts on it. Nothing yet is that something.
- The claim vocabulary is an open string. `identity.NewClaim` now refuses a
  non-canonical *scheme*, closing the `"Ticker"` / `"ticker"` half at rung 1.
  The other half — `"ticker"` and `"symbol"` for the same concept — is
  vocabulary governance and no type can solve it.
- **Per-scheme canonicalisation is FDOS's, and is not done.** Corrected here
  from a stronger and false version of this entry, which claimed a producer
  rendering `"PETR4"` then `"PETR4 "` mints two entities silently.

  It does not. `MintFor` derives through `identity.Derive`, whose
  `canonicaliseSeed` collapses whitespace runs and folds case, so those two
  produce **one identity**. Measured, after asserting the opposite twice from
  reading `Resolve` and `NewClaim` without following `MintFor` through.

  What actually happens is milder and already named in `resolve.go`: `Claim.Equal`
  is byte equality by design, so the spaced claim does not *resolve* against the
  existing mint and minting again yields **two `EntityMinted` facts carrying one
  identity** — *a defect the ledger records rather than hides*, with `Resolve`
  deterministic on the first visible mint.

  **The real gap is narrower.** `canonicaliseSeed` is generic and knows nothing
  about schemes, so variation it cannot fold — suffixes, punctuation, internal
  spacing, `"ticker"` versus `"symbol"` — derives genuinely different identities.
  A per-scheme rule is semantics rather than shape, which puts it here rather
  than on producer discipline. Needs an RFC, and it is a correctness question
  about the canonical model rather than a robustness one about inputs.
- **Minting is deliberately not reachable from admission.**
  `app.Ledger.AcceptHoldingClaim` appends a claim and resolves nothing, so an
  identity never comes into existence because a stranger submitted something.
  Recorded because the alternative is cheap now and expensive later: once a
  producer depends on automatic minting, removing it is a change to what the
  ledger does. Who mints, when, and on whose authority is undecided.
- `HoldingObserved`'s provenance must be `Derived`. proto3 cannot express it and
  the Go domain type does not yet enforce it, so it stays rung 6 exactly as
  ADR-0022 recorded.
- No `IdentifierAssertion` codec. Nothing produces one yet, and adding a codec
  ahead of a producer would be a conformance test with no subject.

---

## B-008 — No release has ever been published

**Blocked on:** nothing. This is a defect, recorded here because the register is
where "decided, not achieved" lives and nowhere else would be read.

**Milestone:** M3 delivered the pipeline, SBOM, provenance attestation and
signing, and the roadmap marks it complete. The machinery exists and is correct.
It has never run to completion.

**The finding.** Fourteen tags are published. The release workflow has run
fourteen times and **failed fourteen times**. Zero GitHub releases exist.

Consequently no published version carries an SBOM, a build-provenance
attestation, a cosign signature, or release notes. `make consumer-check` is a
step *inside* that workflow, so the published module has never been proven
consumable on a tagged commit — only on pull requests, against `main`.

**Why nobody noticed.** A tag push blocks nothing. `verify.yml` gates pull
requests and is green; `release.yml` fires afterwards, fails in about twenty
seconds, and tells no one. The gap between "CI is green" and "the release
happened" was never instrumented.

**What is not affected.** Module resolution. The Go proxy serves a module from
its tag and does not care whether a GitHub Release exists, which is why
`fdos-connectors` builds against `libs/contracts v0.3.0` today. The supply-chain
evidence is missing; the artifact is not.

**The cause,** from the run log of `libs/ledger-wire/v0.2.0`:

```
golangci-lint: not installed (pinned 2.12.2) — required by the current milestone
gitleaks: not installed (pinned 8.30.0) — required by the current milestone
FAIL: 2 toolchain violation(s).
make: *** [Makefile:71: toolchain-check] Error 1
```

The release job sets up Go and then calls `make verify`, but never installs the
rest of the pinned toolchain — unlike `verify.yml`, which does. So every release
dies at the first check.

**First fix.** Both workflows now use one composite action,
`.github/actions/setup-toolchain`, which installs everything `make verify`
needs. Copying the missing steps into `release.yml` would have worked too and
was rejected: a pruned copy of the setup is precisely how this happened.

**RESOLVED, and proven rather than asserted.** A disposable tag —
`libs/release-smoke/v0.0.0-rc.*`, a path that is not a Go module, so the proxy
can neither serve nor cache it — ran the pipeline end to end. Suggested by
`fdos-connectors` in [fdos#26](https://github.com/FabioCaffarello/fdos/issues/26)
on exactly the grounds that a workflow whose later steps have never executed is
not a working workflow.

It paid for itself on the first attempt by failing at step nine.

**The second defect, which only a real run could find.** `cosign` was installed
with no version. The run took whichever release was newest at that moment — one
published under an hour earlier — and that version had made `--output-signature`
a silently-ignored no-op under the new bundle format:

```
WARNING: --output-signature is deprecated when using --new-bundle-format and will be ignored
Error: signing dist/SHA256SUMS: create bundle file: open : no such file or directory
```

Every other tool in this repository reads its version from `mise.toml`. This one
read it from whatever sigstore had shipped that morning — in the step that
decides what "signed" means. ADR-0014 says every build input is pinned; this
input was not pinned anywhere, and no commit here could have predicted the
break.

`cosign` is now pinned in `mise.toml` and the manifest is signed to a **bundle**,
which carries signature, certificate and transparency-log entry together so a
consumer verifies without separately fetching the certificate.

**Second attempt: green.** The release carried four binaries, an SPDX SBOM,
`SHA256SUMS`, the cosign bundle, and two build-provenance attestations.

**Cost incurred, and not recoverable by me.** The smoke release was deleted; its
two tags could not be. The `release-tags` ruleset refuses tag deletion — which is
correct policy working as designed, and was not accounted for when the disposable
tag was proposed. `libs/release-smoke/v0.0.0-rc.1` and `-rc.2` are therefore
permanent. They name a path that is not a module, so no real module's version
list is affected; they are clutter in `git tag`, nothing more. Removing them
needs an admin bypass of the ruleset and is the repository owner's call.

**Still worth doing, and not done here:** making a failed release *visible*. The
silence is what let this run for fourteen tags — a tag push gates nothing, and
nothing announced the failure. Now that the pipeline works, the next failure
will be just as quiet.

**Not back-filled.** The fourteen existing tags remain without releases. They are
resolvable through the Go proxy, which is what consumers actually need, and
re-tagging published versions to attach supply-chain evidence after the fact is
a decision about what an attestation means — not a repair.

---

## B-009 — The governance corpus is vendored, pinned to nothing

**RESOLVED.** `fdos-connectors` vendors `invariants.md` and `boundary.md` pinned
to `ecosystem/v0.1.0` and byte-compared, recorded in `fdos-connectors:ADR-0026`. It also
tracks the corpus pin *separately* from the platform pin, on the reasoning that
one pin for both would couple a Tier-0 amendment to an unrelated script change —
a distinction this repository had not thought of.

**Why it matters.** `fdos-connectors` vendors the Constitution byte-for-byte and
keeps a manifest of inherited enforcement scripts, with its own drift check
against them. That is more discipline than this repository asked for. But it
vendored from `main`, at no version, because there had never been a version to
vendor.

An unpinned vendor cannot distinguish "upstream changed deliberately" from
"upstream changed by accident". The drift check fires either way, and the only
available response is to re-copy — which makes an accidental change downstream's
problem to absorb rather than upstream's to justify.

**The publishing half is done.**

| | |
|---|---|
| Corpus tag | `ecosystem/v0.1.0` |
| Announced | [fdos-connectors#2](https://github.com/FabioCaffarello/fdos-connectors/issues/2) — vendoring instructions, Tier-0 obligations, the five open disputes |
| Mirror | [fdos#20](https://github.com/FabioCaffarello/fdos/issues/20) |

The tag deliberately does not match `libs/*/v*`, so it does not trigger
`release.yml`. The corpus is documentation, not a module: there is nothing to
build, sign or attest, and firing a release pipeline at it would produce an
artifact nobody consumes.

**Why this entry stays open.** The vendoring is `fdos-connectors`' to do, and
this repository must not record it as done on their behalf — the same reasoning
[ADR-0024](adr/0024-contract-lifecycle-and-versioning.md) applies to consumer
migrations at step 6. It is also unobservable from here in the way that matters:
FDOS can see that files were copied, but not that the drift check covers them.

**What closes it:** `fdos-connectors` vendoring `invariants.md` and
`boundary.md` at the pinned tag, recording the tag in its `docs/upstream.md`,
extending its platform drift check to both files, and closing its own issue.
