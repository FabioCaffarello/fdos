# Issue and pull-request labels

The label taxonomy used across the FDOS ecosystem. Written down because labels
that exist only on GitHub are configuration nobody can review, diff, or restore
— and because two repositories cannot agree on a vocabulary neither of them
records.

**Namespaced, and mutually exclusive within a namespace.** Two `type/` labels on
one issue means nobody decided what it is. GitHub cannot enforce that; it is
rung 6.

## Shared namespaces

Identical in `fdos` and `fdos-connectors`. These describe process, and process
is ecosystem policy rather than per-repository preference.

### `type/` — what kind of work this is

| Label | Meaning |
|---|---|
| `type/feature` | New capability |
| `type/bug` | Something behaves wrongly |
| `type/chore` | Maintenance with no behaviour change |
| `type/docs` | Documentation, decisions and directory contracts |
| `type/rfc` | Design exploration, before a decision can be made |
| `type/adr` | Records a decision already reached |
| `type/spike` | Time-boxed investigation; the output is knowledge |
| `type/enforcement` | Adds or changes a mechanism — needs a negative test |

### `status/` — where it is, not who has it

`status/triage` · `status/blocked` · `status/in-progress` ·
`status/awaiting-review` · `status/awaiting-decision`

`status/blocked` requires the blocker to be named in the body. A blocked issue
that does not say what it is waiting on is indistinguishable from a forgotten
one — which is the same argument that produced [`../blocked.md`](../blocked.md).

### `contract/` — changes to the published surface

`contract/change` · `contract/breaking` · `contract/deprecation` ·
`contract/migration`

**`contract/breaking` requires a human decision and is never self-approved.** It
carries the full workflow in
[ADR-0024](../adr/0024-contract-lifecycle-and-versioning.md): an RFC, a consumer
issue opened *before* the change merges, both versions valid, and an N-1 window
of at least one milestone.

### `xrepo/` — the dependency graph between repositories

`xrepo/blocks` · `xrepo/blocked-by` · `xrepo/mirror`

Every cross-repository edge gets an issue on the acting side and a mirror on the
other, so the edge is visible from both ends rather than only to whoever already
knows about it. See [`dependencies.yaml`](dependencies.yaml).

### `risk/` — what breaks if this goes wrong

`risk/truth-path` · `risk/supply-chain` · `risk/data-loss`

**`risk/truth-path` requires a human decision and is never self-approved.** It
means the change touches the ledger or the canonical model, where a mistake is
appended rather than overwritten.

## Repository-specific

### `area/` — which concern

Tier 1: the *namespace* is shared, its values are not. Each repository names its
own concerns, because a taxonomy imported from elsewhere describes a structure
that does not exist here — the "playbook with no subject" problem.

In `fdos`: `kernel` · `ledger` · `contracts` · `risk` · `corp-actions` ·
`graph` · `mcp` · `platform` · `context`.

`fdos-connectors` defines its own and this file does not presume them.

## Enforcement

**Rung 6 throughout.** Nothing checks that a label exists, that exactly one per
namespace is applied, that `contract/breaking` was reviewed by a human, or that
this file still matches the repository's actual labels.

The labels were created from a script rather than by hand, so they are at least
reproducible; that script is not committed, because a one-time provisioning
script kept as though it were a generator is the failure I6 names. If label
drift becomes a real problem, the answer is a check that compares GitHub's
labels against this file — with a negative test — not a script nobody runs.
