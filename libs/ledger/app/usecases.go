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

// ErrAlreadyMinted is returned when a claim already resolves to an identity.
//
// A refusal rather than a silent success. Returning the existing identity as
// though a mint had happened is how a caller learns to stop checking, and the
// second caller to stop checking is the one that mints something it should not
// (ADR-0033).
var ErrAlreadyMinted = errors.New("app: claim already resolves to a minted identity")

// Ledger is the application service. It owns the ports and contains no
// business rules: every decision about what a fact means lives in `domain`.
type Ledger struct {
	store Store
	clock Clock
	rules identity.Ruleset
}

// NewLedger wires the application service.
//
// The ruleset is required and has no default, including no default of "the one
// this build ships". Which rules are deployed determines which claims resolve
// together — that is, which real-world things FDOS treats as one thing — so it
// is a composition-root decision to be made rather than defaulted into
// (ADR-0033). `identity.Canonicalisation()` is what a caller almost always
// wants; being made to name it is the point.
func NewLedger(store Store, clock Clock, rules identity.Ruleset) (*Ledger, error) {
	if store == nil || clock == nil {
		return nil, errors.New("app: ledger needs a store and a clock")
	}
	if rules.IsZero() {
		return nil, errors.New(
			"app: ledger needs a canonicalisation ruleset; pass identity.Canonicalisation() to get the one this build ships")
	}
	return &Ledger{store: store, clock: clock, rules: rules}, nil
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

	// Any(): this observation is stated by its caller and depends on nothing
	// read from the stream, so there is nothing that can have gone stale.
	ref, err := l.store.Append(ctx, cmd.Stream, Any(), envelope, domain.KindObservation,
		domain.HoldingObserved{
			Account:    cmd.Account,
			Instrument: cmd.Instrument,
			Quantity:   cmd.Quantity,
		})
	if err != nil {
		return domain.Ref{}, fmt.Errorf("append: %w", err)
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

	// AtLength: this correction was built from a fact read out of the stream,
	// and its effective interval was taken from that fact. If the stream moved,
	// the read is stale.
	ref, err := l.store.Append(ctx, cmd.Stream, AtLength(stream.Len()), envelope,
		domain.KindObservation, domain.FactCorrected{
			Corrects: cmd.Corrects,
			Kind:     cmd.Kind,
			Reason:   cmd.Reason,
		})
	if err != nil {
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

	// Any(), and note that admission no longer reads the stream at all. A claim
	// is judged on its own merits — nothing about whether it may be admitted
	// depends on what else is in the ledger — so there is no read to go stale,
	// and a producer's submission cannot be rejected because somebody else wrote
	// first (ADR-0034).
	ref, err := l.store.Append(ctx, cmd.Stream, Any(), envelope, domain.KindObservation,
		domain.HoldingClaimed{
			Account:    cmd.Account,
			Instrument: cmd.Instrument,
			Quantity:   cmd.Quantity,
		})
	if err != nil {
		return domain.Ref{}, fmt.Errorf("append: %w", err)
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
	return domain.Unresolved(stream, l.rules, q.AsOf), nil
}

// MintIdentityCommand brings an identity into existence for a claim (ADR-0033).
//
// `Answers` is the claim fact this mint answers, and it is optional. RFC-0007
// established that an account is often minted from operator configuration
// before any observation arrives — the operator configured the connector to
// fetch it, so the claim exists before any provider states it. Requiring a fact
// reference would make that case unrepresentable.
//
// Like every other command here there is no knowledge-time field: FDOS decides
// when it learned (ADR-0009).
//
// `Source` and `Interpreter` are the authority. They are **recorded, never
// checked** — FDOS cannot verify that the named authority is who it says or
// that it was entitled to mint, because there is no actor model and building
// one is D2. ADR-0033 records that at rung 6, and the boundary today is the
// process boundary: whoever can call this can mint.
type MintIdentityCommand struct {
	Stream    string
	Kind      identity.Kind
	Claim     identity.Claim
	Answers   domain.Ref
	Effective temporal.Interval

	Source      provenance.Source
	CollectedAt temporal.Instant
	Interpreter provenance.Interpreter
	Confidence  provenance.Confidence
	References  []provenance.ReferenceBinding
}

// MintIdentity records that an identity came into existence, and returns its
// reference.
//
// **This is the only thing in the repository that appends an EntityMinted
// fact**, and that is the decision rather than an implementation detail
// (ADR-0033):
//
//   - It is not admission. `AcceptHoldingClaim` mints nothing and continues to
//     mint nothing. An identity that came into existence because a stranger
//     submitted a claim is an identity nobody chose.
//   - It is not inspection. `UnresolvedClaims` mints nothing and continues to
//     mint nothing. Minting on inspection would make the act of looking change
//     the ledger.
//   - It resolves first and refuses. A claim that already resolves returns
//     ErrAlreadyMinted naming the identity, rather than quietly appending a
//     second mint or quietly returning the first.
//
// The fact is an **occurrence**, not an observation. Nobody told FDOS that this
// identity exists; FDOS decided it, and the decision happened (ADR-0011). An
// observation would assert that some source stated the identity, which is false
// of every mint.
//
// Its provenance is `Derived`, for the same reason: the identity was computed
// from the claim by a versioned method, and the derivation names that method,
// the claim fact answered, and the ruleset under which the seed was
// canonicalised.
func (l *Ledger) MintIdentity(ctx context.Context, cmd MintIdentityCommand) (domain.Ref, error) {
	if err := cmd.Source.CheckContentAddress(); err != nil {
		return domain.Ref{}, fmt.Errorf("mint: %w", err)
	}
	if !cmd.Kind.Valid() {
		return domain.Ref{}, fmt.Errorf("%w: %d", identity.ErrUnknownKind, cmd.Kind)
	}
	if cmd.Claim.IsZero() {
		return domain.Ref{}, fmt.Errorf("%w: a mint names the claim it is born from",
			domain.ErrIncompleteEnvelope)
	}

	stream, err := l.load(ctx, cmd.Stream)
	if err != nil {
		return domain.Ref{}, err
	}
	resolvedAt := stream.Len()

	knowledge := l.clock.Now()

	// Resolve before minting, at the coordinate this mint would occupy. Asking
	// at any other coordinate would answer a question nobody asked: a mint
	// effective from a later date must not be refused because of one that is
	// not yet in force.
	asOf, err := temporal.NewAsOf(cmd.Effective.From(), knowledge)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("mint: %w", err)
	}
	existing, resolveErr := domain.Resolve(stream, cmd.Claim, l.rules, asOf)
	switch {
	case resolveErr == nil:
		return domain.Ref{}, fmt.Errorf("%w: %s is already %s", ErrAlreadyMinted, cmd.Claim, existing)
	case !errors.Is(resolveErr, domain.ErrUnresolved):
		// Any other failure is a genuine error and must not be read as "not
		// minted yet" — that reading would mint on top of a stream it could not
		// examine.
		return domain.Ref{}, fmt.Errorf("mint: %w", resolveErr)
	}

	var answers []string
	if cmd.Answers.Stream != "" {
		answers = []string{cmd.Answers.String()}
	}

	minted, err := domain.MintFor(cmd.Kind, cmd.Claim, l.rules, answers, cmd.Confidence)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("mint: %w", err)
	}

	coordinates, err := temporal.Assign(cmd.Effective, knowledge)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("assign coordinates: %w", err)
	}

	prov, err := provenance.Derived(
		cmd.Source, cmd.CollectedAt, cmd.Interpreter, minted.Trace(), cmd.Confidence)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("provenance: %w", err)
	}

	envelope, err := domain.NewEnvelope(coordinates, prov, cmd.References)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("envelope: %w", err)
	}

	// AtLength(resolvedAt) is the whole reason Expectation exists. This resolved
	// against the stream and found nothing; a mint that landed since may already
	// answer the claim, and appending anyway would produce the two-mints-for-one-
	// claim duplication ADR-0033 refuses — arriving as a race rather than as a
	// caller error. ErrStaleRead tells the caller to resolve again.
	ref, err := l.store.Append(
		ctx, cmd.Stream, AtLength(resolvedAt), envelope, domain.KindOccurrence, minted.Value())
	if err != nil {
		return domain.Ref{}, fmt.Errorf("append: %w", err)
	}
	return ref, nil
}

