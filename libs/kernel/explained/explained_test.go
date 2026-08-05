package explained_test

import (
	"errors"
	"testing"

	"github.com/FabioCaffarello/fdos/libs/kernel/explained"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
)

// The property FromDerivation exists for.
//
// Content addressing is only meaningful if two derivations share an address
// when, and only when, they are the same computation. A derivation whose
// parameters are dropped collides with every other derivation over the same
// inputs — including ones that produced a different answer. Two resolutions
// differing only by as-of coordinate are exactly that case, and the collision
// is invisible: both traces look complete.
func TestParametersChangeTheContentAddress(t *testing.T) {
	inputs := []string{"acct-1#1", "acct-1#2"}

	early, err := explained.FromDerivation(
		"150", method(t), inputs,
		[]provenance.Parameter{{Name: "as_of", Value: "2026-01-02"}},
		nil, provenance.ConfidenceAsserted,
	)
	if err != nil {
		t.Fatal(err)
	}
	late, err := explained.FromDerivation(
		"150", method(t), inputs,
		[]provenance.Parameter{{Name: "as_of", Value: "2031-01-02"}},
		nil, provenance.ConfidenceAsserted,
	)
	if err != nil {
		t.Fatal(err)
	}

	if early.Trace().Hash() == late.Trace().Hash() {
		t.Fatal("two derivations differing only by parameter share a content address")
	}
}

// The same computation must still deduplicate. An address that varied with
// nothing observable would make content addressing useless in the other
// direction.
func TestTheSameDerivationHasTheSameAddress(t *testing.T) {
	build := func() explained.Value[string] {
		t.Helper()
		v, err := explained.FromDerivation(
			"150", method(t), []string{"acct-1#1"},
			[]provenance.Parameter{{Name: "as_of", Value: "2026-01-02"}},
			nil, provenance.ConfidenceAsserted,
		)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}

	first, second := build(), build()
	if first.Trace().Hash() != second.Trace().Hash() {
		t.Fatalf("the same derivation produced two addresses: %s and %s",
			first.Trace(), second.Trace())
	}
}

// FromObservation is FromDerivation with nothing chosen. Stated as a test so
// the two entry points cannot drift into disagreeing about the base case.
func TestAnObservationIsADerivationWithNoParameters(t *testing.T) {
	observed, err := explained.FromObservation(
		"150", method(t), []string{"acct-1#1"}, provenance.ConfidenceAsserted,
	)
	if err != nil {
		t.Fatal(err)
	}
	derived, err := explained.FromDerivation(
		"150", method(t), []string{"acct-1#1"}, nil, nil, provenance.ConfidenceAsserted,
	)
	if err != nil {
		t.Fatal(err)
	}

	if observed.Trace().Hash() != derived.Trace().Hash() {
		t.Fatal("the two entry points disagree about the base case")
	}
	if got := len(observed.Record().Parameters()); got != 0 {
		t.Errorf("an observation recorded %d parameters", got)
	}
}

// A derivation with no method is not a derivation. The error is a value so a
// caller can test it, per the package contract.
func TestADerivationWithoutAMethodIsRefused(t *testing.T) {
	_, err := explained.FromDerivation(
		"150", provenance.Method{}, []string{"acct-1#1"}, nil, nil, provenance.ConfidenceAsserted,
	)
	if !errors.Is(err, provenance.ErrIncomplete) {
		t.Fatalf("expected ErrIncomplete, got %v", err)
	}
}

func method(t *testing.T) provenance.Method {
	t.Helper()
	m, err := provenance.NewMethod("test.Method", "1")
	if err != nil {
		t.Fatal(err)
	}
	return m
}
