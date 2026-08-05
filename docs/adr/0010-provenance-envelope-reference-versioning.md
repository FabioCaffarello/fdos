---
id: ADR-0010
title: Every fact carries a provenance envelope and pins its reference-data versions
status: Accepted
date: 2026-08-05
deciders:
  - "@FabioCaffarello"
supersedes: []
superseded_by: []
---

# ADR-0010 — Every fact carries a provenance envelope and pins its reference-data versions

Records the acceptance of [RFC-0004](../rfc/0004-provenance-and-reference-data.md).

## Context

Optional provenance becomes absent provenance: the first ingestion path under
time pressure omits it, nothing fails, and a year later a material fraction of
the ledger cannot say where it came from. Separately, Constitution §9 requires
reproducing a report from "the same ledger and reference datasets" — but
nothing captured *which* reference data a calculation used. Regenerating a 2026
report in 2031 with 2031 FX rates silently produces a different number.

Both are permanently unrecoverable if deferred: the information exists only at
capture time. This was the gap neither the original roadmap nor its first
review caught.

Serves Constitution §6 (provenance) and §9 (reproducibility).

## Decision

Every canonical fact is an envelope
`Fact[T] = {payload, effective, knowledge, provenance, references}`, and no
constructor produces a fact without one — a fact with no provenance is
unrepresentable, not discouraged.

Provenance records source, collection time, the source's claimed publication
time (distinct from knowledge time — a source's clock must not contaminate
FDOS's ordering), the versioned interpreter, an optional derivation reference,
and confidence.

**Confidence is ordinal, not numeric:**
`Asserted | Derived | Estimated | Inferred | Disputed`. A numeric confidence
implies a calibration FDOS does not have and invites arithmetic with no
defensible semantics.

**Derivations are referenced, not inlined.** A `DerivationRecord` — method,
inputs, parameters (including every `RoundingContext`, ADR-0008), reference
bindings — is content-addressed and stored once; the envelope carries the hash.
Lineage is a graph, always traversable to primary sources; a calculation that
cannot name its inputs cannot produce a fact.

**Reference data is versioned, immutable and itself bitemporal.** A calculation
records every dataset version it consumed via
`ReferenceBinding = {dataset, version}`; a revised rate is a new version, never
an edit. Reproducing a report pins all three legs: the ledger as of its
temporal coordinates, the reference dataset versions, and the interpreters and
methods — the code being the leg most systems forget.

## Consequences

### Positive

- Any number is auditable back to primary sources; everything derived from a
  source found wrong is mechanically invalidatable.
- Historical reports reproduce exactly, years later.
- The derivation record is already most of a computation trace — ADR-0012
  builds explainability on it rather than duplicating it.

### Negative

- Every write carries an envelope and every calculation signature grows
  reference bindings; storage grows meaningfully. For high-frequency facts the
  envelope may exceed the payload — the batch-provenance question is open and
  risks reintroducing optionality by the back door.
- Content addressing makes dataset immutability checkable but pushes trust
  onto whoever publishes a dataset version FDOS did not produce; that trust
  model is unresolved.

### Enforcement

Today: rung 5 — there is no code. From M2: rung 1 (no constructor omits the
envelope; confidence has no numeric operations; calculation signatures require
reference bindings) and rung 2 (content addressing — a changed dataset has a
different address). From M6: rung 3 — property test that every derived fact
reaches a primary source, and the golden-file reproduction gate: pinned ledger
+ references + methods → byte-identical output.

## Alternatives considered

- **Provenance as an optional side table** — optional becomes absent; joining
  later cannot recover what was never written.
- **Inline transformation history** — quadratic growth in folds.
- **Numeric confidence in `[0,1]`** — arithmetic without semantics; a genuinely
  calibrated probability would be a distinct, explicitly calibrated field.
- **Capturing reference values instead of versions** — duplicates datasets per
  calculation and still cannot answer "which version".
- **Deferring reference data until a calculation needs it** — the retrofit
  that cannot be done.

Full exploration in RFC-0004.

## Notes

Open, deliberately: whether `SourceRef` is an opaque reference resolved
privately (Open Core, Constitution §13); per-batch provenance with per-fact
overrides; the trust model for externally produced dataset versions; whether
`Disputed` is a confidence level or a separate assertion about a fact.
