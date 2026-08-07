---
id: ADR-0038
title: FDOS tracks the Go patch line, because operating a listener makes the standard library reachable
status: Accepted
date: 2026-08-07
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0038 — FDOS tracks the Go patch line, because operating a listener makes the standard library reachable

> **Accepted by @FabioCaffarello**, who instructed that the pin be raised and
> verified in CI. It was drafted as a proposal and accepted without amendment.

## Context

`apps/submitd` ([ADR-0037](0037-delivery-includes-a-service-the-adopter-operates.md))
is the first FDOS process that listens. `make verify` went red on it, on the one
check that had never fired:

```
>>> apps/submitd
GO-2026-5856  Encrypted Client Hello privacy leak in crypto/tls
              Found in crypto/tls@go1.26.2       Fixed in crypto/tls@go1.26.5
    #1: main.go:99:37: submitd.run calls http.Server.ListenAndServe,
        which eventually calls tls.Conn.HandshakeContext

GO-2026-5039  Arbitrary inputs included in errors without escaping in net/textproto
              Found in net/textproto@go1.26.2    Fixed in net/textproto@go1.26.4
    #1: main.go:99:37: submitd.run calls http.Server.ListenAndServe,
        which eventually calls textproto.Reader.ReadMIMEHeader

GO-2026-5037  Inefficient candidate hostname parsing in crypto/x509
              Found in crypto/x509@go1.26.2      Fixed in crypto/x509@go1.26.4
```

**Nothing in the handler causes this and no handler avoids it.** A process that
serves HTTP imports `net/http`, which reaches `crypto/tls`, `net/textproto` and
`crypto/x509` whether or not TLS is configured. Every module before this one was
a library that reached none of them, which is why `make vuln-check` has reported
seven clean modules since M10 and reports six clean and one red today.

### ADR-0035 predicted exactly this, in these words

Rejecting PostgreSQL, it recorded:

> **A networked store makes `crypto/x509` reachable.** `govulncheck` against pgx
> reports `GO-2026-5037` as reachable through TLS. It is fixable by moving the
> Go pin from 1.26.2 to 1.26.4, but **the structure survives the fix**: every
> future standard-library TLS advisory becomes a reachable finding, and
> `make verify` is the gate.

`GO-2026-5037` is one of the three above. The prediction arrived through a
listener rather than through a database, which changes nothing about it: what
made it true was FDOS acquiring a network surface, and the milestone that
acquired one was M11 rather than a Postgres adapter.

So this decision is not "raise a pin". **The pin is the consequence. The
decision is that FDOS now tracks somebody else's release schedule**, and that is
what ADR-0035 said would need deciding when it arrived.

### What was measured rather than assumed

The obvious reading of `Found in crypto/tls@go1.26.2` is that every module's
`go` directive must move, which would raise the minimum Go version for anyone
consuming `libs/contracts` — a change to somebody else's build
([ADR-0025](0025-consumer-facing-surface-is-the-contracts-module.md)), and the
sharp edge this repository is most careful about.

**It does not.** Measured against the tree with Go 1.26.5 as the toolchain and
every `go` directive left at `1.26.2`:

```
$ GOTOOLCHAIN=local GOWORK=off go1.26.5 run \
    golang.org/x/vuln/cmd/govulncheck@v1.6.0 -show=traces ./...
No vulnerabilities found.
```

The advisories are against the **standard library the toolchain ships**, not
against a declared language version. Moving `mise.toml` is sufficient and the
module directives stay where they are. That measurement is the difference
between a one-line change and a change to a downstream build, and it took one
download to establish.

## Decision

### 1. The pin moves to the patched toolchain

`mise.toml`: `go = "1.26.2"` becomes `go = "1.26.5"`. **Nothing else changes** —
no `go` directive in any `go.mod`, so no consumer of `libs/contracts` is
required to upgrade anything.

`mise.toml` is the single source of truth and CI reads it through
`.github/actions/setup-toolchain`, so the developer toolchain and the gate
cannot drift (B-008).

### 2. A reachable standard-library advisory is a gate failure, and the answer is the pin

Stated as a standing rule rather than left to be re-argued each time:

- **`make vuln-check` stays blocking**, and reachable findings are never
  allowlisted. ADR-0035 already settled the reasoning for the unreachable case —
  *"a scanner people learn to ignore enforces nothing"* — and it applies with
  more force to reachable ones.
- **The response is to move the pin to the fixed patch release**, not to work
  around the call path. A handler rewritten to dodge `net/textproto` would be
  dodging the scanner rather than the defect.
- **Unreachable findings stay recorded and unescalated**, exactly as ADR-0035
  decided for the driver's three.

### 3. What this does not decide

**No automation.** Whether a bot proposes patch bumps, and whether this
repository would accept a machine-authored change to its toolchain pin, is a
separate decision with a supply-chain argument of its own. Doing it by hand is
the status quo and this ADR does not change it.

**No policy on minor or major Go releases.** This is about the patch line. A
move from 1.26 to 1.27 changes language and library behaviour and is its own
decision.

## Consequences

### Positive

- Three reachable advisories become zero, and the first FDOS process that
  listens does not ship with a known TLS defect.
