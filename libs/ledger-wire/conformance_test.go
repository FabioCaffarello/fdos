package ledgerwire_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"pgregory.net/rapid"

	ledgerv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ledger/v1"
	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	ledgerwire "github.com/FabioCaffarello/fdos/libs/ledger-wire"
	"github.com/FabioCaffarello/fdos/libs/ledger/domain"
)

// Closes B-003 for the ledger types. Same two properties as libs/kernel-wire:
//
//	domain -> wire -> domain   is the identity   (nothing lost encoding)
//	wire   -> domain -> wire   is the identity   (nothing dropped decoding)
//
// A fact is where the two matter most. If any part of the envelope fails to
// survive, a fact arrives unable to say when it was true, when FDOS learned it,
// or where it came from — and §6 and §7 become claims about the domain only,
// true right up until something is written down.

func TestFactRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := genFact(t)

		wire, err := ledgerwire.EncodeFact(original)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		back, err := ledgerwire.DecodeFact(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		assertFactsEqual(t, original, back)

		reencoded, err := ledgerwire.EncodeFact(back)
		if err != nil {
			t.Fatalf("re-encode: %v", err)
		}
		if !proto.Equal(wire, reencoded) {
			t.Fatal("wire round trip is not the identity: a field was dropped decoding")
		}
	})
}

// A correction is a fact too. It travels the same path, and losing the
// reference it corrects would leave a retraction that retracts nothing.
func TestCorrectionRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		envelope, ref := genEnvelope(t)
		payload := domain.FactCorrected{
			Corrects: domain.Ref{
				Stream:   "acct-1",
				Sequence: rapid.Uint64Range(1, 500).Draw(t, "corrects"),
			},
			Kind: rapid.SampledFrom([]domain.CorrectionKind{
				domain.CorrectionKindCorrected,
				domain.CorrectionKindRetracted,
				domain.CorrectionKindSuperseded,
			}).Draw(t, "kind"),
			Reason: rapid.StringMatching(`[a-z ]{3,40}`).Draw(t, "reason"),
		}

		original, err := domain.NewFact(ref, envelope, domain.KindObservation, payload)
		if err != nil {
			t.Fatalf("new fact: %v", err)
		}

		wire, err := ledgerwire.EncodeFact(original)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		back, err := ledgerwire.DecodeFact(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		got, ok := back.Payload().(domain.FactCorrected)
		if !ok {
			t.Fatalf("payload decoded as %T", back.Payload())
		}
		if got != payload {
			t.Fatalf("correction lost information: %+v -> %+v", payload, got)
		}

		reencoded, _ := ledgerwire.EncodeFact(back)
		if !proto.Equal(wire, reencoded) {
			t.Fatal("wire round trip is not the identity")
		}
	})
}

// A claimed holding is what a connector actually emits, so this is the codec
// path every acquisition takes (ADR-0022). The value must arrive byte-identical:
// a claim altered in transit resolves to a different entity, or to none.
func TestHoldingClaimedRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		envelope, ref := genEnvelope(t)
		payload := domain.HoldingClaimed{
			Account:    genClaim(t, "account_number", "account"),
			Instrument: genClaim(t, "ticker", "instrument"),
			Quantity: money.MustParseQuantity(
				rapid.StringMatching(`[0-9]{1,6}`).Draw(t, "quantity"),
				rapid.SampledFrom([]string{"share", "bond.face"}).Draw(t, "unit"),
			),
		}

		original, err := domain.NewFact(ref, envelope, domain.KindObservation, payload)
		if err != nil {
			t.Fatalf("new fact: %v", err)
		}

		wire, err := ledgerwire.EncodeFact(original)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		back, err := ledgerwire.DecodeFact(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		assertFactsEqual(t, original, back)

		got, ok := back.Payload().(domain.HoldingClaimed)
		if !ok {
			t.Fatalf("payload decoded as %T", back.Payload())
		}
		if got.Account.Value() != payload.Account.Value() {
			t.Fatalf("account claim altered: %q -> %q", payload.Account.Value(), got.Account.Value())
		}
		if got.Instrument.Value() != payload.Instrument.Value() {
			t.Fatalf("instrument claim altered: %q -> %q", payload.Instrument.Value(), got.Instrument.Value())
		}
		if got.Quantity.String() != payload.Quantity.String() {
			t.Fatalf("quantity lost: %s -> %s", payload.Quantity, got.Quantity)
		}

		reencoded, err := ledgerwire.EncodeFact(back)
		if err != nil {
			t.Fatalf("re-encode: %v", err)
		}
		if !proto.Equal(wire, reencoded) {
			t.Fatal("wire round trip is not the identity: a field was dropped decoding")
		}
	})
}

