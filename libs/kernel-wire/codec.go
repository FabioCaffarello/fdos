// Package kernelwire maps the FDOS kernel types to and from their protobuf
// wire form.
//
// A separate module, not a package inside `libs/kernel`, and the reason is the
// dependency graph. Go resolves dependencies per module: a codec living in the
// kernel would put the protobuf runtime into the graph of everyone importing
// `libs/kernel/money`, making Constitution §10 true at the package level and
// false at the level that decides what a consumer is actually coupled to
// (ADR-0013).
//
// This is also where the cost ADR-0018 recorded gets paid. Wire types and
// domain types are two definitions of the same concepts; nothing prevented them
// diverging. The round-trip conformance test in this package is that mechanism:
// domain → wire → domain must be the identity, for every type, over generated
// values.
package kernelwire

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/types/known/timestamppb"

	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// ErrMalformed is returned when a wire message cannot produce a valid domain
// value.
//
// Decoding is validation, not assignment. A wire message is data from outside
// the process, and the domain constructors are the only thing standing between
// it and an invalid value — so every decode goes through them rather than
// setting fields.
var ErrMalformed = errors.New("kernelwire: malformed wire value")

// --- identity ----------------------------------------------------------------

// EncodeEntityID converts an entity identifier to its wire form.
//
// The kind travels with the value. Without it a decoder cannot rebuild the
// domain type, and re-deriving is not an option — `Derive` assigns an identifier
// from a natural key, and an entity whose key has since changed would get a
// different one (ADR-0007).
func EncodeEntityID(id identity.ID) *kernelv1.EntityId {
	return &kernelv1.EntityId{Value: id.String(), Kind: encodeEntityKind(id.Kind())}
}

