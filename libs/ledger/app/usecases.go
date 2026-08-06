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

// AcceptHoldingClaimCommand is what an external producer submits: identifiers
// exactly as it read them, and no identity (ADR-0029).
//
// This is the only entry point an external producer can call. Every other one
// takes an identity.ID, which ADR-0007 and ADR-0022 forbid a producer from
// minting — so before this existed the public application surface was
// structurally unusable from outside.
//
// Like ObserveHoldingCommand there is no knowledge-time field: the producer
// states when the holding was true and FDOS decides when it learned (ADR-0009).
type AcceptHoldingClaimCommand struct {
	Stream     string
	Account    identity.Claim
	Instrument identity.Claim
	Quantity   money.Quantity
	Effective  temporal.Interval

	Source      provenance.Source
	CollectedAt temporal.Instant
	Interpreter provenance.Interpreter
	Confidence  provenance.Confidence
	References  []provenance.ReferenceBinding
}

// AcceptHoldingClaim appends a HoldingClaimed fact and returns its reference.
//
// **It resolves nothing and mints nothing.** Resolution is a derivation
// recorded afterwards, never a precondition of appending (ADR-0022), and
// minting is a deliberate act with an owner rather than a consequence of
// somebody having posted something. An identity that came into existence
// because a stranger submitted a claim is an identity nobody chose.
//
// The claim is admitted or refused; what it means is decided later, by FDOS.
//
// # Admission revalidates
//
// Every check here is repeated regardless of what the caller ran. A producer
// may link a modified build of any helper FDOS publishes, so a library is a
// constructor that makes conforming easy and never a gate that prevents
// non-conforming (ADR-0029). This function assumes a hostile producer, because
// eventually there will be one.
//
// This is where the SourceRef grammar ADR-0028 specified is finally checked.
// That ADR shipped with admissible provenance at rung 6 — a rule nothing
// enforced — because there was no admission point to enforce it at. This is
// that point.
func (l *Ledger) AcceptHoldingClaim(ctx context.Context, cmd AcceptHoldingClaimCommand) (domain.Ref, error) {
	if err := cmd.Source.CheckContentAddress(); err != nil {
		return domain.Ref{}, fmt.Errorf("admission: %w", err)
	}
	if cmd.Account.IsZero() || cmd.Instrument.IsZero() {
		return domain.Ref{}, fmt.Errorf("%w: a holding claim names an account and an instrument",
			domain.ErrIncompleteEnvelope)
	}

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

	extended, ref, err := stream.Append(envelope, domain.KindObservation, domain.HoldingClaimed{
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

// UnresolvedClaimsQuery asks which admitted claims resolve to no identity.
//
// As-of is required and has no default, like every other query here: a default
// would answer with today's knowledge about a past coordinate, which is the
// look-ahead contamination bitemporality exists to prevent (ADR-0009).
type UnresolvedClaimsQuery struct {
	Stream string
	AsOf   temporal.AsOf
}

// UnresolvedClaims lists the claims waiting for an identity.
//
// Without it a producer publishes faithfully into silence: claims accumulate,
// nothing is derived, and nobody is told. Answering the question does not
// resolve anything — looking must not be what stops them waiting.
func (l *Ledger) UnresolvedClaims(
	ctx context.Context,
	q UnresolvedClaimsQuery,
) ([]domain.UnresolvedClaim, error) {
	stream, err := l.store.Load(ctx, q.Stream)
	if err != nil {
		return nil, fmt.Errorf("load: %w", err)
	}
	return domain.Unresolved(stream, q.AsOf), nil
}
