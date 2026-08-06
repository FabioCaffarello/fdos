// Package provenance carries the origin and derivation of every FDOS fact
// (RFC-0004, ADR-0010).
//
// The guarantee this package exists to provide is structural: **there is no way
// to build a Provenance without a source, a collection time, an interpreter and
// a confidence.** The fields are unexported, there is no literal construction,
// and the only constructor requires all four.
//
// That is the rung the wire form could not reach. proto3 has no `required`, so
// the schema can only be checked (ADR-0018); here, omitting provenance is
// unrepresentable.
package provenance

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// Errors returned by this package.
var (
	// ErrIncomplete is returned when a required part of the envelope is absent.
	ErrIncomplete = errors.New("provenance: incomplete")

	// ErrUnknownConfidence is returned when confidence is not one of the
	// defined levels.
	ErrUnknownConfidence = errors.New("provenance: unknown confidence level")
)

// Confidence is ordinal, never numeric.
//
// A numeric confidence implies a calibration FDOS does not have. Nobody can
// defend the difference between 0.87 and 0.88, but everyone would compare them,
// multiply them and threshold on them. This type supports the operation
// actually needed — is this more trustworthy than that — and refuses the
// arithmetic that would be meaningless.
//
// There is deliberately no Confidence arithmetic anywhere in FDOS.
type Confidence uint8

// Confidence levels, ordered from most to least trustworthy. The zero value is
// invalid so an unset field cannot pass for an assertion.
const (
	ConfidenceUnspecified Confidence = iota
	ConfidenceAsserted               // stated directly by the source
	ConfidenceDerived                // computed from asserted facts
	ConfidenceEstimated              // produced by a model with known inputs
	ConfidenceInferred               // concluded indirectly
	ConfidenceDisputed               // sources disagree
)

// String returns the level name, matching fdos.kernel.v1.Confidence.
func (c Confidence) String() string {
	switch c {
	case ConfidenceAsserted:
		return "asserted"
	case ConfidenceDerived:
		return "derived"
	case ConfidenceEstimated:
		return "estimated"
	case ConfidenceInferred:
		return "inferred"
	case ConfidenceDisputed:
		return "disputed"
	default:
		return "unspecified"
	}
}

// Valid reports whether the level is one of the defined ones.
func (c Confidence) Valid() bool {
	return c >= ConfidenceAsserted && c <= ConfidenceDisputed
}

// AtLeastAsTrustworthyAs reports whether c is at least as trustworthy as other.
//
// The only ordering operation defined. Deliberately not `Less`, so that nobody
// is tempted to sort by it and read the result as a score.
func (c Confidence) AtLeastAsTrustworthyAs(other Confidence) bool { return c <= other }

// Weakest returns the least trustworthy of the given levels.
//
// This is how confidence propagates through a derivation: a computed value is
// no more trustworthy than its weakest input. Not multiplication — that would
// be the arithmetic this type exists to refuse.
func Weakest(levels ...Confidence) Confidence {
	worst := ConfidenceAsserted
	for _, l := range levels {
		if l > worst {
			worst = l
		}
	}
	return worst
}

// Source identifies the acquisition a datum came from, by content address
// (ADR-0028).
//
// Opaque and unconstrained are different properties. FDOS never dereferences a
// Source, never learns what it addresses, and gains no dependency on any store
// — resolving one stays a private concern (Constitution §13). That is opacity.
// The form is specified; opacity never required it not to be.
type Source struct{ value string }

// NewSource builds a source reference from an algorithm-prefixed content hash.
//
// The grammar is documented on fdos.kernel.v1.SourceRef and is NOT enforced
// here. ADR-0028 places admissibility at the admission path, which does not
// exist yet — validating in this constructor would move the rule to rung 1 for
// every value crossing the wire codec, which is a wider change than the ADR
// decided and is recorded as a finding rather than taken silently.
func NewSource(value string) (Source, error) {
	if strings.TrimSpace(value) == "" {
		return Source{}, fmt.Errorf("%w: source is empty", ErrIncomplete)
	}
	return Source{value: value}, nil
}

// String returns the content address, unresolved.
func (s Source) String() string { return s.value }

// ErrMalformedSource is returned when a source is not a well-formed
// algorithm-prefixed content address.
var ErrMalformedSource = fmt.Errorf("provenance: malformed source")

// sourceAlgorithm is the only digest algorithm the grammar admits today. It is
// a prefix rather than an assumption so that a second algorithm is a new
// prefix, never a silent reinterpretation of the same 64 characters.
const sourceAlgorithm = "sha256:"

