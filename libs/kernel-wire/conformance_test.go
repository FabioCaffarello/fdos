package kernelwire_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"pgregory.net/rapid"

	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
	kernelwire "github.com/FabioCaffarello/fdos/libs/kernel-wire"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
)

// This file is the mechanism that closes B-003.
//
// ADR-0018 created two definitions of every canonical concept — the protobuf
// wire types and the Go domain types — and recorded that nothing prevented them
// diverging. That was the largest unpaid cost of the decision.
//
// Two properties, checked over generated values:
//
//	domain -> wire -> domain  is the identity   (nothing is lost encoding)
//	wire   -> domain -> wire  is the identity   (nothing is dropped decoding)
//
// The second is what catches a decoder silently ignoring a field. Without it, a
// codec that never reads `published_at` would pass the first property forever,
// because the value it fails to carry was never in the domain value it compares
// against.

func TestEntityIDRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		kind := rapid.SampledFrom([]identity.Kind{
			identity.KindInstrument,
			identity.KindParty,
			identity.KindAccount,
			identity.KindLedgerStream,
		}).Draw(t, "kind")
		original := identity.MustDerive(kind, rapid.StringMatching(`[a-z0-9:_-]{1,24}`).Draw(t, "seed"))

		wire := kernelwire.EncodeEntityID(original)
		back, err := kernelwire.DecodeEntityID(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if !back.Equal(original) {
			t.Fatalf("domain round trip lost information: %s/%s -> %s/%s",
				original.Kind(), original, back.Kind(), back)
		}
		if !proto.Equal(wire, kernelwire.EncodeEntityID(back)) {
			t.Fatal("wire round trip is not the identity: a field was dropped decoding")
		}
	})
}

// A claim's value travels verbatim, which is the property RFC-0007 rests on: a
// codec that trimmed or case-folded would be making a resolution decision in a
// place with no provenance. The generator deliberately produces values with
// leading and trailing space, mixed case, and inner punctuation.
func TestClaimRoundTripsVerbatim(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		scheme := rapid.StringMatching(`[a-z][a-z0-9_]{0,15}`).Draw(t, "scheme")
		value := rapid.StringMatching(`[ ]{0,2}[A-Za-z0-9./ -]{1,24}[ ]{0,2}`).Draw(t, "value")

		// Not a skip: the generators produce a canonical scheme and a non-empty
		// value by construction, so every draw must be constructible. A skip here
		// would let the property pass while covering nothing.
		original, err := identity.NewClaim(scheme, value)
		if err != nil {
			t.Fatalf("generated claim is not constructible: %v", err)
		}

		wire := kernelwire.EncodeClaim(original)
		back, dErr := kernelwire.DecodeClaim(wire)
		if dErr != nil {
			t.Fatalf("decode: %v", dErr)
		}

		if back.Value() != original.Value() {
			t.Fatalf("the codec altered the value: %q -> %q", original.Value(), back.Value())
		}
		if !back.Equal(original) {
			t.Fatalf("domain round trip lost information: %s -> %s", original, back)
		}
		if !proto.Equal(wire, kernelwire.EncodeClaim(back)) {
			t.Fatal("wire round trip is not the identity: a field was dropped decoding")
		}
	})
}

// The decoder goes through the domain constructor, so a scheme the domain
// refuses cannot enter through the wire. "Ticker" and "ticker" would be two
// entities for one thing.
func TestDecodingRejectsANonCanonicalScheme(t *testing.T) {
	for _, scheme := range []string{"Ticker", "TICKER", " ticker", "ticker "} {
		if _, err := kernelwire.DecodeClaim(&kernelv1.IdentifierClaim{
			Scheme: scheme, Value: "PETR4",
		}); err == nil {
			t.Errorf("scheme %q decoded without error", scheme)
		}
	}
}

