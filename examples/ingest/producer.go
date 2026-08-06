// Package main is a worked producer: it builds a claim submission using only
// the published contract module, and checks it against the rules admission
// applies.
//
// # What a producer imports, and what it does not
//
// The producer half below imports `libs/contracts` and nothing else. That is
// deliberate and it is the whole point: ADR-0025 makes `libs/contracts` the only
// module FDOS offers, and a producer that imports `libs/kernel` or `libs/ledger`
// is depending on FDOS internals that carry no compatibility promise.
//
// The conformance half imports plenty. It is FDOS code checking FDOS rules, it
// lives inside this repository, and **a producer does not link it** — a producer
// runs it, or compares against the fixtures in testdata/.
//
// If you are writing a producer in another language, the shape below and the
// serialized fixtures are the specification. There is nothing Go-specific about
// a submission.
package main

import (
	"fmt"
	"os"
	"time"

	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	ingestv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ingest/v1"
	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
)

// build assembles a conforming submission using only published contract types.
//
// Every value here is obviously fictitious. `examples/` forbids real financial
// data, account identifiers or institution names, and a fixture that looked
// real would be a disclosure rather than a demonstration.
func build() *ingestv1.HoldingClaimSubmission {
	collected := time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC)

	return &ingestv1.HoldingClaimSubmission{
		// Where this belongs. A name, not an identity — naming a stream is not
		// asserting who owns it, and nothing yet says who may write to one.
		Stream: "example-account",

		// The identifiers exactly as the producer read them. Never an EntityId:
		// a producer cannot mint one and must not derive one, because deriving
		// identity from an external identifier makes that identifier the
		// primary key (ADR-0007).
		//
		// Render a given value identically forever. Emit "EXMPL4" today and
		// "EXMPL4 " tomorrow and you mint two entities for one instrument,
		// silently, and nothing rejects it because both are well-formed.
		Account:    &kernelv1.IdentifierClaim{Scheme: "account", Value: "EXAMPLE-0001"},
		Instrument: &kernelv1.IdentifierClaim{Scheme: "ticker", Value: "EXMPL4"},

		Quantity: &kernelv1.Quantity{
			Amount: &kernelv1.Decimal{Value: "100"},
			Unit:   "share",
		},

		// When the holding was true, as the source stated it. Open-ended here:
		// a position statement claims the holding is still true. There is no
		// default — a defaulted effective time fabricates a temporal claim.
		Effective: &kernelv1.EffectiveInterval{
			From: timestamppb.New(collected),
		},

		// The content address of the acquisition this came from. FDOS never
		// dereferences it, but the form is specified: sha256: plus 64 lowercase
		// hexadecimal characters (ADR-0028).
		Source: &kernelv1.SourceRef{
			Value: "sha256:1111111111111111111111111111111111111111111111111111111111111111",
		},

		// When the producer acquired it. Not when FDOS did — FDOS acquires
		// nothing.
		CollectedAt: timestamppb.New(collected),

		// No code read this value; a person exported a statement and typed it.
		// The reserved name asserts that, and the version is the version of the
		// convention rather than of code that does not exist.
		//
		// There is no value meaning "I do not know". A producer that ran a
		// parser names it and its version here instead.
		Interpreter: &kernelv1.InterpreterRef{Name: "unmediated", Version: "1"},

		Confidence: kernelv1.Confidence_CONFIDENCE_ASSERTED,
	}
}

func main() {
	submission := build()

	wire, err := proto.Marshal(submission)
	if err != nil {
		fmt.Fprintf(os.Stderr, "marshal: %v\n", err)
		os.Exit(1)
	}

	if err := Check(wire); err != nil {
		fmt.Fprintf(os.Stderr, "this submission would be refused: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("conforming submission, %d bytes\n", len(wire))
	fmt.Printf("  stream      %s\n", submission.GetStream())
	fmt.Printf("  account     %s:%s\n", submission.GetAccount().GetScheme(), submission.GetAccount().GetValue())
	fmt.Printf("  instrument  %s:%s\n", submission.GetInstrument().GetScheme(), submission.GetInstrument().GetValue())
	fmt.Printf("  source      %s\n", submission.GetSource().GetValue())
}
