---
directory: submitd
purpose: The submission service — receives fdos.ingest.v1.HoldingClaimSubmission over HTTP and admits it to the ledger. Composition root only.
owner: "@FabioCaffarello"
allowed:
  - Flag parsing, configuration and process lifecycle
  - Construction of libs/ledger-sqlite, libs/ledger and the clock, wired into app.Ledger
  - HTTP transport — routing, decoding the published message, mapping domain errors onto status codes
  - Tests that exercise the handler through net/http/httptest
forbidden:
  - Business rules, admission criteria or financial calculations of any kind
  - Re-implementing or short-circuiting any check app.Ledger performs
  - Canonical model definitions, or any type that outlives this process
  - Authentication, authorisation or any answer to D2
  - Imports from another application
---

# submitd

The first composition root in `apps/` (ADR-0037). It exists so that a producer
outside this process can submit a claim, which until M11 required being a Go
test in the same binary.

## What it is

```
POST /v1/holding-claim-submissions
Content-Type: application/x-protobuf
body: a serialised fdos.ingest.v1.HoldingClaimSubmission
```

On success, `201 Created` and the assigned reference — `stream#sequence` — as
`text/plain`.

**The response is not a published message**, and that is a deliberate gap.
`fdos.ledger.v1` has no `Ref` message, and adding one is a change to a contract
surface consumed outside this repository — an issue and an RFC, not a
convenience taken while writing a handler (ADR-0024, ADR-0025). Until then the
reference travels as text and a consumer that needs it parses one delimiter.

## What it is not

**It does not decide anything.** Every check that matters is `app.Ledger`'s, and
it performs them all again whatever the caller ran (ADR-0037 §2). This binary
decodes bytes, calls one use case, and maps its errors onto status codes. If a
rule about admission appears in this directory, it is in the wrong place — and
it would be unreachable by the analysers, by the conformance kit, and by every
test that does not start a process.

**It does not authenticate anybody.** D2 — who may write to a named stream — is
open ([fdos#64](https://github.com/FabioCaffarello/fdos/issues/64)), and nothing
here answers it.

## Running it

```sh
submitd -store /var/lib/fdos/ledger.db
```

It listens on `127.0.0.1:8080` by default. **That default is load-bearing.** A
service listening on every interface would answer *"who may write to a stream"*
in the direction of *anyone*, silently, and nobody would have decided it.

To listen anywhere else you must also pass `-callers-are-authenticated`, by
which the operator asserts that authentication sits in front of this process.
There is deliberately no value meaning *"I did not think about it"* — the same
reasoning that produced the `unmediated` sentinel in ADR-0028.

## More than one process

Another process may hold the same database. **That was not true when this binary
shipped, and this file said so** — the claim is corrected here rather than
quietly dropped.

[ADR-0036](../../docs/adr/0036-knowledge-time-is-assigned-under-the-streams-write-lock.md)
serialised writers inside one process, so a second `submitd` — or an operator
CLI beside it — reopened the clock-read/append window that decision had closed.
[ADR-0041](../../docs/adr/0041-the-write-path-serialises-in-the-store.md)
superseded it: mutual exclusion moved into the `app.Store` port as
`Serialise(ctx, name, fn)`, `libs/ledger` reads the clock *inside* that region,
and `libs/ledger-sqlite` implements it as a transaction that spans processes.
This binary carries both versions.

**What that buys is ordering, not throughput.** On SQLite the region is guarded
by a database-wide lock, so concurrent writers serialise on it — ADR-0041
records that as a cost, and ADR-0042's second engine is what exists to pay it
down. Two processes are now *correct*, not fast.

## What it does not promise

- **Crash-safety under real power loss.** ADR-0035 records this as an open gap in
  the storage layer and it is not closed here. Do not read "durable" as
  "survives having the plug pulled".
- **Throughput under concurrent writers.** See above: the region is serialised
  and on SQLite it is database-wide. No figure is promised, and a reader can be
  blocked behind a writer for the length of the region.
- **Rate limiting, quotas or abuse handling.** D2's.