// CheckContentAddress reports whether the source matches the grammar ADR-0028
// specified: `sha256:` followed by 64 lowercase hexadecimal characters.
//
// **Not called by NewSource, deliberately.** The wire codec constructs sources
// on decode, so enforcing there would be enforcement at *decode* rather than at
// admission — and those differ in what they destroy. Admission rejects bad new
// data; decode makes existing data unreadable. A producer whose stored
// references predate this grammar would find facts it already holds
// unrecoverable through the normal path.
//
// So the rule lives here and is called at admission, where the cost of being
// wrong is a rejected submission.
//
// It catches accident, not intent. A well-formed digest of nothing passes.
func (s Source) CheckContentAddress() error {
	if !strings.HasPrefix(s.value, sourceAlgorithm) {
		return fmt.Errorf("%w: %q has no recognised algorithm prefix, expected %q",
			ErrMalformedSource, s.value, sourceAlgorithm)
	}
	digest := strings.TrimPrefix(s.value, sourceAlgorithm)
	if len(digest) != 64 {
		return fmt.Errorf("%w: digest is %d characters, sha256 is 64",
			ErrMalformedSource, len(digest))
	}
	for _, r := range digest {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return fmt.Errorf("%w: %q is not lowercase hexadecimal", ErrMalformedSource, digest)
		}
	}
	return nil
}

// Interpreter names the versioned code that read or computed a value — a
// parser or a calculation method. Pinned in a report so that regenerating it
// uses the interpreter of the time, not of today (ADR-0010).
type Interpreter struct {
	name    string
	version string
}

// NewInterpreter builds an interpreter reference. Both parts are required: a
// name without a version cannot be pinned, which defeats the purpose.
func NewInterpreter(name, version string) (Interpreter, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
		return Interpreter{}, fmt.Errorf("%w: interpreter needs a name and a version", ErrIncomplete)
	}
	return Interpreter{name: name, version: version}, nil
}

// UnmediatedName is the reserved interpreter name asserting that no code read a
// value — it was transcribed as the source stated it (ADR-0028).
const UnmediatedName = "unmediated"

// UnmediatedVersion versions the convention, not code that does not exist. A
// later change to what "unmediated" means stays distinguishable on replay.
const UnmediatedVersion = "1"

// Unmediated is the interpreter of a producer that interpreted nothing
// programmatically — a person submitting an exported statement.
//
// It asserts a positive fact. There is deliberately no value meaning "I do not
// know": a producer that cannot make this assertion names its pipeline or
// cannot submit. A sentinel absorbing both "no code read this" and "nobody
// filled this in" would make the field optional in practice while claiming to
// be required.
//
// Nothing detects a producer using it while having interpreted
// programmatically. It is an affordance for honesty, not a control — rung 6,
// and ADR-0028 says so.
func Unmediated() Interpreter {
	return Interpreter{name: UnmediatedName, version: UnmediatedVersion}
}

// IsUnmediated reports whether this interpreter asserts that no code read the
// value.
func (i Interpreter) IsUnmediated() bool { return i.name == UnmediatedName }

// Name returns the interpreter name.
func (i Interpreter) Name() string { return i.name }

// Version returns the interpreter version.
func (i Interpreter) Version() string { return i.version }

// String renders "name@version".
func (i Interpreter) String() string { return i.name + "@" + i.version }

// ReferenceBinding pins a versioned reference dataset a calculation consumed:
// FX rates, holiday calendars, day-count conventions.
//
// The version, not the values. Values are recoverable from a version; the
// reverse is not true. This is the part that cannot be retrofitted —
// regenerating a 2026 report in 2031 against 2031 rates produces a different
// number, and nothing would indicate that it changed.
type ReferenceBinding struct {
	dataset string
	version string
}

// NewReferenceBinding pins a dataset version.
func NewReferenceBinding(dataset, version string) (ReferenceBinding, error) {
	if strings.TrimSpace(dataset) == "" || strings.TrimSpace(version) == "" {
		return ReferenceBinding{}, fmt.Errorf("%w: reference binding needs a dataset and a version", ErrIncomplete)
	}
	return ReferenceBinding{dataset: dataset, version: version}, nil
}

// Dataset returns the dataset identifier.
func (r ReferenceBinding) Dataset() string { return r.dataset }

// Version returns the pinned, immutable dataset version.
func (r ReferenceBinding) Version() string { return r.version }

// String renders "dataset@version".
func (r ReferenceBinding) String() string { return r.dataset + "@" + r.version }