// A mint is the birth certificate of an identity. Losing the claim it was born
// from would leave an entity nobody can audit — the failure ADR-0022 exists to
// prevent.
func TestEntityMintedRoundTrips(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		envelope, ref := genEnvelope(t)
		claim := genClaim(t, "ticker", "born_from")
		minted, err := domain.MintFor(identity.KindInstrument, claim)
		if err != nil {
			t.Fatalf("mint: %v", err)
		}

		original, err := domain.NewFact(ref, envelope, domain.KindObservation, minted)
		if err != nil {
			t.Fatalf("new fact: %v", err)
		}

		wire, err := ledgerwire.EncodeFact(original)
		if err != nil {
			t.Fatalf("encode: %v", err)
		}
		back, err := ledgerwire.DecodeFact(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		assertFactsEqual(t, original, back)

		got, ok := back.Payload().(domain.EntityMinted)
		if !ok {
			t.Fatalf("payload decoded as %T", back.Payload())
		}
		if !got.Entity.Equal(minted.Entity) {
			t.Fatalf("entity lost: %s -> %s", minted.Entity, got.Entity)
		}
		if !got.BornFrom.Equal(minted.BornFrom) {
			t.Fatalf("the claim the identity was born from was lost: %s -> %s",
				minted.BornFrom, got.BornFrom)
		}

		reencoded, err := ledgerwire.EncodeFact(back)
		if err != nil {
			t.Fatalf("re-encode: %v", err)
		}
		if !proto.Equal(wire, reencoded) {
			t.Fatal("wire round trip is not the identity: a field was dropped decoding")
		}
	})
}

// The declared type is checked against the payload rather than trusted. A fact
// claiming to be one thing while carrying another must not reach a projection —
// it is the shape a corrupted or hostile stream would take.
func TestDecodeRejectsATypeThatContradictsItsPayload(t *testing.T) {
	fact := buildFact(t, money.MustParseQuantity("10", "share"))

	wire, err := ledgerwire.EncodeFact(fact)
	if err != nil {
		t.Fatalf("encode: %v", err)
	}
	wire.Type = "ledger.TradeSettled"

	if _, err := ledgerwire.DecodeFact(wire); err == nil {
		t.Fatal("a fact declaring the wrong type decoded without error")
	}
}

func TestDecodeRejectsAVersionThatContradictsItsPayload(t *testing.T) {
	fact := buildFact(t, money.MustParseQuantity("10", "share"))
	wire, _ := ledgerwire.EncodeFact(fact)
	wire.TypeVersion = 99

	if _, err := ledgerwire.DecodeFact(wire); err == nil {
		t.Fatal("a fact declaring the wrong type version decoded without error")
	}
}

// Enum mappings have no default that silently succeeds. An unspecified kind
// decoding to Observation would silently turn a corrupt fact into a plausible
// one.
func TestDecodeRejectsUnspecifiedEnums(t *testing.T) {
	fact := buildFact(t, money.MustParseQuantity("1", "share"))
	wire, _ := ledgerwire.EncodeFact(fact)
	wire.Kind = ledgerv1.FactKind_FACT_KIND_UNSPECIFIED

	if _, err := ledgerwire.DecodeFact(wire); err == nil {
		t.Fatal("an unspecified fact kind decoded without error")
	}
}

