---
directory: analysis
purpose: Static analysers that turn FDOS architectural principles into build errors.
owner: "@FabioCaffarello"
allowed:
  - go/analysis passes enforcing Constitution principles
  - Test fixtures under testdata/, including deliberately non-compliant Go
  - cmd/fdoslint, the composition root that wires the passes together
forbidden:
  - Financial or domain logic of any kind
  - Rules that fire outside the layer they govern
  - Rules without fixtures for both violating and compliant code
  - Suppression mechanisms beyond the standard //lint:ignore vocabulary
---

# libs/analysis

The mechanisms that move FDOS principles from documentation (rung 5) to static
analysis (rung 2) on the enforcement ladder — see `docs/constitution.md` §15 and
ADR-0005.

This is a **tooling module**, not a bounded context. Per ADR-0013 it carries its
own `cmd/`, because splitting a linter across two modules would require a
local-path `replace` directive — reintroducing exactly the by-path coupling
ADR-0004 exists to prevent.

## The analysers

| Analyser | Enforces | Reports |
|----------|----------|---------|
| `nofloat` | §2, §9 | `float32`/`float64` types and literals in domain packages |
| `nondet` | §2, §9 | clock reads, randomness, environment access, map-iteration order |
| `impurity` | §3, §10 | goroutines, channels, select, `sync`, `context.Context`, serialisation |
| `layering` | §3, §11 | layer inversion and cross-bounded-context imports (ADR-0013) |

```sh
make analyze                       # run them over every module
go run ./cmd/fdoslint ./...        # run them here
```

## Why `nofloat` is the highest-leverage rule

The usual objection to floating point in financial code is representation error
— `0.1 + 0.2 != 0.3`. The decisive problem in FDOS is different and worse:
**floating-point addition is not associative.**

A projection that folds the same events in a different but equally valid order
produces a different total. That breaks Constitution §9 on its own, independently
of precision, because the same ledger must produce a byte-identical report.

## Scope: the rules must not fire everywhere

Every analyser applies only to the layer it governs. An adapter is *supposed* to
read the clock, consult the environment and start goroutines.

This is not politeness. An analyser that fires on legitimate code gets disabled,
and a disabled rule enforces nothing. Every analyser therefore has fixtures for
both cases, and the test asserting the rule stays **silent** outside its layer is
as important as the one asserting it fires inside.

The domain layer is identified by package path (`-domain`, default
`(^|/)domain(/|$)`), matching the layout fixed by ADR-0013.

## The false positive that shaped `nondet`

The remedy for "do not range over a map" is *itself* a map range:

```go
for k := range m {
    keys = append(keys, k)
}
sort.Strings(keys)
```

The first version of the rule reported that loop — making it impossible to
satisfy without a suppression comment, and a rule requiring routine suppression
is a rule that will be switched off. `nondet` now recognises exactly this shape
and stays quiet. Anything wider is still reported.

The false positive was found by writing a *compliant* fixture, not a violating
one. That is why both exist.

## Adding a rule

1. New package with the analyser, named after what it forbids.
2. Fixtures under `testdata/src/ctx/domain/` (must fire) **and**
   `testdata/src/ctx/adapters/` (must stay silent).
3. Register it in `cmd/fdoslint`.
4. Update `docs/constitution.md` §15 — a principle that has climbed a rung must
   be recorded as having climbed.

A rule with no compliant fixture is not finished. It has been tested for
sensitivity and not for specificity, and specificity is what determines whether
anyone leaves it switched on.
