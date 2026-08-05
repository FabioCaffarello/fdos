// Package domain is the pure core of the FDOS ledger.
//
// Everything here is a total function of its inputs. No clock, no randomness,
// no I/O, no concurrency, no serialisation, no binary floating point — the
// `nofloat`, `nondet` and `impurity` analysers enforce every one of those at
// build time (ADR-0013).
//
// Three Constitution principles become structural in this package:
//
//   - **§1 Financial Truth.** There is no position type to store. A position is
//     the return value of a projection, computed from facts. Nothing here can
//     persist derived state because nothing here can persist anything.
//   - **§4 Immutable Ledger.** A Stream has Append and no Update or Delete. A
//     correction is a new fact referencing what it corrects.
//   - **§6/§7 Provenance and Bitemporality.** NewFact takes an Envelope, and an
//     Envelope cannot be built without both temporal coordinates and
//     provenance. A fact missing either is unrepresentable — the rung proto3
//     could not reach (ADR-0018).
package domain

import (
	"errors"
	"fmt"

	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// Errors returned by this package.
var (
	// ErrIncompleteEnvelope is returned when a fact is built without full
	// temporal coordinates or provenance.
	ErrIncompleteEnvelope = errors.New("ledger: incomplete envelope")

	// ErrUnknownKind is returned for a fact kind outside the closed set.
	ErrUnknownKind = errors.New("ledger: unknown fact kind")

	// ErrEmptyType is returned when a fact carries no type name.
	ErrEmptyType = errors.New("ledger: fact type is empty")

	// ErrNotFound is returned when a referenced fact is not in the stream.
	ErrNotFound = errors.New("ledger: fact not found")
)

// Kind separates what happened from what FDOS was told (ADR-0011).
//
// A trade settling is an Occurrence. A broker statement saying the position is
// 100 shares is an Observation — the position is not an event, and recording a
// statement line as though it were a transaction fabricates history that never
// took place.
type Kind uint8

// Fact kinds. The zero value is invalid.
const (
	KindUnspecified Kind = iota
	KindOccurrence
	KindObservation
)

// String returns the kind name, matching fdos.ledger.v1.FactKind.
func (k Kind) String() string {
	switch k {
	case KindOccurrence:
		return "occurrence"
	case KindObservation:
		return "observation"
	default:
		return "unspecified"
	}
}

// Valid reports whether the kind is one of the defined ones.
func (k Kind) Valid() bool { return k == KindOccurrence || k == KindObservation }

// Envelope is what makes something a fact.
//
// The fields are unexported and the only constructor requires both. This is the
// guarantee ADR-0018 could not deliver on the wire: proto3 has no `required`,
// so a schema can only be *checked*. Here, an envelope-less fact cannot be
// written.
type Envelope struct {
	coordinates temporal.Coordinates
	provenance  provenance.Provenance
	references  []provenance.ReferenceBinding
}

// NewEnvelope builds an envelope. Both arguments are required.
//
// `references` pins the reference dataset versions the fact depended on. It may
// be empty for a directly observed fact, which depended on none — but it is a
// parameter rather than an afterthought, so a calculation cannot forget it.
func NewEnvelope(
	coordinates temporal.Coordinates,
	prov provenance.Provenance,
	references []provenance.ReferenceBinding,
) (Envelope, error) {
	if coordinates.Knowledge().IsZero() {
		return Envelope{}, fmt.Errorf("%w: temporal coordinates", ErrIncompleteEnvelope)
	}
	if prov.IsZero() {
		return Envelope{}, fmt.Errorf("%w: provenance", ErrIncompleteEnvelope)
	}
	return Envelope{
		coordinates: coordinates,
		provenance:  prov,
		references:  append([]provenance.ReferenceBinding(nil), references...),
	}, nil
}

// Coordinates returns where the fact sits on both temporal axes.
func (e Envelope) Coordinates() temporal.Coordinates { return e.coordinates }

// Provenance returns where the fact came from.
func (e Envelope) Provenance() provenance.Provenance { return e.provenance }

// References returns the pinned reference dataset versions.
func (e Envelope) References() []provenance.ReferenceBinding {
	return append([]provenance.ReferenceBinding(nil), e.references...)
}

// Ref addresses a fact within a stream.
//
// `Sequence` is the deterministic tiebreaker for facts sharing both temporal
// coordinates. It is ordering machinery, never semantics: no business rule may
// read it, and none in this package does (ADR-0009).
type Ref struct {
	Stream   string
	Sequence uint64
}

// String renders the reference.
func (r Ref) String() string { return fmt.Sprintf("%s#%d", r.Stream, r.Sequence) }

// Payload is what a fact asserts. Implementations are defined in this package
// so that the closed set of fact types is visible in one place.
type Payload interface {
	// FactType returns the domain-qualified type name, past tense for
	// occurrences: "ledger.HoldingObserved". Never "Created" or "Updated" —
	// those describe what a database did, not what is true.
	FactType() string

	// TypeVersion is the per-type major version. There is deliberately no
	// global schema version: a global one couples unrelated types and, in
	// practice, freezes them (ADR-0011).
	TypeVersion() uint32
}

// Fact is the unit the ledger appends.
//
// Immutable: every field is unexported with a read-only accessor, and there is
// no method that returns a mutable view. A correction is a new Fact.
type Fact struct {
	ref      Ref
	envelope Envelope
	kind     Kind
	payload  Payload
}

// NewFact builds a fact. The envelope is required, which is the point.
//
// `ref` is assigned by the stream at append, not by the caller: a caller
// choosing its own sequence could reorder history.
func NewFact(ref Ref, envelope Envelope, kind Kind, payload Payload) (Fact, error) {
	if envelope.provenance.IsZero() {
		return Fact{}, fmt.Errorf("%w: provenance", ErrIncompleteEnvelope)
	}
	if !kind.Valid() {
		return Fact{}, fmt.Errorf("%w: %d", ErrUnknownKind, kind)
	}
	if payload == nil || payload.FactType() == "" {
		return Fact{}, ErrEmptyType
	}
	return Fact{ref: ref, envelope: envelope, kind: kind, payload: payload}, nil
}

// Ref returns the address of this fact.
func (f Fact) Ref() Ref { return f.ref }

// Envelope returns the temporal coordinates, provenance and reference bindings.
func (f Fact) Envelope() Envelope { return f.envelope }

// Kind returns whether the fact is an occurrence or an observation.
func (f Fact) Kind() Kind { return f.kind }

// Type returns the domain-qualified fact type.
func (f Fact) Type() string { return f.payload.FactType() }

// TypeVersion returns the per-type major version.
func (f Fact) TypeVersion() uint32 { return f.payload.TypeVersion() }

// Payload returns what the fact asserts.
func (f Fact) Payload() Payload { return f.payload }

// Visible reports whether this fact is visible to a query at the given
// coordinates. Both conditions must hold: it was already known, and it was in
// force.
func (f Fact) Visible(asOf temporal.AsOf) bool {
	return f.envelope.coordinates.Visible(asOf)
}