// Decoding is validation, not assignment. An envelope missing its provenance is
// exactly what §6 exists to make impossible, and the domain constructor is what
// enforces it — so the codec must go through it rather than around.
func TestDecodeRejectsAnEnvelopeWithoutProvenance(t *testing.T) {
	fact := buildFact(t, money.MustParseQuantity("1", "share"))
	wire, _ := ledgerwire.EncodeFact(fact)
	wire.Envelope.Provenance = nil

	if _, err := ledgerwire.DecodeFact(wire); err == nil {
		t.Fatal("an envelope without provenance decoded without error")
	}
}

func TestDecodeRejectsNil(t *testing.T) {
	if _, err := ledgerwire.DecodeFact(nil); err == nil {
		t.Error("nil fact decoded without error")
	}
	if _, _, err := ledgerwire.DecodeEnvelope(nil); err == nil {
		t.Error("nil envelope decoded without error")
	}
}

// --- helpers -----------------------------------------------------------------

func assertFactsEqual(t *rapid.T, original, back domain.Fact) {
	t.Helper()

	if back.Ref() != original.Ref() {
		t.Fatalf("ref lost: %s -> %s", original.Ref(), back.Ref())
	}
	if back.Kind() != original.Kind() {
		t.Fatalf("kind lost: %s -> %s", original.Kind(), back.Kind())
	}
	if back.Type() != original.Type() || back.TypeVersion() != original.TypeVersion() {
		t.Fatalf("type lost: %s v%d -> %s v%d",
			original.Type(), original.TypeVersion(), back.Type(), back.TypeVersion())
	}

	oc, bc := original.Envelope().Coordinates(), back.Envelope().Coordinates()
	if !bc.Knowledge().Equal(oc.Knowledge()) {
		t.Fatalf("knowledge time lost: %s -> %s", oc.Knowledge(), bc.Knowledge())
	}
	if !bc.Effective().From().Equal(oc.Effective().From()) ||
		bc.Effective().IsOpen() != oc.Effective().IsOpen() {
		t.Fatalf("effective interval lost: %s -> %s", oc.Effective(), bc.Effective())
	}

	op, bp := original.Envelope().Provenance(), back.Envelope().Provenance()
	if bp.Source().String() != op.Source().String() ||
		bp.Confidence() != op.Confidence() ||
		bp.Interpreter().String() != op.Interpreter().String() {
		t.Fatalf("provenance lost: %+v -> %+v", op, bp)
	}

	if len(back.Envelope().References()) != len(original.Envelope().References()) {
		t.Fatalf("reference bindings lost: %d -> %d",
			len(original.Envelope().References()), len(back.Envelope().References()))
	}

	ohold, ok := original.Payload().(domain.HoldingObserved)
	if !ok {
		return
	}
	bhold, ok := back.Payload().(domain.HoldingObserved)
	if !ok {
		t.Fatalf("payload decoded as %T", back.Payload())
	}
	if !bhold.Account.Equal(ohold.Account) || !bhold.Instrument.Equal(ohold.Instrument) {
		t.Fatal("entity identifiers lost")
	}
	if bhold.Quantity.String() != ohold.Quantity.String() ||
		bhold.Quantity.Unit() != ohold.Quantity.Unit() {
		t.Fatalf("quantity lost: %s %s -> %s %s",
			ohold.Quantity, ohold.Quantity.Unit(), bhold.Quantity, bhold.Quantity.Unit())
	}
}

func genFact(t *rapid.T) domain.Fact {
	t.Helper()

	envelope, ref := genEnvelope(t)
	quantity := money.MustParseQuantity(
		rapid.StringMatching(`[0-9]{1,6}`).Draw(t, "quantity"),
		rapid.SampledFrom([]string{"share", "bond.face"}).Draw(t, "unit"),
	)

	fact, err := domain.NewFact(ref, envelope, domain.KindObservation, domain.HoldingObserved{
		Account:    identity.MustDerive(identity.KindAccount, rapid.StringMatching(`[a-z]{3,8}`).Draw(t, "account")),
		Instrument: identity.MustDerive(identity.KindInstrument, rapid.StringMatching(`[A-Z]{4,12}`).Draw(t, "isin")),
		Quantity:   quantity,
	})
	if err != nil {
		t.Fatalf("new fact: %v", err)
	}
	return fact
}