// Every entity kind must survive. A kind that decoded to a neighbour would file
// an account's facts against an instrument.
func TestEveryEntityKindRoundTrips(t *testing.T) {
	for _, kind := range []identity.Kind{
		identity.KindInstrument, identity.KindParty,
		identity.KindAccount, identity.KindLedgerStream,
	} {
		original := identity.MustDerive(kind, "seed")
		back, err := kernelwire.DecodeEntityID(kernelwire.EncodeEntityID(original))
		if err != nil {
			t.Fatalf("%s: decode: %v", kind, err)
		}
		if !back.Equal(original) {
			t.Errorf("%s: %s decoded as %s", kind, original, back)
		}
	}

	unspecified := kernelwire.EncodeEntityID(identity.ID{})
	if _, err := kernelwire.DecodeEntityID(unspecified); err == nil {
		t.Error("an unspecified entity kind decoded without error")
	}
}

func TestMoneyRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := money.MustParse(genDecimalText().Draw(t, "amount"), genCurrency().Draw(t, "currency"))

		wire := kernelwire.EncodeMoney(original)
		back, err := kernelwire.DecodeMoney(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if back.String() != original.String() || back.Currency() != original.Currency() {
			t.Fatalf("domain round trip lost information: %s %s -> %s %s",
				original, original.Currency(), back, back.Currency())
		}
		if !proto.Equal(wire, kernelwire.EncodeMoney(back)) {
			t.Fatal("wire round trip is not the identity: a field was dropped decoding")
		}
	})
}

func TestQuantityRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := money.MustParseQuantity(
			genDecimalText().Draw(t, "amount"),
			rapid.SampledFrom([]string{"share", "bond.face", "oz.t"}).Draw(t, "unit"),
		)

		wire := kernelwire.EncodeQuantity(original)
		back, err := kernelwire.DecodeQuantity(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if back.String() != original.String() || back.Unit() != original.Unit() {
			t.Fatalf("lost information: %s %s -> %s %s", original, original.Unit(), back, back.Unit())
		}
		if !proto.Equal(wire, kernelwire.EncodeQuantity(back)) {
			t.Fatal("wire round trip is not the identity")
		}
	})
}

// Rounding context matters more than most: a mode that decoded to the wrong
// value would round money the wrong way, silently and everywhere.
func TestRoundingContextRoundTrips(t *testing.T) {
	modes := []money.RoundingMode{
		money.RoundingModeHalfEven, money.RoundingModeHalfUp, money.RoundingModeDown,
		money.RoundingModeUp, money.RoundingModeCeiling, money.RoundingModeFloor,
	}
	rapid.Check(t, func(t *rapid.T) {
		original := money.MustRoundingContext(
			rapid.Uint32Range(1, 96).Draw(t, "precision"),
			rapid.SampledFrom(modes).Draw(t, "mode"),
		)

		wire := kernelwire.EncodeRoundingContext(original)
		back, err := kernelwire.DecodeRoundingContext(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if back.Precision() != original.Precision() || back.Mode() != original.Mode() {
			t.Fatalf("lost information: %s -> %s", original, back)
		}
		if !proto.Equal(wire, kernelwire.EncodeRoundingContext(back)) {
			t.Fatal("wire round trip is not the identity")
		}
	})
}

func TestTemporalCoordinatesRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		effective := genInterval().Draw(t, "effective")
		knowledge := genInstant().Draw(t, "knowledge")

		original, err := temporal.Assign(effective, knowledge)
		if err != nil {
			t.Fatalf("assign: %v", err)
		}

		wire := kernelwire.EncodeCoordinates(original)
		back, err := kernelwire.DecodeCoordinates(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if !back.Knowledge().Equal(original.Knowledge()) {
			t.Fatalf("knowledge time lost: %s -> %s", original.Knowledge(), back.Knowledge())
		}
		if !back.Effective().From().Equal(original.Effective().From()) {
			t.Fatalf("effective start lost: %s -> %s", original.Effective(), back.Effective())
		}
		if back.Effective().IsOpen() != original.Effective().IsOpen() {
			t.Fatalf("openness lost: %s -> %s", original.Effective(), back.Effective())
		}
		if !original.Effective().IsOpen() && !back.Effective().To().Equal(original.Effective().To()) {
			t.Fatalf("effective end lost: %s -> %s", original.Effective(), back.Effective())
		}
		if !proto.Equal(wire, kernelwire.EncodeCoordinates(back)) {
			t.Fatal("wire round trip is not the identity")
		}
	})
}

