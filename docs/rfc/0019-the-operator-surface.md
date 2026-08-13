---
id: RFC-0019
title: The operator surface — six use cases with no caller, and what a human mint records
status: Draft
date: 2026-08-13
authors:
  - "@FabioCaffarello"
---

# RFC-0019 — The operator surface: six use cases with no caller, and what a human mint records

## Summary

`libs/ledger/app` exports seven use cases. **One has a caller in production.**
`submitd` calls `AcceptHoldingClaim`; `MintIdentity`, `UnresolvedClaims`,
`ProjectPosition`, `ObserveClaimedHolding`, `ObserveHolding` and `CorrectFact`
have none. Every fact the platform admits today is a claim that will never be
resolved, and nothing tells anybody.

This proposes the surface that gives three of them a caller — an operator CLI,
`fdosctl`, exposing `unresolved` → `mint` → `position`.

It is an RFC rather than an ADR because three of its decisions have real
alternatives that code would otherwise settle silently: **what the CLI talks
to**, now that ADR-0041 made a second process legitimate; **what an operator
must type for a bitemporal read**, given that `AsOf` has no default anywhere in
the app layer and must not acquire one here; and **what a human mint records as
its authority**, when FDOS has no actor model and the field is appended forever.

## Motivation

### The measurement

```
AcceptHoldingClaim      submitd            1 caller
MintIdentity            —                  0
UnresolvedClaims        —                  0
ProjectPosition         —                  0
ObserveClaimedHolding   —                  0
ObserveHolding          —                  0
CorrectFact             —                  0
```

M8's third deliverable was *an unresolved claim is observable*. It was
delivered: `domain.Unresolved` exists, `app.UnresolvedClaims` wraps it, and both
are tested. What was delivered is observability **in principle** — a function
that would tell somebody, if anything called it.

The gap is not a missing feature in `libs/`. It is that the platform has no way
to show its own state, which is the same sentence as *the local stack has
nothing to demonstrate*. M13 and the compose stack downstream both terminate in
this surface: `docker compose up` that admits a claim into silence has
demonstrated plumbing, not a ledger.

### What is at stake

[ADR-0012](../adr/0012-explained-return-type.md) makes explanation a return
type rather than a report. [ADR-0033](../adr/0033-minting-is-an-owned-act-and-canonicalisation-is-per-scheme.md)
makes minting an owned act with a named authority. Neither property has ever
been exercised by a human, because no human-facing surface exists. A property
only tested by tests is a property whose ergonomics nobody has priced, and
ergonomics is where this class of rule is eroded — by the convenience flag added
during a demo.

### What is retrofittable, and what is not

Most of a CLI is retrofittable: flags can be renamed and verbs added.

Two parts are not.

- **What a human mint records is appended and immutable.** An `EntityMinted`
  fact carries its interpreter and confidence forever. Getting that wrong does
  not produce a bad CLI; it produces a ledger whose mints cannot be attributed,
  and the ledger is append-only.
- **A default `AsOf` produces answers that look right.** A wrong flag name is
  discovered immediately. A projection silently taken at *now* while the
  operator believed they asked about a past coordinate is look-ahead
  contamination, and it is indistinguishable from a correct answer at the
  terminal.

Those two are why this document exists before `apps/fdosctl` does.

## Design

### §1 Three verbs, in the order the loop runs

```sh
fdosctl unresolved  --stream S --effective T --knowledge T
fdosctl mint        --stream S --kind K --claim scheme:value --interpreter I ...
fdosctl position    --stream S --account A --instrument I --effective T --knowledge T
```

Flag spellings here are illustrative; what this RFC fixes is which arguments are
**mandatory**, which have **no default**, and which must **never exist**.

The first cut stops at three verbs. `ObserveHolding`, `ObserveClaimedHolding`
and `CorrectFact` are deliberately absent — see §6.

### §2 What the CLI talks to

**This is the fork the RFC exists for.**

- **(a) The store, directly.** `fdosctl` opens the same database `submitd`
  writes to.
- **(b) `submitd`, as a client.** The CLI holds no store and speaks HTTP.