func genEnvelope(t *rapid.T) (domain.Envelope, domain.Ref) {
	t.Helper()

	from := temporal.MustAt(time.Unix(rapid.Int64Range(1_000_000_000, 1_900_000_000).Draw(t, "from"), 0).UTC())
	var effective temporal.Interval
	var err error
	if rapid.Bool().Draw(t, "open") {
		effective, err = temporal.OpenFrom(from)
	} else {
		effective, err = temporal.NewInterval(from,
			temporal.MustAt(from.Time().Add(time.Duration(rapid.Int64Range(0, 86_400).Draw(t, "dur"))*time.Second)))
	}
	if err != nil {
		t.Fatalf("interval: %v", err)
	}

	knowledge := temporal.MustAt(from.Time().Add(time.Hour))
	coordinates, err := temporal.Assign(effective, knowledge)
	if err != nil {
		t.Fatalf("assign: %v", err)
	}

	source, _ := provenance.NewSource("broker.statement")
	interpreter, _ := provenance.NewInterpreter("statement-parser", "1.0.0")
	prov, err := provenance.Observed(source, from, interpreter,
		rapid.SampledFrom([]provenance.Confidence{
			provenance.ConfidenceAsserted, provenance.ConfidenceDerived,
			provenance.ConfidenceEstimated, provenance.ConfidenceInferred,
			provenance.ConfidenceDisputed,
		}).Draw(t, "confidence"))
	if err != nil {
		t.Fatalf("provenance: %v", err)
	}

	var bindings []provenance.ReferenceBinding
	if rapid.Bool().Draw(t, "hasReferences") {
		binding, bErr := provenance.NewReferenceBinding("ecb-fx-daily", "2026-01-02")
		if bErr != nil {
			t.Fatalf("binding: %v", bErr)
		}
		bindings = append(bindings, binding)
	}

	envelope, err := domain.NewEnvelope(coordinates, prov, bindings)
	if err != nil {
		t.Fatalf("envelope: %v", err)
	}
	return envelope, domain.Ref{
		Stream:   "acct-1",
		Sequence: rapid.Uint64Range(1, 1000).Draw(t, "sequence"),
	}
}

// buildFact is the non-property fixture, for tests that mutate the wire form.
func buildFact(t *testing.T, quantity money.Quantity) domain.Fact {
	t.Helper()

	at := temporal.MustAt(time.Unix(1_700_000_000, 0).UTC())
	effective, err := temporal.OpenFrom(at)
	if err != nil {
		t.Fatal(err)
	}
	coordinates, err := temporal.Assign(effective, temporal.MustAt(at.Time().Add(time.Hour)))
	if err != nil {
		t.Fatal(err)
	}
	source, _ := provenance.NewSource("broker.statement")
	interpreter, _ := provenance.NewInterpreter("statement-parser", "1.0.0")
	prov, err := provenance.Observed(source, at, interpreter, provenance.ConfidenceAsserted)
	if err != nil {
		t.Fatal(err)
	}
	envelope, err := domain.NewEnvelope(coordinates, prov, nil)
	if err != nil {
		t.Fatal(err)
	}
	fact, err := domain.NewFact(
		domain.Ref{Stream: "acct-1", Sequence: 1},
		envelope, domain.KindObservation,
		domain.HoldingObserved{
			Account:    identity.MustDerive(identity.KindAccount, "acct-1"),
			Instrument: identity.MustDerive(identity.KindInstrument, "BRPETRACNOR9"),
			Quantity:   quantity,
		})
	if err != nil {
		t.Fatal(err)
	}
	return fact
}

// genClaim generates a claim whose value deliberately includes padding and
// mixed case. RFC-0007 forbids the boundary normalising either, so the codec
// must carry them through untouched.
func genClaim(t *rapid.T, scheme, label string) identity.Claim {
	t.Helper()
	value := rapid.StringMatching(`[ ]{0,1}[A-Za-z0-9.-]{1,12}[ ]{0,1}`).Draw(t, label)
	c, err := identity.NewClaim(scheme, value)
	if err != nil {
		t.Fatalf("claim %q: %v", value, err)
	}
	return c
}
