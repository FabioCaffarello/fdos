// Package ledgerwire maps the FDOS ledger types to and from their protobuf wire
// form.
//
// The companion to `libs/kernel-wire`, and together they close B-003: ADR-0018
// created two definitions of every canonical concept and recorded that nothing
// prevented them diverging. The round-trip conformance suites are that
// mechanism.
//
// A separate module from `libs/ledger` for the same reason `libs/kernel-wire` is
// separate from the kernel: Go resolves dependencies per module, so a codec
// inside the context would put the protobuf runtime into the graph of everyone
// importing `libs/ledger/domain` (ADR-0013).
package ledgerwire

import (
	"errors"
	"fmt"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"

	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
	payloadv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ledger/payload/v1"
	ledgerv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ledger/v1"
	kernelwire "github.com/FabioCaffarello/fdos/libs/kernel-wire"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// protoMessage is the subset of proto.Message this package packs into an Any.
type protoMessage = proto.Message

// provenanceBinding aliases the kernel type so the envelope decoder reads
// clearly without importing the package name into every signature.
type provenanceBinding = provenance.ReferenceBinding

// ErrMalformed is returned when a wire message cannot produce a valid domain
// value.
//
// Decoding is validation, not assignment: every decode goes through the domain
// constructor, which is the only thing standing between data from outside the
// process and an invalid fact.
var ErrMalformed = errors.New("ledgerwire: malformed wire value")

// --- identity ----------------------------------------------------------------
//
// The identity and claim codecs live in `libs/kernel-wire`, where the directory
// contract says codecs for kernel types belong. They were here until the claim
// codec forced the question and made the drift visible; `identity.ID` and
// `identity.Claim` are kernel types and `fdos.kernel.v1.EntityId` and
// `IdentifierClaim` are kernel messages, so nothing about them was ever the
// ledger's.

// --- refs and envelope -------------------------------------------------------

// EncodeRef converts a fact reference to its wire form.
func EncodeRef(r domain.Ref) *kernelv1.FactRef {
	return &kernelv1.FactRef{Stream: r.Stream, Sequence: r.Sequence}
}

// DecodeRef rebuilds a fact reference.
func DecodeRef(w *kernelv1.FactRef) (domain.Ref, error) {
	if w == nil {
		return domain.Ref{}, fmt.Errorf("%w: fact ref is nil", ErrMalformed)
	}
	if w.GetStream() == "" || w.GetSequence() == 0 {
		return domain.Ref{}, fmt.Errorf("%w: fact ref %q#%d is incomplete", ErrMalformed, w.GetStream(), w.GetSequence())
	}
	return domain.Ref{Stream: w.GetStream(), Sequence: w.GetSequence()}, nil
}

// EncodeEnvelope converts an envelope to its wire form.
//
// The reference lives on the wire envelope but on the domain *fact*, so it is
// passed in rather than read from the envelope. That asymmetry is the one place
// the two models genuinely differ in shape.
func EncodeEnvelope(e domain.Envelope, ref domain.Ref) *ledgerv1.Envelope {
	references := e.References()
	wireRefs := make([]*kernelv1.ReferenceBinding, 0, len(references))
	for _, r := range references {
		wireRefs = append(wireRefs, kernelwire.EncodeReferenceBinding(r))
	}
	return &ledgerv1.Envelope{
		Temporal:   kernelwire.EncodeCoordinates(e.Coordinates()),
		Provenance: kernelwire.EncodeProvenance(e.Provenance()),
		References: wireRefs,
		Ref:        EncodeRef(ref),
	}
}

// DecodeEnvelope rebuilds an envelope and the reference it carried.
func DecodeEnvelope(w *ledgerv1.Envelope) (domain.Envelope, domain.Ref, error) {
	if w == nil {
		return domain.Envelope{}, domain.Ref{}, fmt.Errorf("%w: envelope is nil", ErrMalformed)
	}
	coordinates, err := kernelwire.DecodeCoordinates(w.GetTemporal())
	if err != nil {
		return domain.Envelope{}, domain.Ref{}, err
	}
	prov, err := kernelwire.DecodeProvenance(w.GetProvenance())
	if err != nil {
		return domain.Envelope{}, domain.Ref{}, err
	}
	ref, err := DecodeRef(w.GetRef())
	if err != nil {
		return domain.Envelope{}, domain.Ref{}, err
	}

	references := make([]provenanceBinding, 0, len(w.GetReferences()))
	for _, r := range w.GetReferences() {
		binding, bErr := kernelwire.DecodeReferenceBinding(r)
		if bErr != nil {
			return domain.Envelope{}, domain.Ref{}, bErr
		}
		references = append(references, binding)
	}

	envelope, err := domain.NewEnvelope(coordinates, prov, references)
	if err != nil {
		return domain.Envelope{}, domain.Ref{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return envelope, ref, nil
}

// --- facts -------------------------------------------------------------------

// EncodeFact converts a fact to its wire form, packing the payload into an Any.
func EncodeFact(f domain.Fact) (*ledgerv1.Fact, error) {
	payload, err := encodePayload(f.Payload())
	if err != nil {
		return nil, err
	}
	packed, err := anypb.New(payload)
	if err != nil {
		return nil, fmt.Errorf("pack payload: %w", err)
	}
	return &ledgerv1.Fact{
		Envelope:    EncodeEnvelope(f.Envelope(), f.Ref()),
		Kind:        encodeFactKind(f.Kind()),
		Type:        f.Type(),
		TypeVersion: f.TypeVersion(),
		Payload:     packed,
	}, nil
}

// DecodeFact rebuilds a fact through the domain constructor.
//
// The declared `type` is checked against the unpacked payload rather than
// trusted. A fact claiming to be a `ledger.TradeSettled` while carrying a
// holding observation is exactly the kind of thing that must not reach a
// projection.
func DecodeFact(w *ledgerv1.Fact) (domain.Fact, error) {
	if w == nil {
		return domain.Fact{}, fmt.Errorf("%w: fact is nil", ErrMalformed)
	}
	envelope, ref, err := DecodeEnvelope(w.GetEnvelope())
	if err != nil {
		return domain.Fact{}, err
	}
	kind, err := decodeFactKind(w.GetKind())
	if err != nil {
		return domain.Fact{}, err
	}
	payload, err := decodePayload(w.GetPayload())
	if err != nil {
		return domain.Fact{}, err
	}
	if payload.FactType() != w.GetType() {
		return domain.Fact{}, fmt.Errorf(
			"%w: fact declares type %q but carries %q",
			ErrMalformed, w.GetType(), payload.FactType())
	}
	if payload.TypeVersion() != w.GetTypeVersion() {
		return domain.Fact{}, fmt.Errorf(
			"%w: fact declares %s v%d but the payload is v%d",
			ErrMalformed, w.GetType(), w.GetTypeVersion(), payload.TypeVersion())
	}

	fact, err := domain.NewFact(ref, envelope, kind, payload)
	if err != nil {
		return domain.Fact{}, fmt.Errorf("%w: %w", ErrMalformed, err)
	}
	return fact, nil
}

func encodeFactKind(k domain.Kind) ledgerv1.FactKind {
	switch k {
	case domain.KindOccurrence:
		return ledgerv1.FactKind_FACT_KIND_OCCURRENCE
	case domain.KindObservation:
		return ledgerv1.FactKind_FACT_KIND_OBSERVATION
	default:
		return ledgerv1.FactKind_FACT_KIND_UNSPECIFIED
	}
}

func decodeFactKind(w ledgerv1.FactKind) (domain.Kind, error) {
	switch w {
	case ledgerv1.FactKind_FACT_KIND_OCCURRENCE:
		return domain.KindOccurrence, nil
	case ledgerv1.FactKind_FACT_KIND_OBSERVATION:
		return domain.KindObservation, nil
	default:
		return domain.KindUnspecified,
			fmt.Errorf("%w: fact kind %s has no domain equivalent", ErrMalformed, w)
	}
}

// --- payloads ----------------------------------------------------------------

// encodePayload dispatches on the domain payload type.
//
// An explicit switch with no reflective fallback: a new payload type must be
// added here deliberately, and forgetting produces an error at the first encode
// rather than a silently empty Any.
func encodePayload(p domain.Payload) (protoMessage, error) {
	switch v := p.(type) {
	case domain.HoldingObserved:
		return &payloadv1.HoldingObserved{
			Account:    kernelwire.EncodeEntityID(v.Account),
			Instrument: kernelwire.EncodeEntityID(v.Instrument),
			Quantity:   kernelwire.EncodeQuantity(v.Quantity),
		}, nil
	case domain.HoldingClaimed:
		return &payloadv1.HoldingClaimed{
			Account:    kernelwire.EncodeClaim(v.Account),
			Instrument: kernelwire.EncodeClaim(v.Instrument),
			Quantity:   kernelwire.EncodeQuantity(v.Quantity),
		}, nil
	case domain.EntityMinted:
		return &payloadv1.EntityMinted{
			Entity:   kernelwire.EncodeEntityID(v.Entity),
			BornFrom: kernelwire.EncodeClaim(v.BornFrom),
		}, nil
	case domain.FactCorrected:
		return &ledgerv1.Correction{
			Corrects: EncodeRef(v.Corrects),
			Kind:     encodeCorrectionKind(v.Kind),
			Reason:   v.Reason,
		}, nil
	default:
		return nil, fmt.Errorf("%w: payload type %s has no wire form", ErrMalformed, p.FactType())
	}
}

func decodePayload(w *anypb.Any) (domain.Payload, error) {
	if w == nil {
		return nil, fmt.Errorf("%w: payload is nil", ErrMalformed)
	}
	unpacked, err := w.UnmarshalNew()
	if err != nil {
		return nil, fmt.Errorf("%w: unpack payload: %w", ErrMalformed, err)
	}

	switch v := unpacked.(type) {
	case *payloadv1.HoldingObserved:
		account, aErr := kernelwire.DecodeEntityID(v.GetAccount())
		if aErr != nil {
			return nil, aErr
		}
		instrument, iErr := kernelwire.DecodeEntityID(v.GetInstrument())
		if iErr != nil {
			return nil, iErr
		}
		quantity, qErr := kernelwire.DecodeQuantity(v.GetQuantity())
		if qErr != nil {
			return nil, qErr
		}
		return domain.HoldingObserved{Account: account, Instrument: instrument, Quantity: quantity}, nil

	case *payloadv1.HoldingClaimed:
		account, aErr := kernelwire.DecodeClaim(v.GetAccount())
		if aErr != nil {
			return nil, aErr
		}
		instrument, iErr := kernelwire.DecodeClaim(v.GetInstrument())
		if iErr != nil {
			return nil, iErr
		}
		quantity, qErr := kernelwire.DecodeQuantity(v.GetQuantity())
		if qErr != nil {
			return nil, qErr
		}
		return domain.HoldingClaimed{Account: account, Instrument: instrument, Quantity: quantity}, nil

	case *payloadv1.EntityMinted:
		entity, eErr := kernelwire.DecodeEntityID(v.GetEntity())
		if eErr != nil {
			return nil, eErr
		}
		bornFrom, bErr := kernelwire.DecodeClaim(v.GetBornFrom())
		if bErr != nil {
			return nil, bErr
		}
		return domain.EntityMinted{Entity: entity, BornFrom: bornFrom}, nil

	case *ledgerv1.Correction:
		corrects, rErr := DecodeRef(v.GetCorrects())
		if rErr != nil {
			return nil, rErr
		}
		kind, kErr := decodeCorrectionKind(v.GetKind())
		if kErr != nil {
			return nil, kErr
		}
		return domain.FactCorrected{Corrects: corrects, Kind: kind, Reason: v.GetReason()}, nil

	default:
		return nil, fmt.Errorf("%w: wire payload %T has no domain equivalent", ErrMalformed, unpacked)
	}
}

func encodeCorrectionKind(k domain.CorrectionKind) ledgerv1.CorrectionKind {
	switch k {
	case domain.CorrectionKindCorrected:
		return ledgerv1.CorrectionKind_CORRECTION_KIND_CORRECTED
	case domain.CorrectionKindRetracted:
		return ledgerv1.CorrectionKind_CORRECTION_KIND_RETRACTED
	case domain.CorrectionKindSuperseded:
		return ledgerv1.CorrectionKind_CORRECTION_KIND_SUPERSEDED
	default:
		return ledgerv1.CorrectionKind_CORRECTION_KIND_UNSPECIFIED
	}
}

func decodeCorrectionKind(w ledgerv1.CorrectionKind) (domain.CorrectionKind, error) {
	switch w {
	case ledgerv1.CorrectionKind_CORRECTION_KIND_CORRECTED:
		return domain.CorrectionKindCorrected, nil
	case ledgerv1.CorrectionKind_CORRECTION_KIND_RETRACTED:
		return domain.CorrectionKindRetracted, nil
	case ledgerv1.CorrectionKind_CORRECTION_KIND_SUPERSEDED:
		return domain.CorrectionKindSuperseded, nil
	default:
		return domain.CorrectionKindUnspecified,
			fmt.Errorf("%w: correction kind %s has no domain equivalent", ErrMalformed, w)
	}
}