// Every defined confidence level must survive. An ordinal that decoded to a
// neighbouring level would silently change how much a number is trusted.
func TestEveryConfidenceLevelRoundTrips(t *testing.T) {
	for _, level := range []provenance.Confidence{
		provenance.ConfidenceAsserted, provenance.ConfidenceDerived,
		provenance.ConfidenceEstimated, provenance.ConfidenceInferred,
		provenance.ConfidenceDisputed,
	} {
		back, err := kernelwire.DecodeConfidence(kernelwire.EncodeConfidence(level))
		if err != nil {
			t.Fatalf("%s: decode: %v", level, err)
		}
		if back != level {
			t.Errorf("%s decoded as %s", level, back)
		}
	}

	// The unspecified level has no domain equivalent and must be rejected
	// rather than mapped to a plausible neighbour.
	if _, err := kernelwire.DecodeConfidence(kernelwire.EncodeConfidence(provenance.ConfidenceUnspecified)); err == nil {
		t.Error("unspecified confidence decoded without error")
	}
}

// Provenance is the envelope §6 depends on. If any part of it fails to survive
// the wire, a fact arrives unable to say where it came from.
func TestProvenanceRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		source, err := provenance.NewSource(rapid.StringMatching(`[a-z][a-z.]{2,20}`).Draw(t, "source"))
		if err != nil {
			t.Fatalf("source: %v", err)
		}
		interpreter, err := provenance.NewInterpreter(
			rapid.StringMatching(`[a-z][a-z-]{2,12}`).Draw(t, "parser"),
			rapid.StringMatching(`[0-9]\.[0-9]\.[0-9]`).Draw(t, "version"),
		)
		if err != nil {
			t.Fatalf("interpreter: %v", err)
		}
		collectedAt := genInstant().Draw(t, "collected")
		confidence := rapid.SampledFrom([]provenance.Confidence{
			provenance.ConfidenceAsserted, provenance.ConfidenceDerived,
			provenance.ConfidenceEstimated, provenance.ConfidenceInferred,
			provenance.ConfidenceDisputed,
		}).Draw(t, "confidence")

		original, err := provenance.Observed(source, collectedAt, interpreter, confidence)
		if err != nil {
			t.Fatalf("observed: %v", err)
		}

		// published_at is the field a lazy decoder drops: it is optional, so
		// omitting it never fails the domain round trip on its own.
		if rapid.Bool().Draw(t, "published") {
			original = original.WithPublishedAt(genInstant().Draw(t, "publishedAt"))
		}

		// A derived fact carries a derivation reference. This is the path that
		// needed provenance.NewRef to exist at all.
		if rapid.Bool().Draw(t, "derived") {
			method, mErr := provenance.NewMethod("test.Method", "1")
			if mErr != nil {
				t.Fatalf("method: %v", mErr)
			}
			record, dErr := provenance.NewDerivation(method, []string{"s#1"}, nil, nil, confidence)
			if dErr != nil {
				t.Fatalf("derivation: %v", dErr)
			}
			original, err = provenance.Derived(source, collectedAt, interpreter, record.Ref(), confidence)
			if err != nil {
				t.Fatalf("derived: %v", err)
			}
		}

		wire := kernelwire.EncodeProvenance(original)
		back, err := kernelwire.DecodeProvenance(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if back.Source().String() != original.Source().String() {
			t.Errorf("source lost: %s -> %s", original.Source(), back.Source())
		}
		if !back.CollectedAt().Equal(original.CollectedAt()) {
			t.Errorf("collected_at lost: %s -> %s", original.CollectedAt(), back.CollectedAt())
		}
		if !back.PublishedAt().Equal(original.PublishedAt()) {
			t.Errorf("published_at lost: %s -> %s", original.PublishedAt(), back.PublishedAt())
		}
		if back.Interpreter().String() != original.Interpreter().String() {
			t.Errorf("interpreter lost: %s -> %s", original.Interpreter(), back.Interpreter())
		}
		if back.Confidence() != original.Confidence() {
			t.Errorf("confidence lost: %s -> %s", original.Confidence(), back.Confidence())
		}

		originalRef, originalDerived := original.Derivation()
		backRef, backDerived := back.Derivation()
		if originalDerived != backDerived || originalRef.Hash() != backRef.Hash() {
			t.Errorf("derivation lost: %v/%s -> %v/%s",
				originalDerived, originalRef, backDerived, backRef)
		}

		if !proto.Equal(wire, kernelwire.EncodeProvenance(back)) {
			t.Fatal("wire round trip is not the identity: a field was dropped decoding")
		}
	})
}

func TestReferenceBindingRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original, err := provenance.NewReferenceBinding(
			rapid.StringMatching(`[a-z][a-z-]{2,16}`).Draw(t, "dataset"),
			rapid.StringMatching(`[0-9a-f]{8}`).Draw(t, "version"),
		)
		if err != nil {
			t.Fatalf("binding: %v", err)
		}
		wire := kernelwire.EncodeReferenceBinding(original)
		back, err := kernelwire.DecodeReferenceBinding(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		if back.String() != original.String() {
			t.Fatalf("lost information: %s -> %s", original, back)
		}
		if !proto.Equal(wire, kernelwire.EncodeReferenceBinding(back)) {
			t.Fatal("wire round trip is not the identity")
		}
	})
}

// Decoding is validation, not assignment. A wire message is data from outside
// the process, and the domain constructors are the only thing between it and an
// invalid value.
func TestDecodingRejectsInvalidWireValues(t *testing.T) {
	if _, err := kernelwire.DecodeMoney(nil); err == nil {
		t.Error("nil money decoded without error")
	}
	if _, err := kernelwire.DecodeProvenance(nil); err == nil {
		t.Error("nil provenance decoded without error")
	}
	if _, err := kernelwire.DecodeCoordinates(nil); err == nil {
		t.Error("nil coordinates decoded without error")
	}
	if _, err := kernelwire.DecodeEntityID(nil); err == nil {
		t.Error("nil entity id decoded without error")
	}
	if _, err := kernelwire.DecodeClaim(nil); err == nil {
		t.Error("nil claim decoded without error")
	}
}

// --- generators --------------------------------------------------------------

func genDecimalText() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		out := rapid.StringMatching(`[0-9]{1,10}`).Draw(t, "whole")
		if scale := rapid.IntRange(0, 6).Draw(t, "scale"); scale > 0 {
			frac := rapid.StringMatching(`[0-9]{1,6}`).Draw(t, "frac")
			if len(frac) > scale {
				frac = frac[:scale]
			}
			out += "." + frac
		}
		if rapid.Bool().Draw(t, "negative") {
			out = "-" + out
		}
		return out
	})
}

func genCurrency() *rapid.Generator[string] {
	return rapid.SampledFrom([]string{"BRL", "USD", "EUR", "JPY"})
}

func genInstant() *rapid.Generator[temporal.Instant] {
	return rapid.Custom(func(t *rapid.T) temporal.Instant {
		// Bounded to a realistic range: the property is about the codec, and
		// unbounded timestamps would test protobuf's range handling instead.
		secs := rapid.Int64Range(1_000_000_000, 2_000_000_000).Draw(t, "unix")
		nanos := rapid.Int64Range(0, 999_999_999).Draw(t, "nanos")
		return temporal.MustAt(time.Unix(secs, nanos).UTC())
	})
}

func genInterval() *rapid.Generator[temporal.Interval] {
	return rapid.Custom(func(t *rapid.T) temporal.Interval {
		from := genInstant().Draw(t, "from")
		if rapid.Bool().Draw(t, "open") {
			iv, err := temporal.OpenFrom(from)
			if err != nil {
				panic(err)
			}
			return iv
		}
		extra := rapid.Int64Range(0, 86_400).Draw(t, "duration")
		to := temporal.MustAt(from.Time().Add(time.Duration(extra) * time.Second))
		iv, err := temporal.NewInterval(from, to)
		if err != nil {
			panic(err)
		}
		return iv
	})
}