// DecodeEntityID rebuilds an entity identifier.
func DecodeEntityID(w *kernelv1.EntityId) (identity.ID, error) {
	if w == nil {
		return identity.ID{}, fmt.Errorf("%w: entity id is nil", ErrMalformed)
	}
	kind, err := decodeEntityKind(w.GetKind())
	if err != nil {
		return identity.ID{}, err
	}
	id, err := identity.Restore(kind, w.GetValue())
	if err != nil {
		return identity.ID{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return id, nil
}

func encodeEntityKind(k identity.Kind) kernelv1.EntityKind {
	switch k {
	case identity.KindInstrument:
		return kernelv1.EntityKind_ENTITY_KIND_INSTRUMENT
	case identity.KindParty:
		return kernelv1.EntityKind_ENTITY_KIND_PARTY
	case identity.KindAccount:
		return kernelv1.EntityKind_ENTITY_KIND_ACCOUNT
	case identity.KindLedgerStream:
		return kernelv1.EntityKind_ENTITY_KIND_LEDGER_STREAM
	default:
		return kernelv1.EntityKind_ENTITY_KIND_UNSPECIFIED
	}
}

func decodeEntityKind(w kernelv1.EntityKind) (identity.Kind, error) {
	switch w {
	case kernelv1.EntityKind_ENTITY_KIND_INSTRUMENT:
		return identity.KindInstrument, nil
	case kernelv1.EntityKind_ENTITY_KIND_PARTY:
		return identity.KindParty, nil
	case kernelv1.EntityKind_ENTITY_KIND_ACCOUNT:
		return identity.KindAccount, nil
	case kernelv1.EntityKind_ENTITY_KIND_LEDGER_STREAM:
		return identity.KindLedgerStream, nil
	default:
		return identity.KindUnspecified,
			fmt.Errorf("%w: entity kind %s has no domain equivalent", ErrMalformed, w)
	}
}

// EncodeClaim converts an identifier claim to its wire form.
//
// The value travels verbatim. A codec that trimmed or case-folded it would be
// making a resolution decision in a place with no provenance, and RFC-0007 is
// explicit that canonicalisation is a resolver's decision to record rather than
// a parser's to make silently.
func EncodeClaim(c identity.Claim) *kernelv1.IdentifierClaim {
	return &kernelv1.IdentifierClaim{Scheme: c.Scheme(), Value: c.Value()}
}

// DecodeClaim rebuilds an identifier claim.
//
// Through the domain constructor, which rejects a non-canonical scheme. A claim
// arriving with scheme "Ticker" is a claim FDOS cannot compare byte-exactly to
// one written "ticker", and admitting it would let two entities exist for one
// thing.
func DecodeClaim(w *kernelv1.IdentifierClaim) (identity.Claim, error) {
	if w == nil {
		return identity.Claim{}, fmt.Errorf("%w: identifier claim is nil", ErrMalformed)
	}
	c, err := identity.NewClaim(w.GetScheme(), w.GetValue())
	if err != nil {
		return identity.Claim{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return c, nil
}

// --- money -------------------------------------------------------------------

// EncodeMoney converts Money to its wire form.
func EncodeMoney(m money.Money) *kernelv1.Money {
	return &kernelv1.Money{
		Amount:   &kernelv1.Decimal{Value: m.String()},
		Currency: m.Currency().String(),
	}
}

// DecodeMoney rebuilds Money, validating through the domain constructor.
func DecodeMoney(w *kernelv1.Money) (money.Money, error) {
	if w == nil || w.GetAmount() == nil {
		return money.Money{}, fmt.Errorf("%w: money is nil", ErrMalformed)
	}
	m, err := money.Parse(w.GetAmount().GetValue(), w.GetCurrency())
	if err != nil {
		return money.Money{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return m, nil
}

// EncodeQuantity converts Quantity to its wire form.
func EncodeQuantity(q money.Quantity) *kernelv1.Quantity {
	return &kernelv1.Quantity{
		Amount: &kernelv1.Decimal{Value: q.String()},
		Unit:   q.Unit().String(),
	}
}

// DecodeQuantity rebuilds Quantity.
func DecodeQuantity(w *kernelv1.Quantity) (money.Quantity, error) {
	if w == nil || w.GetAmount() == nil {
		return money.Quantity{}, fmt.Errorf("%w: quantity is nil", ErrMalformed)
	}
	q, err := money.ParseQuantity(w.GetAmount().GetValue(), w.GetUnit())
	if err != nil {
		return money.Quantity{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return q, nil
}

// EncodeRoundingContext converts a RoundingContext to its wire form.
//
// The scale crosses only when the context fixes one, because absent and zero are
// different instructions: absent is "no scale constraint", zero is "round to
// whole units", and zero is JPY (ADR-0040). Writing zero for an absent scale
// would turn every unconstrained context into a yen one on the far side.
func EncodeRoundingContext(rc money.RoundingContext) *kernelv1.RoundingContext {
	w := &kernelv1.RoundingContext{
		Precision: rc.Precision(),
		Mode:      encodeRoundingMode(rc.Mode()),
	}
	if scale, ok := rc.Scale(); ok {
		w.Scale = &scale
	}
	return w
}

// DecodeRoundingContext rebuilds a RoundingContext.
func DecodeRoundingContext(w *kernelv1.RoundingContext) (money.RoundingContext, error) {
	if w == nil {
		return money.RoundingContext{}, fmt.Errorf("%w: rounding context is nil", ErrMalformed)
	}
	mode, err := decodeRoundingMode(w.GetMode())
	if err != nil {
		return money.RoundingContext{}, err
	}
	rc, err := money.NewRoundingContext(w.GetPrecision(), mode)
	if err != nil {
		return money.RoundingContext{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	// Presence, not GetScale(): the generated getter returns zero for an absent
	// field, and zero is a real scale. Reading it through the getter would make
	// every unconstrained context decode as "round to whole units".
	if w.Scale != nil {
		rc = rc.WithScale(w.GetScale())
	}
	return rc, nil
}

// The enum mappings are exhaustive switches with no default that silently
// succeeds. A new mode on either side becomes a compile or decode error rather
// than a value that quietly rounds the wrong way.
func encodeRoundingMode(m money.RoundingMode) kernelv1.RoundingMode {
	switch m {
	case money.RoundingModeHalfEven:
		return kernelv1.RoundingMode_ROUNDING_MODE_HALF_EVEN
	case money.RoundingModeHalfUp:
		return kernelv1.RoundingMode_ROUNDING_MODE_HALF_UP
	case money.RoundingModeDown:
		return kernelv1.RoundingMode_ROUNDING_MODE_DOWN
	case money.RoundingModeUp:
		return kernelv1.RoundingMode_ROUNDING_MODE_UP
	case money.RoundingModeCeiling:
		return kernelv1.RoundingMode_ROUNDING_MODE_CEILING
	case money.RoundingModeFloor:
		return kernelv1.RoundingMode_ROUNDING_MODE_FLOOR
	default:
		return kernelv1.RoundingMode_ROUNDING_MODE_UNSPECIFIED
	}
}

func decodeRoundingMode(w kernelv1.RoundingMode) (money.RoundingMode, error) {
	switch w {
	case kernelv1.RoundingMode_ROUNDING_MODE_HALF_EVEN:
		return money.RoundingModeHalfEven, nil
	case kernelv1.RoundingMode_ROUNDING_MODE_HALF_UP:
		return money.RoundingModeHalfUp, nil
	case kernelv1.RoundingMode_ROUNDING_MODE_DOWN:
		return money.RoundingModeDown, nil
	case kernelv1.RoundingMode_ROUNDING_MODE_UP:
		return money.RoundingModeUp, nil
	case kernelv1.RoundingMode_ROUNDING_MODE_CEILING:
		return money.RoundingModeCeiling, nil
	case kernelv1.RoundingMode_ROUNDING_MODE_FLOOR:
		return money.RoundingModeFloor, nil
	default:
		return money.RoundingModeUnspecified,
			fmt.Errorf("%w: rounding mode %s has no domain equivalent", ErrMalformed, w)
	}
}

// --- temporal ----------------------------------------------------------------

// EncodeInstant converts an Instant to a protobuf timestamp.
func EncodeInstant(i temporal.Instant) *timestamppb.Timestamp {
	if i.IsZero() {
		return nil
	}
	return timestamppb.New(i.Time())
}

// DecodeInstant rebuilds an Instant. A nil timestamp decodes to the zero
// Instant, which callers must treat as absent rather than as epoch.
func DecodeInstant(w *timestamppb.Timestamp) (temporal.Instant, error) {
	if w == nil {
		return temporal.Instant{}, nil
	}
	i, err := temporal.At(w.AsTime())
	if err != nil {
		return temporal.Instant{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return i, nil
}

// EncodeInterval converts an effective interval to its wire form.
func EncodeInterval(iv temporal.Interval) *kernelv1.EffectiveInterval {
	return &kernelv1.EffectiveInterval{
		From: EncodeInstant(iv.From()),
		To:   EncodeInstant(iv.To()), // nil when open
	}
}

// DecodeInterval rebuilds an effective interval.
func DecodeInterval(w *kernelv1.EffectiveInterval) (temporal.Interval, error) {
	if w == nil {
		return temporal.Interval{}, fmt.Errorf("%w: interval is nil", ErrMalformed)
	}
	from, err := DecodeInstant(w.GetFrom())
	if err != nil {
		return temporal.Interval{}, err
	}
	to, err := DecodeInstant(w.GetTo())
	if err != nil {
		return temporal.Interval{}, err
	}
	iv, err := temporal.NewInterval(from, to)
	if err != nil {
		return temporal.Interval{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return iv, nil
}

// EncodeCoordinates converts temporal coordinates to their wire form.
func EncodeCoordinates(c temporal.Coordinates) *kernelv1.TemporalCoordinates {
	return &kernelv1.TemporalCoordinates{
		Effective: EncodeInterval(c.Effective()),
		Knowledge: EncodeInstant(c.Knowledge()),
	}
}

// DecodeCoordinates rebuilds temporal coordinates.
func DecodeCoordinates(w *kernelv1.TemporalCoordinates) (temporal.Coordinates, error) {
	if w == nil {
		return temporal.Coordinates{}, fmt.Errorf("%w: coordinates are nil", ErrMalformed)
	}
	effective, err := DecodeInterval(w.GetEffective())
	if err != nil {
		return temporal.Coordinates{}, err
	}
	knowledge, err := DecodeInstant(w.GetKnowledge())
	if err != nil {
		return temporal.Coordinates{}, err
	}
	c, err := temporal.Assign(effective, knowledge)
	if err != nil {
		return temporal.Coordinates{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return c, nil
}

// --- provenance --------------------------------------------------------------

// EncodeConfidence converts an ordinal confidence to its wire form.
func EncodeConfidence(c provenance.Confidence) kernelv1.Confidence {
	switch c {
	case provenance.ConfidenceAsserted:
		return kernelv1.Confidence_CONFIDENCE_ASSERTED
	case provenance.ConfidenceDerived:
		return kernelv1.Confidence_CONFIDENCE_DERIVED
	case provenance.ConfidenceEstimated:
		return kernelv1.Confidence_CONFIDENCE_ESTIMATED
	case provenance.ConfidenceInferred:
		return kernelv1.Confidence_CONFIDENCE_INFERRED
	case provenance.ConfidenceDisputed:
		return kernelv1.Confidence_CONFIDENCE_DISPUTED
	default:
		return kernelv1.Confidence_CONFIDENCE_UNSPECIFIED
	}
}

// DecodeConfidence rebuilds an ordinal confidence.
func DecodeConfidence(w kernelv1.Confidence) (provenance.Confidence, error) {
	switch w {
	case kernelv1.Confidence_CONFIDENCE_ASSERTED:
		return provenance.ConfidenceAsserted, nil
	case kernelv1.Confidence_CONFIDENCE_DERIVED:
		return provenance.ConfidenceDerived, nil
	case kernelv1.Confidence_CONFIDENCE_ESTIMATED:
		return provenance.ConfidenceEstimated, nil
	case kernelv1.Confidence_CONFIDENCE_INFERRED:
		return provenance.ConfidenceInferred, nil
	case kernelv1.Confidence_CONFIDENCE_DISPUTED:
		return provenance.ConfidenceDisputed, nil
	default:
		return provenance.ConfidenceUnspecified,
			fmt.Errorf("%w: confidence %s has no domain equivalent", ErrMalformed, w)
	}
}

// EncodeReferenceBinding converts a pinned dataset version to its wire form.
func EncodeReferenceBinding(r provenance.ReferenceBinding) *kernelv1.ReferenceBinding {
	return &kernelv1.ReferenceBinding{Dataset: r.Dataset(), Version: r.Version()}
}

// DecodeReferenceBinding rebuilds a pinned dataset version.
func DecodeReferenceBinding(w *kernelv1.ReferenceBinding) (provenance.ReferenceBinding, error) {
	if w == nil {
		return provenance.ReferenceBinding{}, fmt.Errorf("%w: reference binding is nil", ErrMalformed)
	}
	r, err := provenance.NewReferenceBinding(w.GetDataset(), w.GetVersion())
	if err != nil {
		return provenance.ReferenceBinding{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return r, nil
}

// EncodeProvenance converts provenance to its wire form.
func EncodeProvenance(p provenance.Provenance) *kernelv1.Provenance {
	out := &kernelv1.Provenance{
		Source:      &kernelv1.SourceRef{Value: p.Source().String()},
		CollectedAt: EncodeInstant(p.CollectedAt()),
		PublishedAt: EncodeInstant(p.PublishedAt()), // nil when absent
		Interpreter: &kernelv1.InterpreterRef{
			Name:    p.Interpreter().Name(),
			Version: p.Interpreter().Version(),
		},
		Confidence: EncodeConfidence(p.Confidence()),
	}
	if ref, derived := p.Derivation(); derived {
		out.Derivation = &kernelv1.DerivationRef{ContentHash: ref.Hash()}
	}
	return out
}

// DecodeProvenance rebuilds provenance through the domain constructors, which
// are the only thing that can reject an incomplete envelope.
func DecodeProvenance(w *kernelv1.Provenance) (provenance.Provenance, error) {
	if w == nil {
		return provenance.Provenance{}, fmt.Errorf("%w: provenance is nil", ErrMalformed)
	}

	source, err := provenance.NewSource(w.GetSource().GetValue())
	if err != nil {
		return provenance.Provenance{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	collectedAt, err := DecodeInstant(w.GetCollectedAt())
	if err != nil {
		return provenance.Provenance{}, err
	}
	interpreter, err := provenance.NewInterpreter(
		w.GetInterpreter().GetName(), w.GetInterpreter().GetVersion(),
	)
	if err != nil {
		return provenance.Provenance{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	confidence, err := DecodeConfidence(w.GetConfidence())
	if err != nil {
		return provenance.Provenance{}, err
	}

	var out provenance.Provenance
	if d := w.GetDerivation(); d != nil {
		ref, refErr := provenance.NewRef(d.GetContentHash())
		if refErr != nil {
			return provenance.Provenance{}, fmt.Errorf("%w: %w", ErrMalformed, refErr)
		}
		out, err = provenance.Derived(source, collectedAt, interpreter, ref, confidence)
	} else {
		out, err = provenance.Observed(source, collectedAt, interpreter, confidence)
	}
	if err != nil {
		return provenance.Provenance{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}

	if published, pErr := DecodeInstant(w.GetPublishedAt()); pErr == nil && !published.IsZero() {
		out = out.WithPublishedAt(published)
	}
	return out, nil
}
