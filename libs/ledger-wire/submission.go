package ledgerwire

import (
	"fmt"

	kernelwire "github.com/FabioCaffarello/fdos/libs/kernel-wire"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"

	ingestv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ingest/v1"
	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
)

// EncodeHoldingClaimSubmission maps an admission command to the wire.
//
// The direction a producer needs, and the one FDOS itself has least use for —
// it exists so the round trip can be asserted in both directions. A codec that
// only decodes passes a one-way conformance test forever, because a field it
// never learned to write was never in the value it compares against.
func EncodeHoldingClaimSubmission(cmd app.AcceptHoldingClaimCommand) *ingestv1.HoldingClaimSubmission {
	out := &ingestv1.HoldingClaimSubmission{
		Stream:      cmd.Stream,
		Account:     kernelwire.EncodeClaim(cmd.Account),
		Instrument:  kernelwire.EncodeClaim(cmd.Instrument),
		Quantity:    kernelwire.EncodeQuantity(cmd.Quantity),
		Effective:   kernelwire.EncodeInterval(cmd.Effective),
		Source:      &kernelv1.SourceRef{Value: cmd.Source.String()},
		CollectedAt: kernelwire.EncodeInstant(cmd.CollectedAt),
		Interpreter: &kernelv1.InterpreterRef{
			Name:    cmd.Interpreter.Name(),
			Version: cmd.Interpreter.Version(),
		},
		Confidence: kernelwire.EncodeConfidence(cmd.Confidence),
	}
	for _, r := range cmd.References {
		out.References = append(out.References, kernelwire.EncodeReferenceBinding(r))
	}
	return out
}

// DecodeHoldingClaimSubmission maps a submission to an admission command.
//
// Every value goes through a kernel constructor rather than being assigned:
// assigning would let a submission carry something the domain would refuse to
// build, which is the whole reason the constructors exist.
//
// It does not validate the source grammar. That is admission's job, and doing
// it here would make it enforcement at decode — which does not reject bad new
// data, it makes existing data unreadable (ADR-0028).
func DecodeHoldingClaimSubmission(w *ingestv1.HoldingClaimSubmission) (app.AcceptHoldingClaimCommand, error) {
	if w == nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("%w: submission is nil", ErrMalformed)
	}

	account, err := kernelwire.DecodeClaim(w.GetAccount())
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("account: %w", err)
	}
	instrument, err := kernelwire.DecodeClaim(w.GetInstrument())
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("instrument: %w", err)
	}
	quantity, err := kernelwire.DecodeQuantity(w.GetQuantity())
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("quantity: %w", err)
	}
	effective, err := kernelwire.DecodeInterval(w.GetEffective())
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("effective: %w", err)
	}
	source, err := provenance.NewSource(w.GetSource().GetValue())
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("source: %w", err)
	}
	collectedAt, err := kernelwire.DecodeInstant(w.GetCollectedAt())
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("collected_at: %w", err)
	}
	interpreter, err := provenance.NewInterpreter(
		w.GetInterpreter().GetName(),
		w.GetInterpreter().GetVersion(),
	)
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("interpreter: %w", err)
	}
	confidence, err := kernelwire.DecodeConfidence(w.GetConfidence())
	if err != nil {
		return app.AcceptHoldingClaimCommand{}, fmt.Errorf("confidence: %w", err)
	}

	cmd := app.AcceptHoldingClaimCommand{
		Stream:      w.GetStream(),
		Account:     account,
		Instrument:  instrument,
		Quantity:    quantity,
		Effective:   effective,
		Source:      source,
		CollectedAt: collectedAt,
		Interpreter: interpreter,
		Confidence:  confidence,
	}
	for i, r := range w.GetReferences() {
		binding, bErr := kernelwire.DecodeReferenceBinding(r)
		if bErr != nil {
			return app.AcceptHoldingClaimCommand{}, fmt.Errorf("reference %d: %w", i, bErr)
		}
		cmd.References = append(cmd.References, binding)
	}
	return cmd, nil
}
