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
		name   string
		break_ func(*ingestv1.HoldingClaimSubmission)
		want   string
	}{
		{
			name:   "a source that is not a content address",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Source.Value = "https://example.invalid/statement.pdf" },
			want:   "malformed source",
		},
		{
			name:   "a source digest of the wrong length",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Source.Value = "sha256:abc123" },
			want:   "malformed source",
		},
		{
			name:   "an uppercase digest",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Source.Value = "sha256:" + strings.Repeat("A", 64) },
			want:   "malformed source",
		},
		{
			name:   "no source at all",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Source = nil },
			want:   "source",
		},
		{
			name:   "no instrument",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Instrument = nil },
			want:   "instrument",
		},
		{
			name:   "an identifier scheme that is not canonical",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Instrument.Scheme = "Ticker" },
			want:   "canonical",
		},
		{
			name:   "an interpreter with no version",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Interpreter.Version = "" },
			want:   "interpreter",
		},
		{
			name:   "no interpreter at all",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Interpreter = nil },
			want:   "interpreter",
		},
		{
			name:   "no effective interval",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Effective = nil },
			want:   "effective",
		},
		{
			name:   "an unspecified confidence",
			break_: func(s *ingestv1.HoldingClaimSubmission) { s.Confidence = kernelv1.Confidence_CONFIDENCE_UNSPECIFIED },
			want:   "confidence",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			submission := build()
			tc.break_(submission)

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
func TestFixturesMatchTheWorkedProducer(t *testing.T) {
	submission := build()

	wire, err := proto.Marshal(submission)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	text, err := prototext.MarshalOptions{Multiline: true, Indent: "  "}.Marshal(submission)
	if err != nil {
		t.Fatalf("prototext: %v", err)
	}

	for name, want := range map[string][]byte{
		"conforming.bin":       wire,
		"conforming.textproto": text,
	} {
		path := filepath.Join("testdata", name)
		got, rErr := os.ReadFile(path)
		if rErr != nil {
			if os.IsNotExist(rErr) && os.Getenv("UPDATE_FIXTURES") != "" {
				if wErr := os.WriteFile(path, want, 0o644); wErr != nil {
					t.Fatalf("write %s: %v", path, wErr)
				}
				continue
			}
			t.Fatalf("read %s: %v (run with UPDATE_FIXTURES=1 to create)", path, rErr)
		}
		if string(got) != string(want) {
			t.Fatalf("%s is stale — the submission shape changed. Re-create with UPDATE_FIXTURES=1 and review the diff.", path)
		}
	}
}