// load reads a stream, treating an absent one as empty.
//
// A stream that has never been written and a stream with no facts are the same
// thing to a writer: there is nothing to have gone stale, and the store creates
// it on first append.
func (l *Ledger) load(ctx context.Context, name string) (domain.Stream, error) {
	stream, err := l.store.Load(ctx, name)
	if err == nil {
		return stream, nil
	}
	if !errors.Is(err, ErrStreamNotFound) {
		return domain.Stream{}, err
	}
	return domain.NewStream(name)
}

// ObserveClaimedHoldingCommand derives an observed holding from an admitted
// claim (ADR-0022).
//
// The last step of the acquisition path: a claim reaches the ledger, identities
// are minted for what it names, and then — and only then — the observation is
// derived from both.
type ObserveClaimedHoldingCommand struct {
	Stream string
	Claim  domain.Ref
	AsOf   temporal.AsOf

	Source      provenance.Source
	CollectedAt temporal.Instant
	Interpreter provenance.Interpreter
}

// ObserveClaimedHolding appends the HoldingObserved derived from a claim fact.
//
// The observation's provenance is `Derived` and names the derivation that
// produced it — the claim fact and both mints consumed. That is the obligation
// ADR-0022 recorded at rung 6 for want of a caller; here there is one, and it
// is met by construction because `domain.DeriveHoldingObserved` returns the
// derivation record and nothing else can build this fact.
//
// Confidence is not a parameter. It propagates from the claim as the weakest
// input, and letting a caller assert a stronger one here would let an
// observation be more trustworthy than the claim it came from.
func (l *Ledger) ObserveClaimedHolding(
	ctx context.Context,
	cmd ObserveClaimedHoldingCommand,
) (domain.Ref, error) {
	stream, err := l.store.Load(ctx, cmd.Stream)
	if err != nil {
		return domain.Ref{}, err
	}
	claimed, err := stream.Get(cmd.Claim)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("observe: %w", err)
	}

	observed, err := domain.DeriveHoldingObserved(stream, claimed, l.rules, cmd.AsOf)
	if err != nil {
		return domain.Ref{}, err
	}

	// The observation is in force over exactly the interval the claim was. A
	// derivation does not know something the evidence did not.
	effective := claimed.Envelope().Coordinates().Effective()
	coordinates, err := temporal.Assign(effective, l.clock.Now())
	if err != nil {
		return domain.Ref{}, err
	}

	prov, err := provenance.Derived(
		cmd.Source,
		cmd.CollectedAt,
		cmd.Interpreter,
		observed.Trace(),
		observed.Record().Confidence(),
	)
	if err != nil {
		return domain.Ref{}, fmt.Errorf("provenance: %w", err)
	}

	envelope, err := domain.NewEnvelope(coordinates, prov, claimed.Envelope().References())
	if err != nil {
		return domain.Ref{}, fmt.Errorf("envelope: %w", err)
	}

	// AtLength: the derivation named the mints visible in this stream, and its
	// content address depends on them. A mint appended since could change which
	// identity the claim resolves to, so the derivation must be redone rather
	// than recorded against a stream it no longer describes.
	ref, err := l.store.Append(ctx, cmd.Stream, AtLength(stream.Len()), envelope,
		domain.KindObservation, observed.Value())
	if err != nil {
		return domain.Ref{}, fmt.Errorf("append: %w", err)
	}
	return ref, nil
}
