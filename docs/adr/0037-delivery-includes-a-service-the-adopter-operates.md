---
id: ADR-0037
title: Delivery includes a service the adopter operates, and FDOS operates none
status: Accepted
date: 2026-08-07
deciders:
  - "@FabioCaffarello"
supersedes:
  - ADR-0029
superseded_by: []
---

# ADR-0037 — Delivery includes a service the adopter operates, and FDOS operates none

## Context

Records what [RFC-0015](../rfc/0015-the-submission-service-and-the-admission-race.md)
settled about delivery, in answer to the M11 gate
([PR #63](https://github.com/FabioCaffarello/fdos/pull/63)).

[ADR-0029](0029-the-public-surface-receives-a-claim.md) decided three things.
Two of them stand and are carried forward below unchanged. The third said:

> ### Delivery is a library, never a service FDOS operates
>
> **A service re-fails `E9`.** If ingesting requires calling something running on
> FDOS infrastructure, the open core is not usable alone — it is usable for as
> long as FDOS keeps the lights on.

and its alternatives section said, without the qualifier:

> **A service.** Rejected: it recreates the dependency `E9` exists to eliminate.

**The two sentences do not say the same thing, and M11 sits in the gap.** The
decision body rejects a service *FDOS operates* — the failure mode is a
dependency on somebody else keeping the lights on. The alternatives line rejects
*a service*. An open-source binary in `apps/` that an adopter runs on their own
machine fails the second and not the first.

Three further facts, none decisive alone:

- `apps/README.md` has listed *"Transport bootstrapping (HTTP, gRPC, CLI,
  workers)"* as allowed since M2. The directory contract already anticipates
  this; it predates ADR-0029 and neither cites the other.
- [`../ecosystem/roadmap.md`](../ecosystem/roadmap.md) has scheduled M11 as *"a
  submission service over `fdos.ingest.v1`"* since before ADR-0029 was accepted,
  and ADR-0029 did not edit it.
- ADR-0029's own enforcement table recorded every row at *none*, on the grounds
  that nothing existed to violate. Two of those rows are now built.

### Review changed the shape of this decision

RFC-0015 recommended a **narrowing** — a new ADR recording that the clause means
*a service FDOS operates*, with ADR-0029 keeping `status: Accepted`. The
argument for it: the decision body already carries the qualifier, so reading it
that way restates rather than reverses, and RFC-0008 with
[ADR-0026](0026-canonical-contracts-and-language-toolchains.md) is this
repository's precedent for narrowing rather than superseding.

**That recommendation was not taken.** Supersession was.

The argument that won: an accepted decision whose alternatives section flatly
rejects *"a service"* cannot be read into permitting one without the reading
itself being a decision. A narrowing would leave `status: Accepted` on a
document that, read start to finish, refuses what this repository is about to
build — and the next reader would have to know that a *later* ADR reinterprets
it, from a document that does not say so. ADR-0000 provides exactly one
mechanism for a decision that no longer holds as written, and it is this one.

Recorded because CONTRIBUTING requires it: a decision log holding only the
winner cannot distinguish a considered rejection from an option nobody thought
of. The narrowing was a real candidate, it was the author's recommendation, and
it lost.

**The cost is stated rather than discovered.** Superseding ADR-0029 supersedes a
decision most of which stands. That is why §1 and §2 below restate it rather
than referring to it: a decision left only in a superseded document is a
decision that will be read as withdrawn.

## Decision

### §1 — The public surface receives a claim *(carried forward from ADR-0029, unchanged)*

An external producer asserts identifiers — `{scheme, value}`, verbatim as it read
them — and FDOS resolves them internally, where
[ADR-0007](0007-internal-deterministic-identity.md) puts resolution and where the
resulting mint is itself a recorded fact
([ADR-0022](0022-minting-an-identity-is-a-fact.md)).

A producer never mints, never guesses, and never needs to. This is why
`app.Ledger.AcceptHoldingClaim` exists and why every other entry point — which
takes an already-resolved `identity.ID` — was structurally unusable from outside.

### §2 — The library builds; the ledger admits *(carried forward from ADR-0029, unchanged)*

**Admissibility never lives in linked code.** A third party can link a modified
build and post whatever it likes. Any library FDOS publishes is a **constructor
that makes conforming easy — never a gate that prevents non-conforming.**

The ledger revalidates everything, every time, **as though the producer were
hostile**, because eventually one will be. Every check a library performs is a
convenience; every check that matters is repeated at admission, on the ledger's
side of the boundary, assuming nothing about what the caller ran.

**The conformance kit tests the producer, not the ledger.** If passing it ever
becomes the evidence that the ledger will accept a fact, the truth boundary has
left this repository — through the door marked "developer experience", which is
the only door it was ever likely to leave by.

**This survives a service unchanged**, and it is the reason a service is
tolerable at all. A listener does not relax admission; it gives admission
callers.

### §3 — Delivery includes a service the adopter operates *(this is what changes)*

**FDOS publishes a submission service as a deployable application in `apps/`,
and FDOS operates nothing.**

The discriminator is *who runs it*, and it is not a technicality:

- **A service FDOS operates re-fails `E9`** — ingesting would require calling
  something on FDOS infrastructure, so the open core would be usable only while
  the lights stay on. That is the dependency the private repository created,
  wearing friendlier clothes, and an institutional adopter finds it in the first
  due-diligence pass. **This remains rejected, and nothing in this ADR softens
  it.** [ADR-0030](0030-the-submission-shape.md) §"What this does not decide"
  states the same thing and stays correct as written.
- **A binary an adopter runs is the opposite of that dependency.** The open core
  stays usable with FDOS absent, deleted, or out of business, which is the whole
  of what `E9` asks. Shipping the composition root is what turns *"anyone can
  ingest"* from a shape into a path.

The cheaper arguments for refusing a service — no transport decided, D2 open, no
persistence, no deployment — were true when ADR-0029 was written and are **not**
the reason it is now permitted. Persistence landed (ADR-0034, ADR-0035),
transport is decided in §4, and **D2 is still open** and is handled in §5 by
refusing to answer it rather than by answering it.

**The open-core line is unmoved:** anyone can ingest; doing it against a dozen
authenticated institutions is what you pay for.

### §4 — The transport is HTTP on the standard library

A single `POST` endpoint accepting a `fdos.ingest.v1.HoldingClaimSubmission` as
`application/x-protobuf`, on `net/http`, returning the assigned reference or a
typed error.

**Zero new dependencies**, which is the discriminating criterion and the same one
[ADR-0035](0035-the-sqlite-driver-and-its-provenance-risk.md) used to choose a
driver: build-input integrity first. gRPC would be the second heavy dependency in
the repository, arriving with the same audit obligation ADR-0035 paid for the
first, to serve one endpoint.

**The contract stays protobuf** ([ADR-0018](0018-contract-surface-is-protobuf.md)).
The body is the published message rather than a JSON re-expression of it, so a
producer in any language sends the bytes the conformance kit already produces.
**No wire type crosses into the domain:** `ledgerwire.DecodeHoldingClaimSubmission`
returns an `app.AcceptHoldingClaimCommand`, and that is the only value the
handler passes on.

**gRPC stays additive.** The port is the application-layer use case, not the
handler, so a second front end is a second composition root when a consumer asks
for one. Choosing the smaller thing first is reversible.

### §5 — What the service does while D2 is open

**It refuses to answer D2 by accident.**

- It **binds to loopback by default.** A default listening on `0.0.0.0` would
  answer *"who may write to a stream"* in the direction of *anyone*, silently,
  and nobody would have written that decision down.
- It **refuses to start** when configured to listen off-loopback without an
  explicit acknowledgement flag by which the operator asserts that
  authentication sits in front of it. This is ADR-0028's `unmediated` pattern: an
  explicit assertion beats an absent one, and there is deliberately no value
  meaning *"I did not think about it"*.
- `apps/<name>/README.md` says so, and that front matter is a binding contract
  here rather than documentation.

**None of this is an answer to D2**
([#64](https://github.com/FabioCaffarello/fdos/issues/64)). Option A in that
issue — *the deployment boundary is the answer* — may well be the decision, and
if it is, it deserves an ADR rather than a flag default that grew into a policy.

## Consequences

### Positive

- **`E9` acquires a path that ships**, not merely one that exists in principle.
  ADR-0029 could only say the surface *should* be reachable; this puts a process
  in front of it.
- **The truth boundary is unmoved.** §2 survives verbatim, and admission
  revalidates as if the producer were hostile whether the caller is a test, the
  conformance kit, or a socket.
- **The dependency `E9` exists to eliminate is still eliminated**, and now
  explicitly rather than by refusing to build anything.
- **The decision log stops disagreeing with itself.** A reader of ADR-0029 alone
  would have concluded that M11 violates an accepted decision. That was true, and
  it is now false for a stated reason instead of a conversational one.

### Negative

- **A service is a support surface, and this one has an operator who is not
  FDOS.** Publishing it means owning its configuration, its failure modes and its
  upgrade story for adopters this repository will never meet — the same cost
  ADR-0029 named for a library, now with a process attached.
- **"Anyone can ingest" invites volume this repository still has no answer for.**
  Rate, abuse and identity of submitters are D2's, D2 is open, and §5 defers the
  question rather than answering it. A listener makes the gap reachable from a
  network for the first time.
- **Supersession costs more than it buys here, and that is accepted knowingly.**
  Most of ADR-0029 stands; this ADR supersedes a decision largely in order to
  keep it, and §1 and §2 are restatement rather than new thinking. The
  alternative was leaving `status: Accepted` on a document that refuses what the
  next milestone builds.
- **A superseded ADR is still linked from live decisions.** ADR-0030 and RFC-0012
  both cite ADR-0029's rejection of a service. Those citations remain *correct* —
  what they cite is the FDOS-operated case, which §3 keeps rejecting — but a
  reader now has to follow a banner to establish that. Recorded because it will
  look like drift.
- **The operator can still misconfigure it.** §5 makes the unsafe deployment
  loud rather than impossible. Nothing prevents an adopter from passing the
  acknowledgement flag and putting no authentication in front of it, and nothing
  in this repository could.

### Enforcement

ADR-0029's table recorded every row at *none*, correctly, because nothing
existed. Two rows have since been built. Restated honestly:

| Property | Rung today | Mechanism |
|---|---|---|
| The public surface accepts a claim | **1** | `AcceptHoldingClaimCommand` takes `identity.Claim`, not `identity.ID`; there is no field to put a minted identity in |
| The ledger revalidates as if the producer were hostile | **3** | `libs/ledger/app` admission tests: malformed `SourceRef`, incomplete envelope, inadmissible provenance |
| A producer can check its own conformance | **3** | `examples/ingest` — the kit runs in CI and restates no rules, calling the real admission path |
| `apps/` holds no logic | **2** | `make contracts-check` against the directory contract |
| The transport adds no dependency | **2** | `make tidy-check`, `make vuln-check` over an `apps/` module requiring only first-party modules and `google.golang.org/protobuf` |
| No wire type reaches the domain | **2** | the analysers; the handler's only outbound call takes an `app.AcceptHoldingClaimCommand` |
| The service does not listen off-loopback by accident | **3** | test: the default configuration binds loopback; off-loopback without the acknowledgement flag refuses to start |
| Only an authorised party may write to a stream | **none** | **D2 is open.** Unchanged by this decision, and it must not be read as changed |
| FDOS does not operate an ingestion service | **6** | review. Nothing in a repository can detect that somebody deployed it and pointed producers at it |

**Execution-context question.** The last row is the honest weak point and it is
the one this decision turns on. The distinction between *a service FDOS operates*
and *a service an adopter operates* is a fact about the world, not about the
tree, and no check can see it. What the repository can do is refuse to ship
anything that presumes an FDOS-run instance — no hosted endpoint in any default,
no FDOS hostname in any configuration, no registration step — and that is a
review obligation stated here so it can be reviewed against.

**Constitution §15 moves no principle up a rung.** The rows above are properties
of this decision, not of the fourteen principles.

## Alternatives considered

**A narrowing ADR, leaving ADR-0029 accepted.** Recommended by RFC-0015 and not
taken; the argument on both sides is in §Context above rather than compressed
here, because it is the reason this document has the shape it has.

**No service — M11 ships a CLI reading submission bytes from a file or stdin.**
Alternative F of RFC-0015. It satisfies ADR-0029 as written with no
interpretation required, delivers `apps/`, and defers D2 honestly. Rejected: it
also does not make D2 live, which is half of what the milestone exists for, and
it exercises none of the concurrency [ADR-0036](0036-knowledge-time-is-assigned-under-the-streams-write-lock.md)
was just written to fix. The deferral would be real and the question would return
unchanged the first time anyone wanted a socket. **Not a straw man — it is the
cheapest correct answer, and it loses on what it declines to learn.**

**A service FDOS operates.** Rejected, for exactly the reason ADR-0029 gave and
in its words: it recreates the dependency `E9` exists to eliminate, and cost
arguments lose to anyone willing to pay while that one does not.

**gRPC rather than HTTP.** The native fit for a protobuf contract, with
generated clients in every language a producer might use. Rejected on
dependency footprint: `google.golang.org/grpc` and its graph would be the second
heavy dependency in the repository, audited on the terms ADR-0035 established,
to serve one endpoint that `net/http` serves with none. Additive later, and the
port makes it cheap.

**Publishing a Go client library instead of a service.** Rejected by ADR-0030 and
its reasoning is unchanged: it serves Go producers only, for an invariant whose
words are *any* third party. The submission message already exists so that a
producer in any language has something to send; a Go client would be a
convenience on top, not a substitute.

## Notes

Accepted by @FabioCaffarello, who accepted RFC-0015 and chose supersession over
the narrowing that RFC recommended.

**The `SourceRef` rename does not block this.** ADR-0029 closed by saying the
rename obligation on `fdos.kernel.v2` *"binds this work: the public ingress must
not publish while `SourceRef` is still named `value`"*. It delegated that to
`roadmap.md`, and `roadmap.md` has since retracted the deadline in a section
headed *"The `content_hash` rename, and why it has no deadline"* — a rename can
only ride a major boundary, and a major boundary migrates every consumer by
construction, so third-party adoption of `v1` does not make it more expensive by
a single unit. **The obligation stands; the deadline was invented and is gone.**
Carried forward corrected rather than repeated, because the superseded text says
the opposite and someone will read it.

Open and deliberately not decided here:

- **D2** ([#64](https://github.com/FabioCaffarello/fdos/issues/64)) — who may
  write to a named stream. `risk/truth-path`-adjacent, and §5 refuses it by
  default rather than answering it.
- **Batch submission.** One fact per request. Partial success is a different
  admission semantics and wants its own decision.
- **TLS, rate limiting, quotas, observability, multi-instance deployment.** The
  first is the operator's under §5; the next two are D2's; the last is ADR-0034's
  open clock-skew question and ADR-0036 §"What this does not fix".
- **Crash-safety under real power loss** — ADR-0035's named gap. A running
  service invites a deployment story, and that story must not imply this is
  closed.
