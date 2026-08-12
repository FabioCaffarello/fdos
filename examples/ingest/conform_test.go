package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"

	ingestv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/ingest/v1"
	kernelv1 "github.com/FabioCaffarello/fdos/libs/contracts/gen/fdos/kernel/v1"
)

// The conforming case, which is also the fixture a producer in another language
// compares against.
func TestTheWorkedProducerConforms(t *testing.T) {
	wire, err := proto.Marshal(build())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := Check(wire); err != nil {
		t.Fatalf("the worked example does not conform: %v", err)
	}
}

// Every way a submission is refused, with the reason a producer author needs.
//
// These are the specification. If one of them disagrees with what admission
// does, admission is right and this table is the bug — Check runs admission
// rather than describing it, so the two cannot drift, and a failure here means
// the expectation was wrong rather than the rule.
func TestEveryRefusal(t *testing.T) {
	for _, tc := range []struct {
		name    string
		corrupt func(*ingestv1.HoldingClaimSubmission)
		want    string
	}{
		{
			name:    "a source that is not a content address",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Source.Value = "https://example.invalid/statement.pdf" },
			want:    "malformed source",
		},
		{
			name:    "a source digest of the wrong length",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Source.Value = "sha256:abc123" },
			want:    "malformed source",
		},
		{
			name:    "an uppercase digest",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Source.Value = "sha256:" + strings.Repeat("A", 64) },
			want:    "malformed source",
		},
		{
			name:    "no source at all",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Source = nil },
			want:    "source",
		},
		{
			name:    "no instrument",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Instrument = nil },
			want:    "instrument",
		},
		{
			name:    "an identifier scheme that is not canonical",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Instrument.Scheme = "Ticker" },
			want:    "canonical",
		},
		{
			name:    "an interpreter with no version",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Interpreter.Version = "" },
			want:    "interpreter",
		},
		{
			name:    "no interpreter at all",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Interpreter = nil },
			want:    "interpreter",
		},
		{
			name:    "no effective interval",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Effective = nil },
			want:    "effective",
		},
		{
			name:    "an unspecified confidence",
			corrupt: func(s *ingestv1.HoldingClaimSubmission) { s.Confidence = kernelv1.Confidence_CONFIDENCE_UNSPECIFIED },
			want:    "confidence",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			submission := build()
			tc.corrupt(submission)

			wire, err := proto.Marshal(submission)
			if err != nil {
				t.Fatalf("marshal: %v", err)
			}

			err = Check(wire)
			if err == nil {
				t.Fatalf("want a refusal, got none")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("want an error mentioning %q, got %v", tc.want, err)
			}
		})
	}
}

// The fixtures are committed so a producer in another language has something to
// compare bytes against. This keeps them honest: if the shape changes, the
// committed fixture stops matching what the worked producer emits.
//
// **The wire fixture is byte-compared; the textproto one is not, and cannot be.**
// `prototext` output is destabilised on purpose —
// `google.golang.org/protobuf/internal/detrand` is "seeded by the program binary
// itself" and "ensur[es] that the output is unstable across different builds" —
// so a byte-compared textproto fixture passes only for the build that produced
// it. It disagreed between the workspace and `GOWORK=off`, and a dependency bump
// moved which of the two won. Regenerating it would only move that again.
//
// So the textproto is validated by **parsing it and comparing the message**,
// which is the property it actually needs to carry: a producer in another
// language reads it to learn the shape, and a reader that round-trips to an equal
// message has learned the right shape whatever the whitespace. The wire fixture
// keeps the byte comparison, because `proto.Marshal` of a fixed message is stable
// for a fixed binary and it is what a foreign producer diffs.
//
// This is the same class of defect as the derivation pre-image RFC-0016 named:
// protobuf serialization is documented as not canonical, so nothing may assume
// its bytes are.
func TestFixturesMatchTheWorkedProducer(t *testing.T) {
	submission := build()

	wire, err := proto.Marshal(submission)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	wirePath := filepath.Join("testdata", "conforming.bin")
	got, err := os.ReadFile(wirePath)
	if err != nil {
		if os.IsNotExist(err) && os.Getenv("UPDATE_FIXTURES") != "" {
			if wErr := os.WriteFile(wirePath, wire, 0o644); wErr != nil {
				t.Fatalf("write %s: %v", wirePath, wErr)
			}
		} else {
			t.Fatalf("read %s: %v (run with UPDATE_FIXTURES=1 to create)", wirePath, err)
		}
	} else if string(got) != string(wire) {
		t.Errorf("%s is stale — the submission shape changed. Re-create with UPDATE_FIXTURES=1 "+
			"and review the diff.", wirePath)
	}

	textPath := filepath.Join("testdata", "conforming.textproto")
	text, err := os.ReadFile(textPath)
	if err != nil {
		if os.IsNotExist(err) && os.Getenv("UPDATE_FIXTURES") != "" {
			rendered, mErr := prototext.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(submission)
			if mErr != nil {
				t.Fatalf("prototext: %v", mErr)
			}
			if wErr := os.WriteFile(textPath, rendered, 0o644); wErr != nil {
				t.Fatalf("write %s: %v", textPath, wErr)
			}
			return
		}
		t.Fatalf("read %s: %v (run with UPDATE_FIXTURES=1 to create)", textPath, err)
	}

	var parsed ingestv1.HoldingClaimSubmission
	if uErr := prototext.Unmarshal(text, &parsed); uErr != nil {
		t.Fatalf("%s does not parse as a HoldingClaimSubmission: %v", textPath, uErr)
	}
	if !proto.Equal(&parsed, submission) {
		t.Errorf("%s describes a different submission than the worked producer emits. "+
			"Re-create with UPDATE_FIXTURES=1 and review the diff.", textPath)
	}
}