// Provenance travels with every fact.
//
// There is no exported constructor that omits any part of it. A fact without
// provenance is unrepresentable, not merely discouraged — which is the whole
// distinction between rung 1 and rung 3.
type Provenance struct {
	source      Source
	collectedAt temporal.Instant
	publishedAt temporal.Instant // optional: a claim by the source
	interpreter Interpreter
	derivation  *Ref // set when the fact was computed rather than observed
	confidence  Confidence
}

// Observed builds provenance for a fact FDOS was told.
//
// Every argument is required. `publishedAt` is the one optional part and is set
// separately, because it is a claim by the source that may be absent or wrong —
// conflating it with knowledge time would let a source's clock contaminate
// FDOS's ordering.
func Observed(
	source Source,
	collectedAt temporal.Instant,
	interpreter Interpreter,
	confidence Confidence,
) (Provenance, error) {
	if source.value == "" {
		return Provenance{}, fmt.Errorf("%w: source", ErrIncomplete)
	}
	if collectedAt.IsZero() {
		return Provenance{}, fmt.Errorf("%w: collected_at", ErrIncomplete)
	}
	if interpreter.name == "" {
		return Provenance{}, fmt.Errorf("%w: interpreter", ErrIncomplete)
	}
	if !confidence.Valid() {
		return Provenance{}, fmt.Errorf("%w: %d", ErrUnknownConfidence, confidence)
	}
	return Provenance{
		source:      source,
		collectedAt: collectedAt,
		interpreter: interpreter,
		confidence:  confidence,
	}, nil
}

// Derived builds provenance for a computed fact, referencing the derivation
// that produced it.
//
// A calculation that cannot name its inputs cannot produce a fact: the Ref is
// required, and Refs are only obtainable from a completed DerivationRecord.
func Derived(
	source Source,
	collectedAt temporal.Instant,
	interpreter Interpreter,
	derivation Ref,
	confidence Confidence,
) (Provenance, error) {
	p, err := Observed(source, collectedAt, interpreter, confidence)
	if err != nil {
		return Provenance{}, err
	}
	if derivation.hash == "" {
		return Provenance{}, fmt.Errorf("%w: derivation reference", ErrIncomplete)
	}
	p.derivation = &derivation
	return p, nil
}

// WithPublishedAt records the source's own publication claim.
func (p Provenance) WithPublishedAt(at temporal.Instant) Provenance {
	p.publishedAt = at
	return p
}

// Source returns who asserted the fact.
func (p Provenance) Source() Source { return p.source }

// CollectedAt returns when FDOS fetched it.
func (p Provenance) CollectedAt() temporal.Instant { return p.collectedAt }

// PublishedAt returns the source's publication claim, which may be unset.
func (p Provenance) PublishedAt() temporal.Instant { return p.publishedAt }

// Interpreter returns the versioned code that produced the value.
func (p Provenance) Interpreter() Interpreter { return p.interpreter }

// Derivation returns the derivation reference and whether the fact was
// computed rather than observed.
func (p Provenance) Derivation() (Ref, bool) {
	if p.derivation == nil {
		return Ref{}, false
	}
	return *p.derivation, true
}

// Confidence returns how much to trust the fact.
func (p Provenance) Confidence() Confidence { return p.confidence }

// IsZero reports whether this is the unusable zero value. Useful for asserting
// that an envelope was actually populated.
func (p Provenance) IsZero() bool { return p.source.value == "" }

// Ref is a content address of a DerivationRecord.
//
// Referenced rather than inlined: a value derived from a thousand facts would
// otherwise carry a thousand entries, making a fold quadratic. Content
// addressing keeps lineage a DAG and deduplicates identical derivations.
type Ref struct{ hash string }

// NewRef rebuilds a reference from a content address.
//
// Exists for one reason: a codec decoding a fact from the wire must reconstruct
// the derivation reference it carries, and every other route to a Ref goes
// through building the DerivationRecord itself — which a decoder cannot do,
// because the record lives elsewhere.
//
// Deliberately minimal validation. This asserts the address is well-formed, not
// that the record exists: resolving it is a store's job, and a kernel that
// tried would need I/O it must not have.
func NewRef(hash string) (Ref, error) {
	if len(hash) != sha256HexLen {
		return Ref{}, fmt.Errorf("%w: content address is not a %d-character hex digest", ErrIncomplete, sha256HexLen)
	}
	for _, r := range hash {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return Ref{}, fmt.Errorf("%w: content address %q is not lowercase hex", ErrIncomplete, hash)
		}
	}
	return Ref{hash: hash}, nil
}