Until recently (b) was close to forced. ADR-0036 serialised writers *inside one
process*, so a second process against one database reopened exactly the ordering
window that decision closed — `apps/submitd/README.md` still says so, and
[#147](https://github.com/FabioCaffarello/fdos/issues/147) is the correction.

That reason is gone. [ADR-0041](../adr/0041-the-write-path-serialises-in-the-store.md)
moved mutual exclusion into the `app.Store` port as `Serialise(ctx, name, fn)`,
`libs/ledger v0.9.0` reads the clock *inside* the region, and
`libs/ledger-sqlite v0.4.0` implements it as a transaction that spans processes.
A second process is now correct for a recorded reason rather than by luck.

**This proposes (a)**, and the argument splits by path:

- **The read path never needed a service.** `UnresolvedClaims` and
  `ProjectPosition` call `Load` and derive; they append nothing and serialise
  nothing. Routing them through `submitd` would require query messages on
  `fdos.ledger.v1` — a change to a contract surface consumed outside this
  repository, carrying ADR-0024's full workflow, to reach a function that
  touches no shared state.
- **The write path is one act, and it is local by design.** `MintIdentity` is
  the only thing in the repository that appends `EntityMinted`, and ADR-0033
  makes it owned. Putting it behind a network surface with no actor model
  publishes a minting endpoint before D2 answers who may call it.

The costs of (a), stated rather than discovered:

1. Two binaries open the ledger. That is safe under ADR-0041 and it is still two
   binaries; a bug in `Serialise` now has a second caller.
2. The operator's process must reach the database. In compose that is a shared
   volume; on a remote deployment it is a reason to revisit this decision, and
   the revisit is a new RFC rather than a flag.
3. When ADR-0042's Postgres engine lands, "the file" becomes "the DSN". The CLI
   follows the same configuration the service uses — see Open questions.

### §3 A bitemporal read has no default, and the CLI must not invent one

Both queries require `temporal.AsOf`, and the app layer is explicit about why:

> `AsOf` is required and has no default. A projection that silently means "now"
> is how look-ahead bias enters an analysis, so there is no overload without it.

A CLI is where that discipline is cheapest to lose. Two full timestamps per
read is real friction, and friction produces a wrapper script that fills them
in — the same defect, one layer out, where no analyser will ever see it.

**Proposal: both coordinates are mandatory, and `now` is a value the operator
types.**

```sh
fdosctl position --stream demo --account … --instrument … \
        --effective 2026-08-13T00:00:00Z --knowledge now
```

`now` is accepted as a literal, resolved by the CLI, and **printed back in the
output header as the instant it resolved to**. The distinction from a default is
not cosmetic: the invocation records what was asked, so the shell history and
the transcript both say which coordinate produced the number.

There is deliberately **no `--now` shorthand** setting both coordinates, and no
mode in which a missing coordinate is inferred. A shorthand would make the
common case the one that hides the coordinate, which is the ergonomics that
erodes the rule.

### §4 What a human mint records as its authority

`MintIdentityCommand` carries `Source`, `CollectedAt`, `Interpreter`,
`Confidence` and `References`, and the app layer says plainly what they are
worth:

> They are **recorded, never checked** — FDOS cannot verify that the named
> authority is who it says or that it was entitled to mint, because there is no
> actor model and building one is D2. ADR-0033 records that at rung 6, and the
> boundary today is the process boundary: whoever can call this can mint.

A CLI turns *the process boundary* into *a shell prompt*. That does not weaken
the guarantee — it was already this weak — but it is the first time the weakness
is reachable by a human hand, and what that hand writes is appended forever.

**Proposal:**

- **`--interpreter` is mandatory and has no default.** It names the human or the
  runbook that decided. There is deliberately no value meaning *"I did not think
  about it"* — the same shape as `unmediated` in
  [ADR-0028](../adr/0028-provenance-admissibility.md) and
  `-callers-are-authenticated` in `submitd`.
- **The CLI never names itself as the interpreter.** A ledger whose mints are
  all attributed to `fdosctl` records the tool and loses the fact anyone would
  later want to audit.
- **`--confidence` is mandatory.** A default confidence is an assertion nobody
  made.
- The mint's provenance stays `Derived`, assembled by the use case from the
  claim, the ruleset and the fact answered. The CLI supplies the human half and
  fabricates none of the derived half.

**What the surface must never offer**, because each is a plausible convenience
that ends the ownership property:

- `--all` or any batch mode over `unresolved` output. Minting the list is
  minting on inspection, one indirection away from the thing
  `domain.Unresolved` refuses to do.
- `--yes`, or any non-interactive confirmation bypass, when the invocation is
  not already fully explicit.
- A retry that could mint twice. `ErrAlreadyMinted` names the existing identity;
  the CLI surfaces it as a refusal and never swallows it.

### §5 The output carries the explanation, or it is not the output

`ProjectPosition` returns `explained.Value[domain.Position]`: the answer travels
with the derivation that produced it (ADR-0012).

Two audiences, and both get the whole value:

- **A terminal.** The default rendering prints the position *and* its
  derivation. Long is acceptable; a paged or `--brief` rendering that elides
  *steps* may be added later, and it elides steps, never the derivation's
  existence.
- **A machine.** `--output json` emits the full explained value, and this is
  what the compose e2e asserts against.

There is **no mode that prints the value alone**. A `--quiet` returning the
scalar is the single flag that would let the repository's central property be
discarded at exactly the surface by which people judge whether it is real.

### §6 Scope

**Not covered, and each for a reason:**

- **Authentication and authorisation.** D2
  ([#64](https://github.com/FabioCaffarello/fdos/issues/64)). This binary
  answers none of it, exactly as `submitd` answers none of it.
- **`CorrectFact`.** A correction typed at a shell with no review is the least
  reversible act the system has: it appends, so a mistaken correction is
  permanent and needs its own correction. It wants its own decision about what
  review looks like, and inheriting a CLI's ergonomics is not that decision.
- **`ObserveHolding` and `ObserveClaimedHolding`.** Both presume an identity the
  operator would have to supply by hand. They are the derivation path, and the
  derivation path's caller is a service, not a person.
- **Submitting claims.** That is `submitd`'s, and it stays there.
- **Stream discovery.** `--stream` is required and there is no listing verb yet;
  see Open questions.

## Enforcement

Constitution §15's ladder, per rule, honestly:

| Rule | Rung | Mechanism |
|---|---|---|
| No invocation succeeds without both coordinates | 2 — test | Negative test: each read verb exits non-zero with neither coordinate, and with only one |
| `now` is echoed as the instant it resolved to | 2 — test | Golden output asserts the header carries an instant, not the literal |
| `--interpreter` and `--confidence` are mandatory | 2 — test | Negative test per flag |
| No output mode omits the derivation | 2 — test | Every rendering is asserted to contain it; a new mode fails the suite by default |
| No batch mint | 6 — discipline, plus review | Nothing prevents it being added; the test suite cannot assert the absence of a flag nobody wrote |
| The CLI never names itself interpreter | 2 — test | The default is absent, so the assertion is that no default exists |
| Whoever can run the binary can mint | 6 — by construction | D2. Recorded, not solved |

The two rung-6 rows are the honest part. A surface's *absences* are largely
unenforceable, and this document is the record of which absences are deliberate
so that a future addition is visibly a reversal rather than an oversight.

## Alternatives

1. **Expose the use cases as HTTP endpoints on `submitd`.** Genuine, and it is
   how most systems do this. Rejected: it turns reads into a published contract
   obligation on a module an external repository pins, and it publishes a
   minting endpoint while D2 is open. The write half is the disqualifier; the
   read half alone would be arguable.

2. **A CLI that speaks to `submitd` for everything** (§2 option b). Genuine: one
   process holds the store, which is the simplest possible story about
   concurrency, and it survives a deployment where the operator cannot reach the
   database. Rejected *for now*: it costs a published query surface and makes
   the operator dependent on a running service to answer a question that touches
   no shared state. ADR-0041 is what makes this a choice rather than the only
   option, and if remote operation becomes real this is where to return.

3. **Extend `examples/ingest` instead of shipping a binary.** Rejected:
   `examples/` demonstrates contracts. An operator tool that is an example is
   one nobody can install, and it would not even be covered by the gate —
   [#79](https://github.com/FabioCaffarello/fdos/issues/79) is open precisely
   because `examples/` sits outside every verify target.

4. **Wait for D2.** Rejected: D2 governs who may *write*. Reading unresolved
   claims needs no actor model at all, and the mint boundary is already recorded
   at rung 6 by ADR-0033 — a CLI does not lower it. Waiting keeps the platform
   unable to show its own state in exchange for a decision that would change
   only the mint verb.

## Prior art

**Plumbing and porcelain.** Git's split is the standard answer to "one surface
for humans, one for scripts", and §5's `--output json` is the porcelain-safe
half. The lesson worth taking is the failure: git's human output changed shape
and scripts parsed it anyway, which is why the machine format here is the
asserted one and the human one is not a contract.

**Bitemporal stores that refuse a default basis.** Systems in this family
(asserted-versioning designs, and Datomic/XTDB-style `as-of` bases) consistently
make the basis explicit at the query API and consistently grow a convenience
layer that defaults it, after which the discipline lives only in the layer
nobody uses. §3's answer — accept `now` as a *typed literal*, echo what it
resolved to, and refuse to infer — is an attempt to keep the ergonomics without
the default.

**Accounting consoles.** Double-entry systems have had operator consoles for
decades, and the durable rule from them is that a posting is attributed to a
person. §4 is that rule under a system that cannot verify the attribution, which
is why the answer is *record it explicitly* rather than *check it*.

## Open questions

- **Engine selection when ADR-0042's Postgres lands.** Does `-store` become a
  DSN, and does the CLI grow an engine flag? Whatever the answer, it must
  satisfy `deploy/`'s standing constraint: configuration selects infrastructure,
  never behaviour. Resolved by whoever implements the second engine's wiring.
- **Stream discovery.** `--stream` is mandatory with no listing verb, so an
  operator must know the name. A `streams` verb is plausible and is deferred:
  it is a read with no coordinate, and what it means to enumerate streams
  as-of a coordinate is not obvious. Resolved when the compose demo shows
  whether operators actually get stuck.
- **Whether `--interpreter` is ever checked.** Today it is recorded. If D2
  produces an actor model, this is the field that would gain a check, and the
  ADR that adds one supersedes §4's second bullet rather than quietly
  reinterpreting it.

## Consequences

**Easier.** The platform can show its own state, so the compose stack has a
payoff rather than a green log. Three use cases gain a caller and therefore a
test that exercises them the way a person will. The mint loop becomes
demonstrable, which is the precondition for anyone judging whether ADR-0033's
ownership rule is workable.

**Harder.** Two binaries open the ledger. Every read costs two explicit
coordinates, forever, and that will be the most complained-about property of
this tool.

**Impossible.** Minting unattended from a list. Printing a position without its
derivation. Asking a bitemporal question without saying when.
