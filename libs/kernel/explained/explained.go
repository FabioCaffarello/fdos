// Package explained makes a calculation that cannot explain itself impossible
// to write (RFC-0006, ADR-0012).
//
// Constitution §8 was the weakest principle in FDOS: the only one with no
// mechanism above documentation. The remedy is not a reporting feature — it is
// a return type. A domain calculation returns Explained[T], never a bare T, so
// the trace is produced by the computation rather than reconstructed afterwards
// as a plausible story about it.
//
// The combinators are the load-bearing part. Threading traces by hand
// guarantees they get dropped; Map, Combine2 and Fold construct the derivation
// record as a side effect of composing, so losing it requires deliberately
// reaching for Value.
//
// Scope: financial values produced by domain calculations. Not predicates, not
// lookups, not parsing. `Explained[bool]` for a validity check explains nothing
// anyone will read, and applying this universally would produce ceremony
// without safety.
package explained

import (
	"errors"
	"fmt"

	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
)

// ErrNoTrace is returned when a value is built without a derivation. It exists
// so the failure is a value a caller can test, not a panic.
var ErrNoTrace = errors.New("explained: value has no derivation")

// Value is a computed financial value paired with the trace that produced it.
//
// The fields are unexported and there is no literal construction: a Value comes
// from FromObservation or from a combinator, both of which produce a trace. A
// bare struct literal cannot manufacture an unexplained value.
type Value[T any] struct {
	value T
	trace provenance.Ref
	// The full record is carried alongside the reference so a caller can render
	// an explanation without a store lookup. The reference is what is
	// persisted; this is convenience for the same process.
	record provenance.DerivationRecord
}

// FromObservation lifts an observed value into the explained world.
//
// The base case: a value FDOS was told rather than computed. Its derivation
// records the interpreter that read it, so even an unprocessed observation can
// say where it came from.
func FromObservation[T any](
	value T,
	method provenance.Method,
	inputs []string,
	confidence provenance.Confidence,
) (Value[T], error) {
	record, err := provenance.NewDerivation(method, inputs, nil, nil, confidence)
	if err != nil {
		return Value[T]{}, err
	}
	return Value[T]{value: value, trace: record.Ref(), record: record}, nil
}

// Value returns the computed value.
//
// Named so that reaching past the explanation is visible at the call site. A
// caller that only wants the number can have it — the point is that producing
// it without a trace was never possible.
func (v Value[T]) Value() T { return v.value }

// Trace returns the content address of the derivation.
func (v Value[T]) Trace() provenance.Ref { return v.trace }

// Record returns the full derivation record.
func (v Value[T]) Record() provenance.DerivationRecord { return v.record }

// Confidence returns the trustworthiness of the computed value.
func (v Value[T]) Confidence() provenance.Confidence { return v.record.Confidence() }

// Map applies a total function to an explained value, extending the trace.
//
// For calculations that cannot fail. The trace is constructed here, not by the
// caller, which is what makes it impossible to forget.
func Map[A, B any](
	in Value[A],
	method provenance.Method,
	parameters []provenance.Parameter,
	references []provenance.ReferenceBinding,
	f func(A) B,
) (Value[B], error) {
	record, err := provenance.NewDerivation(
		method,
		[]string{in.trace.Hash()},
		parameters,
		references,
		in.record.Confidence(),
	)
	if err != nil {
		return Value[B]{}, err
	}
	return Value[B]{value: f(in.value), trace: record.Ref(), record: record}, nil
}

// TryMap is Map for a function that can fail.
func TryMap[A, B any](
	in Value[A],
	method provenance.Method,
	parameters []provenance.Parameter,
	references []provenance.ReferenceBinding,
	f func(A) (B, error),
) (Value[B], error) {
	out, err := f(in.value)
	if err != nil {
		return Value[B]{}, err
	}
	record, err := provenance.NewDerivation(
		method,
		[]string{in.trace.Hash()},
		parameters,
		references,
		in.record.Confidence(),
	)
	if err != nil {
		return Value[B]{}, err
	}
	return Value[B]{value: out, trace: record.Ref(), record: record}, nil
}

// Combine2 applies a binary operation to two explained values.
//
// Confidence propagates as the weakest of the inputs — not a product. A
// computed value is no more trustworthy than the least trustworthy thing it
// was computed from, and multiplying ordinal levels would be the arithmetic
// `provenance.Confidence` exists to refuse.
func Combine2[A, B, C any](
	a Value[A],
	b Value[B],
	method provenance.Method,
	parameters []provenance.Parameter,
	references []provenance.ReferenceBinding,
	f func(A, B) (C, error),
) (Value[C], error) {
	out, err := f(a.value, b.value)
	if err != nil {
		return Value[C]{}, err
	}
	record, err := provenance.NewDerivation(
		method,
		[]string{a.trace.Hash(), b.trace.Hash()},
		parameters,
		references,
		provenance.Weakest(a.record.Confidence(), b.record.Confidence()),
	)
	if err != nil {
		return Value[C]{}, err
	}
	return Value[C]{value: out, trace: record.Ref(), record: record}, nil
}

// Fold reduces explained values into one, recording every input.
//
// The important combinator: a thousand-input calculation would otherwise
// produce a thousand-entry inline trace. Content addressing keeps the lineage a
// DAG rather than a tree, and the record names each input by hash.
//
// Fold over an empty slice is an error rather than a seed: an aggregate with no
// inputs has nothing to explain, and silently returning the seed would produce
// a number whose trace claims it came from nowhere.
func Fold[A, B any](
	inputs []Value[A],
	seed B,
	method provenance.Method,
	parameters []provenance.Parameter,
	references []provenance.ReferenceBinding,
	f func(B, A) (B, error),
) (Value[B], error) {
	if len(inputs) == 0 {
		return Value[B]{}, fmt.Errorf("%w: fold over no inputs", ErrNoTrace)
	}

	acc := seed
	hashes := make([]string, 0, len(inputs))
	confidences := make([]provenance.Confidence, 0, len(inputs))

	for _, in := range inputs {
		var err error
		if acc, err = f(acc, in.value); err != nil {
			return Value[B]{}, err
		}
		hashes = append(hashes, in.trace.Hash())
		confidences = append(confidences, in.record.Confidence())
	}

	record, err := provenance.NewDerivation(
		method, hashes, parameters, references, provenance.Weakest(confidences...),
	)
	if err != nil {
		return Value[B]{}, err
	}
	return Value[B]{value: acc, trace: record.Ref(), record: record}, nil
}
