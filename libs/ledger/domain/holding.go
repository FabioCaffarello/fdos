package domain

import (
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
)

// HoldingObserved asserts that FDOS was *told* an account held a quantity of an
// instrument at some effective time.
//
// An Observation, not an Occurrence, and the distinction is the whole point
// (ADR-0011). A broker statement is a report about the world, not an event in
// it. Recording a statement line as though it were a trade fabricates a
// transaction that never took place — and once fabricated, no audit can tell
// the difference.
//
// Deriving occurrences from observations (what trades must have happened
// between two statements) is a domain computation with its own provenance,
// never an ingestion shortcut.
type HoldingObserved struct {
	Account    identity.ID
	Instrument identity.ID
	Quantity   money.Quantity
}

// FactType returns the domain-qualified type name.
func (HoldingObserved) FactType() string { return "ledger.HoldingObserved" }

// TypeVersion returns the per-type major version.
func (HoldingObserved) TypeVersion() uint32 { return 1 }

// CorrectionKind distinguishes three answers an auditor treats differently
// (ADR-0011). Collapsing them loses the difference between "we were wrong",
// "it did not happen" and "we now have a better source".
type CorrectionKind uint8

// Correction kinds. The zero value is invalid.
const (
	CorrectionKindUnspecified CorrectionKind = iota
	CorrectionKindCorrected                  // occurred, recorded with a wrong value
	CorrectionKindRetracted                  // should never have been recorded
	CorrectionKindSuperseded                 // a better-sourced fact replaces it
)

// String returns the kind name, matching fdos.ledger.v1.CorrectionKind.
func (c CorrectionKind) String() string {
	switch c {
	case CorrectionKindCorrected:
		return "corrected"
	case CorrectionKindRetracted:
		return "retracted"
	case CorrectionKindSuperseded:
		return "superseded"
	default:
		return "unspecified"
	}
}

// Valid reports whether the kind is one of the defined ones.
func (c CorrectionKind) Valid() bool {
	return c >= CorrectionKindCorrected && c <= CorrectionKindSuperseded
}

// FactCorrected is itself a fact, never a mutation.
//
// The corrected fact remains readable: asking the ledger as of a knowledge time
// before the correction returns the original, which is exactly what "what did we
// believe on date D" means and exactly what an audit requires.
type FactCorrected struct {
	Corrects Ref
	Kind     CorrectionKind
	Reason   string
}

// FactType returns the domain-qualified type name.
func (FactCorrected) FactType() string { return "ledger.FactCorrected" }

// TypeVersion returns the per-type major version.
func (FactCorrected) TypeVersion() uint32 { return 1 }
