package app

import (
	"context"
	"errors"
	"fmt"

	"github.com/FabioCaffarello/fdos/libs/kernel/explained"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// Ledger is the application service. It owns the ports and contains no
// business rules: every decision about what a fact means lives in `domain`.
type Ledger struct {
	store Store
	clock Clock
}

// NewLedger wires the application service.
func NewLedger(store Store, clock Clock) (*Ledger, error) {
	if store == nil || clock == nil {
		return nil, errors.New("app: ledger needs a store and a clock")
	}
	return &Ledger{store: store, clock: clock}, nil
}

// ObserveHoldingCommand records that a source told FDOS what an account held.
//
// Note what is absent: there is no knowledge-time field. The caller states when
// the holding was true (`Effective`), and FDOS decides when it learned
// (ADR-0009). Accepting a knowledge time here would reintroduce backdating,
// which is the contamination bitemporality exists to prevent.
type ObserveHoldingCommand struct {
	Stream     string
	Account    identity.ID
	Instrument identity.ID
	Quantity   money.Quantity
	Effective  temporal.Interval

	Source      provenance.Source
	CollectedAt temporal.Instant
	Interpreter provenance.Interpreter
	Confidence  provenance.Confidence
	References  []provenance.ReferenceBinding
}

// ObserveHolding appends a HoldingObserved fact and returns its reference.
//
// This is the only place knowledge time is produced, and it comes from the
// injected clock rather than from the caller or from `time.Now()`.
func (l *Ledger) ObserveHolding(ctx context.Context, cmd ObserveHoldingCommand) (domain.Ref, error) {
	stream, err := l.loadOrCreate(ctx, cmd.Stream)
	if err != nil {
		return domain.Ref{}, err
	}

	coordinates, err := temporal.Assign(cmd.Effective, l.clock.Now())
	if err != nil {
		return domain.Ref{}, fmt.Errorf("assign coordinates: %w", err)
	}

	prov, err := provenance.Observed(cmd.Source, cmd.CollectedAt, cmd.Interpreter, cmd.Confidence)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("provenance: %w", err)
	}

	envelope, err := domain.NewEnvelope(coordinates, prov, cmd.References)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("envelope: %w", err)
	}

	extended, ref, err := stream.Append(envelope, domain.KindObservation, domain.HoldingObserved{
		Account:    cmd.Account,
		Instrument: cmd.Instrument,
		Quantity:   cmd.Quantity,
	})
	if err != nil {
		return domain.Ref{}, fmt.Errorf("append: %w", err)
	}

	if err := l.store.Save(ctx, extended); err != nil {
		return domain.Ref{}, fmt.Errorf("save: %w", err)
	}
	return ref, nil
}

// CorrectFactCommand records a correction. The corrected fact stays readable:
// asking as of a knowledge time before this correction still returns the
// original, which is what an audit requires (ADR-0011).
//
// There is deliberately **no effective interval**. A correction is a statement
// about a record, not about the world at a moment, so its effective interval is
// taken from the fact it corrects.
//
// Letting a caller choose one is a correctness hazard found by a failing test
// during M6: a retraction given a narrower interval than the fact it retracts
// is silently invisible to queries outside that window, and the position it was
// meant to remove quietly persists.
type CorrectFactCommand struct {
	Stream   string
	Corrects domain.Ref
	Kind     domain.CorrectionKind
	Reason   string

	Source      provenance.Source
	CollectedAt temporal.Instant
	Interpreter provenance.Interpreter
	Confidence  provenance.Confidence
}

// CorrectFact appends a correction as a new fact. Nothing is mutated.
func (l *Ledger) CorrectFact(ctx context.Context, cmd CorrectFactCommand) (domain.Ref, error) {
	stream, err := l.store.Load(ctx, cmd.Stream)
	if err != nil {
		return domain.Ref{}, err
	}
	corrected, err := stream.Get(cmd.Corrects)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("correct: %w", err)
	}
	if !cmd.Kind.Valid() {
		return domain.Ref{}, fmt.Errorf("%w: %d", domain.ErrUnknownKind, cmd.Kind)
	}

	// The correction is in force over exactly the interval the corrected fact
	// was, so it cannot be narrower and silently fail to apply.
	effective := corrected.Envelope().Coordinates().Effective()

	coordinates, err := temporal.Assign(effective, l.clock.Now())
	if err != nil {
		return domain.Ref{}, err
	}
	prov, err := provenance.Observed(cmd.Source, cmd.CollectedAt, cmd.Interpreter, cmd.Confidence)
	if err != nil {
		return domain.Ref{}, err
	}
	envelope, err := domain.NewEnvelope(coordinates, prov, nil)
	if err != nil {
		return domain.Ref{}, err
	}

	extended, ref, err := stream.Append(envelope, domain.KindObservation, domain.FactCorrected{
		Corrects: cmd.Corrects,
		Kind:     cmd.Kind,
		Reason:   cmd.Reason,
	})
	if err != nil {
		return domain.Ref{}, err
	}
	if err := l.store.Save(ctx, extended); err != nil {
		return domain.Ref{}, err
	}
	return ref, nil
}

// ProjectPositionQuery asks what an account held.
//
// `AsOf` is required and has no default. A projection that silently means "now"
// is how look-ahead bias enters an analysis, so there is no overload without it.
type ProjectPositionQuery struct {
	Stream     string
	Account    identity.ID
	Instrument identity.ID
	AsOf       temporal.AsOf
}

// ProjectPosition computes a position, explained.
//
// Returns Explained[Position]: the answer travels with the derivation that
// produced it, so "why is this 150 shares" is data rather than a story told
// afterwards (ADR-0012).
func (l *Ledger) ProjectPosition(
	ctx context.Context,
	q ProjectPositionQuery,
) (explained.Value[domain.Position], error) {
	stream, err := l.store.Load(ctx, q.Stream)
	if err != nil {
		return explained.Value[domain.Position]{}, err
	}
	return domain.ProjectPosition(stream, q.Account, q.Instrument, q.AsOf)
}

func (l *Ledger) loadOrCreate(ctx context.Context, name string) (domain.Stream, error) {
	stream, err := l.store.Load(ctx, name)
	if err == nil {
		return stream, nil
	}
	if !errors.Is(err, ErrStreamNotFound) {
		return domain.Stream{}, err
	}
	return domain.NewStream(name)
}