- **No downstream cost, measured rather than hoped.** Module `go` directives are
  untouched, so the private repository pinning `libs/contracts` is unaffected.
- One line, in the file that is already the single source of truth for both
  developers and CI.

### Negative

- **The gate now fails on Go's schedule rather than on FDOS's.** A standard
  library security release makes the next CI run red with nobody here having
  changed anything. That is a real operational dependency on an external
  cadence, it did not exist while every module was a library, and **it is
  permanent** — this pin bump buys the current three and nothing more.
- **Every developer must upgrade**, immediately. `toolchain-check` treats a
  wrong version as an error and says so deliberately: *"a wrong version is worse
  than no version"*. There is no grace period and this ADR does not add one.
- **`repro-check` digests change.** `fdoslint` currently builds reproducibly at
  `0b4d5f235ac5857a`; a different toolchain produces a different digest. That is
  correct behaviour and it invalidates any external record of the old one,
  including the signed manifests of releases built before the bump. Those
  releases stay verifiable against their own toolchain; nothing is re-derivable
  across the bump.
- **The `go` directives staying at 1.26.2 leaves a gap that only a check
  closes.** A developer with Go 1.26.2 on `PATH` builds a vulnerable binary,
  because `GOTOOLCHAIN` will not fetch a newer toolchain to satisfy a directive
  that does not ask for one. `toolchain-check` is the only thing that refuses,
  which puts this at rung 3 rather than rung 1. **Raising the directives would
  close it at rung 1 and would cost a downstream build**, and that trade is the
  one alternative below worth arguing with.
- **This cannot be validated on darwin.** A toolchain change is precisely the
  class ADR-0014's failure mode covers and M10 paid for: `CGO_ENABLED=0` was
  green on every local run and red on the first CI run to reach it. The bump is
  pushed and verified in CI before it is called green, and the measurement above
  was taken on darwin/arm64 and proves the advisory is cleared, not that the
  gate passes.

### Enforcement

| Rule | Rung | Mechanism |
|---|---|---|
| The installed toolchain matches the pin | 3 | `make toolchain-check`, in `verify` and in CI |
| No reachable vulnerability in any module | 3 | `make vuln-check`, blocking |
| Builds stay byte-reproducible across the bump | 3 | `make repro-check` — it re-establishes a digest, it does not preserve one |
| A developer cannot build with the vulnerable toolchain | **3** | `toolchain-check` only. See the gap above; rung 1 is available and costs a downstream build |
| FDOS notices the *next* advisory | **3** | `vuln-check` on every pull request — and **not** on a schedule, so a quiet week finds nothing |

**Execution-context question.** The last row is the honest weak point. Nothing
runs `govulncheck` unless somebody opens a pull request, so an advisory
published during a quiet period is discovered by the next contributor rather
than when it lands. That is the same shape as the gap this ADR closes and it is
not closed here.

## Alternatives considered

**Raise the `go` directive in every module as well.** It would close the last
enforcement gap at rung 1: a toolchain older than the directive is fetched or
refused by Go itself, with no check involved. Rejected because it is
**unnecessary** — measured above — and because it raises the minimum Go version
for every consumer of `libs/contracts`, which is a change to somebody else's
build made to buy a rung this repository can occupy with a check it already
runs. If the gap ever bites, this is the fix, and the cost is known.

**Bump to 1.26.4.** Clears `GO-2026-5039` and `GO-2026-5037` and leaves
`GO-2026-5856`, which is fixed in 1.26.5. Rejected: a bump that leaves a
reachable finding leaves the gate red, which is the same as not bumping.

**Allowlist the three advisories.** Rejected in ADR-0035's own words. A scanner
people learn to ignore enforces nothing, and the first suppression is the one
that teaches the habit.

**Ship a CLI instead of a listener** — revert ADR-0037 §3 and §4 to alternative
F of RFC-0015. **This is the only option that avoids the cadence dependency
entirely**, because a process that does not serve HTTP reaches none of these
packages, and it is recorded rather than dismissed because that is a real
property and not a trivial one.

Rejected because it chooses the milestone by what the gate finds convenient. The
exposure is not the transport's fault: any FDOS process that ever listens has
it, including the query surface and the MCP server that D2's register entry
already anticipates. Deferring it by not listening defers it once.

**Do nothing and accept three reachable advisories.** Rejected. `E9` invites any
third party to run this binary; shipping it with a known TLS defect and a green
badge is the failure the supply-chain posture exists to prevent.

## Notes

Accepted by @FabioCaffarello. Drafted as a proposal on their instruction and
accepted as written; the change is `mise.toml` and nothing else, pushed and
verified in CI before being called green.

Open and deliberately not decided here:

- automation of patch bumps, and whether a machine-authored toolchain change is
  acceptable in a repository whose posture starts at build-input integrity;
- whether `vuln-check` should also run on a schedule, so an advisory is found
  when it is published rather than when somebody next opens a pull request;
- whether the `go` directives should eventually move for the rung-1 guarantee,
  which is a conversation with the one consumer rather than a decision here;
- the Go minor-version policy, which this ADR does not touch.

`main` is green today because it holds no listener. The red is on
`feat/m11-submitd`, which is where the first one is.
