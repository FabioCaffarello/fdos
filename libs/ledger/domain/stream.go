package domain

import (
	"fmt"
	"sort"

	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// Stream is an append-only sequence of facts (Constitution §4).
//
// There is Append, and there is no Update and no Delete. That is not a
// convention enforced by review — the slice is unexported, every accessor
// returns a copy, and no method mutates an existing entry. A correction is a
// new fact referencing what it corrects.
//
// Stream is a value: Append returns a new Stream rather than mutating the
// receiver. In a pure core that costs a copy and buys the guarantee that no
// caller holds a reference through which history could change underneath
// another reader.
type Stream struct {
	name  string
	facts []Fact
}

// NewStream creates an empty stream.
func NewStream(name string) (Stream, error) {
	if name == "" {
		return Stream{}, fmt.Errorf("%w: stream name is empty", ErrEmptyType)
	}
	return Stream{name: name}, nil
}

// Name returns the stream identifier.
func (s Stream) Name() string { return s.name }

// Len returns how many facts the stream holds.
func (s Stream) Len() int { return len(s.facts) }

// Append records a fact and returns the extended stream and the assigned
// reference.
//
// The sequence is assigned here, never by the caller: a caller choosing its own
// could reorder history, and the sequence is the deterministic tiebreaker that
// makes projections reproducible when two facts share both temporal
// coordinates (ADR-0009).
//
// Knowledge time is likewise not a parameter of this method — it arrives inside
// the envelope, which the application layer built from an injected clock. A
// domain rule cannot invent one.
func (s Stream) Append(envelope Envelope, kind Kind, payload Payload) (Stream, Ref, error) {
	ref := Ref{Stream: s.name, Sequence: uint64(len(s.facts)) + 1}

	fact, err := NewFact(ref, envelope, kind, payload)
	if err != nil {
		return Stream{}, Ref{}, err
	}

	extended := Stream{
		name:  s.name,
		facts: make([]Fact, len(s.facts), len(s.facts)+1),
	}
	copy(extended.facts, s.facts)
	extended.facts = append(extended.facts, fact)

	return extended, ref, nil
}

// Facts returns every fact in append order.
func (s Stream) Facts() []Fact { return append([]Fact(nil), s.facts...) }

// Get returns the fact at a reference.
func (s Stream) Get(ref Ref) (Fact, error) {
	if ref.Stream != s.name || ref.Sequence == 0 || ref.Sequence > uint64(len(s.facts)) {
		return Fact{}, fmt.Errorf("%w: %s", ErrNotFound, ref)
	}
	return s.facts[ref.Sequence-1], nil
}

// VisibleAt returns the facts visible at a query coordinate, in deterministic
// order.
//
// The ordering is (effective_from, knowledge, sequence), which is a total order
// even when two facts share both temporal coordinates. Without the sequence
// tiebreaker a projection could differ between runs on the same data, which is
// the failure Constitution §9 forbids.
//
// `asOf` is required and has no default. That is the mechanism that makes
// look-ahead bias structural: the common failure is a knowledge time defaulting
// to "now", which is invisible and almost always wrong in an analysis.
func (s Stream) VisibleAt(asOf temporal.AsOf) []Fact {
	visible := make([]Fact, 0, len(s.facts))
	for _, f := range s.facts {
		if f.Visible(asOf) {
			visible = append(visible, f)
		}
	}

	sort.SliceStable(visible, func(i, j int) bool {
		a, b := visible[i].Envelope().Coordinates(), visible[j].Envelope().Coordinates()
		if !a.Effective().From().Equal(b.Effective().From()) {
			return a.Effective().From().Before(b.Effective().From())
		}
		if !a.Knowledge().Equal(b.Knowledge()) {
			return a.Knowledge().Before(b.Knowledge())
		}
		return visible[i].Ref().Sequence < visible[j].Ref().Sequence
	})

	return visible
}

// Corrections returns the corrections visible at a coordinate, indexed by the
// reference they correct.
//
// Returned as a slice rather than a map so that iteration order is the caller's
// to decide. A map here would make a fold over corrections depend on Go's
// randomised iteration order — which `nondet` would catch, and which would be a
// reproducibility bug if it did not.
func (s Stream) Corrections(asOf temporal.AsOf) []FactCorrected {
	out := make([]FactCorrected, 0)
	for _, f := range s.VisibleAt(asOf) {
		if c, ok := f.Payload().(FactCorrected); ok {
			out = append(out, c)
		}
	}
	return out
}
