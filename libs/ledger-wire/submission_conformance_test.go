package ledgerwire_test

import (
	"testing"
	"time"

	"google.golang.org/protobuf/proto"
	"pgregory.net/rapid"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
	"github.com/FabioCaffarello/fdos/libs/kernel/money"
	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
	"github.com/FabioCaffarello/fdos/libs/kernel/temporal"
	ledgerwire "github.com/FabioCaffarello/fdos/libs/ledger-wire"
	"github.com/FabioCaffarello/fdos/libs/ledger/app"
)

// submissions generates admission commands across the whole shape.
func submissions(t *rapid.T) app.AcceptHoldingClaimCommand {
	at, err := temporal.At(time.Unix(rapid.Int64Range(1_000_000, 2_000_000_000).Draw(t, "at"), 0).UTC())
	if err != nil {
		t.Fatalf("instant: %v", err)
	}
	effective, err := temporal.OpenFrom(at)
	if err != nil {
		t.Fatalf("interval: %v", err)
	}
	quantity, err := money.ParseQuantity(
		rapid.StringMatching(`[1-9][0-9]{0,4}`).Draw(t, "qty"),
		rapid.SampledFrom([]string{"share", "unit", "bond"}).Draw(t, "unit"),
	)
	if err != nil {
		t.Fatalf("quantity: %v", err)
	}
	source, err := provenance.NewSource(rapid.StringMatching(`sha256:[0-9a-f]{64}`).Draw(t, "source"))
	if err != nil {
		t.Fatalf("source: %v", err)
	}

	// Both interpreter shapes: a versioned parser, and the reserved sentinel a
	// producer uses when no code read the value (ADR-0028).
	interpreter := provenance.Unmediated()
	if rapid.Bool().Draw(t, "programmatic") {
		interpreter, err = provenance.NewInterpreter(
			rapid.StringMatching(`[a-z][a-z.]{2,12}`).Draw(t, "interp"),
			rapid.StringMatching(`[0-9]\.[0-9]\.[0-9]`).Draw(t, "version"),
		)
		if err != nil {
			t.Fatalf("interpreter: %v", err)
		}
	}

	cmd := app.AcceptHoldingClaimCommand{
		Stream:      rapid.StringMatching(`[a-z][a-z0-9-]{2,20}`).Draw(t, "stream"),
		Account:     identity.MustClaim("account", rapid.StringMatching(`[A-Z0-9-]{3,10}`).Draw(t, "account")),
		Instrument:  identity.MustClaim("ticker", rapid.StringMatching(`[A-Z0-9]{4,6}`).Draw(t, "instrument")),
		Quantity:    quantity,
		Effective:   effective,
		Source:      source,
		CollectedAt: at,
		Interpreter: interpreter,
		Confidence: rapid.SampledFrom([]provenance.Confidence{
			provenance.ConfidenceAsserted,
			provenance.ConfidenceDerived,
			provenance.ConfidenceEstimated,
		}).Draw(t, "confidence"),
	}

	for range rapid.IntRange(0, 3).Draw(t, "refs") {
		binding, bErr := provenance.NewReferenceBinding(
			rapid.StringMatching(`[a-z][a-z.]{2,12}`).Draw(t, "dataset"),
			rapid.StringMatching(`v[0-9]\.[0-9]`).Draw(t, "refversion"),
		)
		if bErr != nil {
			t.Fatalf("binding: %v", bErr)
		}
		cmd.References = append(cmd.References, binding)
	}
	return cmd
}

// command -> wire -> command is the identity. Nothing is lost encoding.
func TestSubmissionRoundTripsFromTheCommand(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		original := submissions(t)

		decoded, err := ledgerwire.DecodeHoldingClaimSubmission(
			ledgerwire.EncodeHoldingClaimSubmission(original),
		)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}

		if decoded.Stream != original.Stream {
			t.Fatalf("stream: %q -> %q", original.Stream, decoded.Stream)
		}
		if !decoded.Account.Equal(original.Account) || !decoded.Instrument.Equal(original.Instrument) {
			t.Fatalf("claims changed: %s/%s -> %s/%s",
				original.Account, original.Instrument, decoded.Account, decoded.Instrument)
		}
		if decoded.Quantity.String() != original.Quantity.String() {
			t.Fatalf("quantity: %s -> %s", original.Quantity, decoded.Quantity)
		}
		if decoded.Source.String() != original.Source.String() {
			t.Fatalf("source: %s -> %s", original.Source, decoded.Source)
		}
		if decoded.Interpreter.String() != original.Interpreter.String() {
			t.Fatalf("interpreter: %s -> %s", original.Interpreter, decoded.Interpreter)
		}
		if decoded.Confidence != original.Confidence {
			t.Fatalf("confidence: %s -> %s", original.Confidence, decoded.Confidence)
		}
		if len(decoded.References) != len(original.References) {
			t.Fatalf("references: %d -> %d", len(original.References), len(decoded.References))
		}
	})
}

// wire -> command -> wire is the identity. Nothing is dropped decoding.
//
// This is the direction that earns its keep. A codec that never reads a field
// passes the first property forever, because the value it fails to carry was
// never in the command it compares against.
func TestSubmissionRoundTripsFromTheWire(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		wire := ledgerwire.EncodeHoldingClaimSubmission(submissions(t))

		cmd, err := ledgerwire.DecodeHoldingClaimSubmission(wire)
		if err != nil {
			t.Fatalf("decode: %v", err)
		}
		again := ledgerwire.EncodeHoldingClaimSubmission(cmd)

		if !proto.Equal(wire, again) {
			t.Fatalf("wire changed across a decode/encode cycle:\n  %v\n  %v", wire, again)
		}
	})
}

// The submission carries no knowledge time and no derivation, and there is no
// code path by which either could arrive (ADR-0030). This asserts the absence
// stays an absence: a field added later would make both claimable again.
func TestSubmissionCannotCarryKnowledgeTimeOrDerivation(t *testing.T) {
	wire := ledgerwire.EncodeHoldingClaimSubmission(app.AcceptHoldingClaimCommand{})

	fields := wire.ProtoReflect().Descriptor().Fields()
	for i := range fields.Len() {
		switch name := string(fields.Get(i).Name()); name {
		case "knowledge", "knowledge_time", "derivation":
			t.Fatalf("submission gained a %q field — a producer must be able to claim neither", name)
		}
	}
}
