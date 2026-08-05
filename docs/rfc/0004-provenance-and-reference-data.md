---
id: RFC-0004
title: Provenance envelope and reference data versioning
status: Proposed
date: 2026-08-05
authors:
  - "@FabioCaffarello"
---

# RFC-0004 — Provenance envelope and reference data versioning

## Summary

Proposes that provenance is **universal and structural** rather than optional,
defines the envelope, and specifies how versioned reference data is captured so
that a report remains reproducible after the reference data itself has changed.

## Motivation

Two problems, one envelope.

**Provenance.** Constitution §6 says provenance is preserved "whenever
applicable". Optional provenance becomes absent provenance: the first ingestion
path under time pressure omits it, nothing fails, and a year later a material
fraction of the ledger cannot say where it came from. A datum with no origin
cannot be audited, re-verified, or invalidated when its source is found to be
wrong.

**Reference data.** This is the gap that neither the original roadmap nor its
first review caught. Constitution §9 requires a report to be reproducible "using
the same ledger and reference datasets" — but nothing captured *which* reference
data was used. Regenerating a 2026 report in 2031 with 2031 FX rates produces a
different number, and nothing indicates that the number changed.

Both are **permanently unrecoverable** if deferred. The information exists only
at capture time.

## Design

### The envelope is not optional

Every canonical fact is an envelope. There is no constructor that produces a
fact without one, so a fact with no provenance is unrepresentable rather than
merely discouraged.

```
Fact[T] := {
    payload:     T
    effective:   EffectiveInterval     # RFC-0003
    knowledge:   Instant               # RFC-0003
    provenance:  Provenance
    references:  []ReferenceBinding
}
```

### Provenance

```
Provenance := {
    source:            SourceRef        # who asserted it
    collected_at:      Instant          # when FDOS fetched it
    published_at:      Instant | None   # when the source published it
    interpreter:       InterpreterRef   # parser or calculator, versioned
    derivation:        DerivationRef | None
    confidence:        Confidence
}
```

Note `published_at` is distinct from knowledge time. Knowledge time is when FDOS
could act on it; `published_at` is a claim by the source and may be wrong or
absent. Conflating them lets a source's clock contaminate FDOS's ordering.

### Confidence is ordinal, not numeric

```
Confidence := Asserted | Derived | Estimated | Inferred | Disputed
```

A numeric confidence (`0.87`) implies a calibration FDOS does not have. Nobody
can defend the difference between 0.87 and 0.88, but everyone will compare them,
multiply them, and threshold on them. An ordinal scale supports the operation
that is actually needed — *is this more trustworthy than that* — and refuses the
arithmetic that would be meaningless.

It is also decimal-free, avoiding the numeric questions of RFC-0002 entirely.

### Derivation is referenced, not inlined

Naively, transformation history is a list carried in the envelope. That list
grows with every derivation step, and a value derived from a thousand facts
carries a thousand entries — quadratic in a fold.

Instead, a derivation is a content-addressed record stored once:

```
DerivationRecord := {
    id:        Hash            # content address of this record
    method:    MethodRef       # calculation, versioned
    inputs:    []FactRef
    parameters: []Parameter    # incl. RoundingContext (RFC-0002)
    references: []ReferenceBinding
}
```

The envelope carries only the hash. Identical derivations deduplicate
automatically, and the full lineage remains traversable.

This is also the substrate RFC-0006 builds explainability on: a derivation
record is already most of a computation trace.

### Reference data

A reference dataset is versioned, immutable data a calculation depends on but
which is not itself a ledger fact: FX rates, holiday calendars, day-count
conventions, issuer classifications, corporate action schedules.

```
ReferenceBinding := {
    dataset:  DatasetRef       # e.g. "ecb-fx-daily"
    version:  DatasetVersion   # immutable, content-addressed
}
```

Three rules:

1. **A calculation records every dataset version it consumed.** Not the values —
   the version. Values are recoverable from the version; the reverse is not true.
2. **Dataset versions are immutable.** A revised FX rate is a new version, never
   an edit. Reference data obeys the same law as the ledger.
3. **Reference data is itself bitemporal.** "The EUR/USD rate for 2026-03-01, as
   published on 2026-03-01" differs from the same rate as revised on 2026-03-05.
   Both must be addressable.

### Reproduction

To regenerate a report:

```
Reproduce(report) =
    ledger as-of (report.effective, report.knowledge)
  + reference datasets pinned to report.references
  + interpreters and methods pinned to report.provenance
```

All three are required. Pinning the ledger alone is the mistake this RFC exists
to prevent — and pinning the *code* is the third leg that is equally easy to
forget.

### Provenance survives derivation

The rule Constitution §6 states as "no computation may lose provenance" becomes
concrete: a derived fact's provenance references the derivation record, which
references its inputs' facts. Lineage is a graph, always traversable to primary
sources.

A calculation that cannot name its inputs cannot produce a fact.

## Enforcement

| Rule | Rung | Mechanism |
|------|------|-----------|
| No fact without provenance | 1 | No constructor omits the envelope |
| Confidence is not arithmetic | 1 | Ordinal type; no numeric operations defined |
| No calculation without recorded references | 1 | Calculation signature requires `[]ReferenceBinding` |
| Dataset versions immutable | 2 | Content-addressed; a changed dataset has a different address |
| Lineage always traversable | 3 | Property test: every derived fact reaches a primary source |
| Reproduction is exact | 3 | Golden-file test: pinned ledger + references + methods → byte-identical output |

## Alternatives

**Provenance as an optional side table.** Cheaper writes, smaller events.
Rejected: optional provenance becomes absent provenance, and joining after the
fact cannot recover what was never written.

**Inline transformation history.** Simple and self-contained. Rejected on the
quadratic growth above; content addressing gets the same traversability at
constant envelope size.

**Numeric confidence in `[0,1]`.** Standard, and superficially more expressive.
Rejected: it invites arithmetic that has no defensible semantics. If a calibrated
probability is ever genuinely available, it belongs as a distinct, explicitly
calibrated field — not as a reinterpretation of this one.

**Capture reference *values* instead of versions.** Guarantees reproduction with
no external dependency. Rejected: it duplicates entire datasets per calculation
and still cannot answer "which version was that". Worth revisiting for small,
critical datasets where the duplication is cheap.

**Defer reference data to when a calculation needs it.** Rejected — this is the
retrofit that cannot be done. The binding must exist from the first event.

## Prior art

W3C PROV models provenance as a graph of entities, activities and agents; the
derivation-record design here is a narrowed version of that. Content-addressed
immutable datasets are the same insight as Nix and Git, applied to data rather
than code. The reproducibility failure being prevented is well documented in
quantitative finance, where backtests are routinely irreproducible because
reference data drifted underneath them.

## Open questions

- Does `SourceRef` belong in the public core, given that sources are largely
  private connectors? Probably an opaque reference here, resolved privately.
- Envelope size: for high-frequency facts, provenance may exceed the payload.
  Is per-batch provenance with per-fact overrides acceptable, and does that
  reintroduce optionality by the back door?
- Who publishes reference datasets, and what is the trust model for a dataset
  version FDOS did not produce?
- Should `Disputed` be a confidence level, or a separate assertion about a fact?

## Consequences

**Easier:** auditing any number back to primary sources; invalidating everything
derived from a source found to be wrong; reproducing historical reports.

**Harder:** every write carries an envelope. Every calculation signature grows
reference bindings. Storage grows meaningfully.

**Impossible:** producing a fact that cannot say where it came from; silently
regenerating a report against different reference data.
