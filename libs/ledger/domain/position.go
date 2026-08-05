package domain

import (
	"errors"
	"fmt"

	"github.com/FabioCaffarello/fdos/libs/kernel/explained"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// ErrNoFacts is returned when a projection has nothing to compute from.
//
// An error rather than a zero position: a position of zero is a claim about the
// world, and "we hold nothing" is different from "we know nothing". Returning
// zero for both would make an absence of data indistinguishable from an
// asserted absence.
var ErrNoFacts = errors.New("ledger: no visible facts to project")

// projectionMethod names the calculation for the derivation record. Versioned:
// changing how a position is computed is a new method version, pinned in the
// report so a regeneration years later uses the method of the time (ADR-0010).
var projectionMethod = mustMethod("ledger.ProjectPosition", "1")

// Position is the answer to a question, not a stored thing.
//
// There is no way to persist one: it has no envelope, cannot become a Payload,
// and every field is a value. This is Constitution §1 made structural — a
// position is the return value of a fold over facts, and if the facts change,
// the answer changes with them because there is no cached copy to go stale.
type Position struct {
	Account    identity.ID
	Instrument identity.ID
	Quantity   money.Quantity
	AsOf       temporal.AsOf
}

// ProjectPosition folds the visible observations into a position, explained.
//
// Returns Explained[Position], never a bare Position: a computed financial
// value on the FDOS domain surface always travels with the derivation that
// produced it (ADR-0012). The trace names every fact consumed, so the answer to
// "why is this 150 shares" is data rather than a story told afterwards.
//
// `asOf` is required. A projection with no temporal coordinate is a projection
// that silently means "now", which is how look-ahead bias enters an analysis.
//
// Later observations replace earlier ones rather than accumulating: an
// observation asserts what the holding *is*, not what changed. Summing
// statements would double-count, which is precisely the error the
// Occurrence/Observation split exists to prevent.
func ProjectPosition(
	stream Stream,
	account identity.ID,
	instrument identity.ID,
	asOf temporal.AsOf,
) (explained.Value[Position], error) {
	visible := stream.VisibleAt(asOf)

	retracted := make(map[Ref]bool)
	for _, c := range stream.Corrections(asOf) {
		if c.Kind == CorrectionKindRetracted {
			retracted[c.Corrects] = true
		}
	}

	observations := make([]explained.Value[HoldingObserved], 0, len(visible))
	for _, f := range visible {
		if retracted[f.Ref()] {
			continue
		}
		holding, ok := f.Payload().(HoldingObserved)
		if !ok {
			continue
		}
		if !holding.Account.Equal(account) || !holding.Instrument.Equal(instrument) {
			continue
		}

		observed, err := explained.FromObservation(
			holding,
			projectionMethod,
			[]string{f.Ref().String()},
			f.Envelope().Provenance().Confidence(),
		)
		if err != nil {
			return explained.Value[Position]{}, err
		}
		observations = append(observations, observed)
	}

	if len(observations) == 0 {
		return explained.Value[Position]{}, fmt.Errorf("%w: %s", ErrNoFacts, asOf)
	}

	seed := Position{Account: account, Instrument: instrument, AsOf: asOf}

	// VisibleAt returns a deterministic total order, so the last observation is
	// well defined even when two share both temporal coordinates.
	return explained.Fold(
		observations,
		seed,
		projectionMethod,
		[]provenance.Parameter{
			{Name: "account", Value: account.String()},
			{Name: "instrument", Value: instrument.String()},
			{Name: "as_of", Value: asOf.String()},
		},
		nil,
		func(acc Position, obs HoldingObserved) (Position, error) {
			acc.Quantity = obs.Quantity
			return acc, nil
		},
	)
}

func mustMethod(name, version string) provenance.Method {
	m, err := provenance.NewMethod(name, version)
	if err != nil {
		panic(err)
	}
	return m
}