// sha256HexLen is the length of a hex-encoded SHA-256 digest, which is what
// contentAddress produces.
const sha256HexLen = 64

// Hash returns the content address.
func (r Ref) Hash() string { return r.hash }

// String returns the content address.
func (r Ref) String() string { return r.hash }

// IsZero reports whether the reference is unset.
func (r Ref) IsZero() bool { return r.hash == "" }

// Method names a versioned calculation.
type Method struct {
	name    string
	version string
}

// NewMethod builds a method reference.
func NewMethod(name, version string) (Method, error) {
	if strings.TrimSpace(name) == "" || strings.TrimSpace(version) == "" {
		return Method{}, fmt.Errorf("%w: method needs a name and a version", ErrIncomplete)
	}
	return Method{name: name, version: version}, nil
}

// String renders "name@version".
func (m Method) String() string { return m.name + "@" + m.version }

// Name returns the method name.
func (m Method) Name() string { return m.name }

// Version returns the method version.
func (m Method) Version() string { return m.version }

// Parameter is a named input to a calculation that is not itself a fact —
// notably the RoundingContext of every division (ADR-0008).
type Parameter struct {
	Name  string
	Value string
}

// DerivationRecord is the content-addressed record of one computation.
//
// It answers "how was this number produced" and is what makes it reproducible:
// inputs, method, parameters, and the reference dataset versions consumed.
//
// It is also most of a computation trace already, which is why making
// calculations explain themselves costs almost nothing structurally (ADR-0012).
type DerivationRecord struct {
	method     Method
	inputs     []string // opaque fact references, ordered
	parameters []Parameter
	references []ReferenceBinding
	confidence Confidence
	hash       string
}

// NewDerivation builds a derivation record and computes its content address.
//
// The hash is computed over a canonicalised form with sorted parameters and
// references, so two identical computations produce the same address regardless
// of the order the caller happened to supply them — which is what makes
// deduplication meaningful.
func NewDerivation(
	method Method,
	inputs []string,
	parameters []Parameter,
	references []ReferenceBinding,
	confidence Confidence,
) (DerivationRecord, error) {
	if method.name == "" {
		return DerivationRecord{}, fmt.Errorf("%w: method", ErrIncomplete)
	}
	if !confidence.Valid() {
		return DerivationRecord{}, fmt.Errorf("%w: %d", ErrUnknownConfidence, confidence)
	}

	d := DerivationRecord{
		method:     method,
		inputs:     append([]string(nil), inputs...),
		parameters: append([]Parameter(nil), parameters...),
		references: append([]ReferenceBinding(nil), references...),
		confidence: confidence,
	}
	sort.Slice(d.parameters, func(i, j int) bool { return d.parameters[i].Name < d.parameters[j].Name })
	sort.Slice(d.references, func(i, j int) bool { return d.references[i].dataset < d.references[j].dataset })
	d.hash = d.contentAddress()
	return d, nil
}

// Ref returns the content address of this record.
func (d DerivationRecord) Ref() Ref { return Ref{hash: d.hash} }

// Method returns the calculation that produced the value.
func (d DerivationRecord) Method() Method { return d.method }

// Inputs returns the fact references consumed.
func (d DerivationRecord) Inputs() []string { return append([]string(nil), d.inputs...) }

// Parameters returns the non-fact inputs, sorted by name.
func (d DerivationRecord) Parameters() []Parameter {
	return append([]Parameter(nil), d.parameters...)
}

// References returns the pinned reference dataset versions, sorted by dataset.
func (d DerivationRecord) References() []ReferenceBinding {
	return append([]ReferenceBinding(nil), d.references...)
}

// Confidence returns the trustworthiness of the derived value.
func (d DerivationRecord) Confidence() Confidence { return d.confidence }

// contentAddress hashes a canonical rendering. Deterministic by construction:
// no map iteration, no clock, no randomness — the `nondet` analyser would
// reject any of them here.
func (d DerivationRecord) contentAddress() string {
	var b strings.Builder
	b.WriteString("method=")
	b.WriteString(d.method.String())
	for _, in := range d.inputs {
		b.WriteString("\ninput=")
		b.WriteString(in)
	}
	for _, p := range d.parameters {
		b.WriteString("\nparam=")
		b.WriteString(p.Name)
		b.WriteString("=")
		b.WriteString(p.Value)
	}
	for _, r := range d.references {
		b.WriteString("\nref=")
		b.WriteString(r.String())
	}
	b.WriteString("\nconfidence=")
	b.WriteString(d.confidence.String())

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}
