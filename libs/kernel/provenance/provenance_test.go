package provenance_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/FabioCaffarello/fdos/libs/kernel/provenance"
)

// A Ref rebuilt from a record's own address must equal it, or a decoded fact
// cannot point at the derivation it came with.
func TestNewRefRoundTripsARecordAddress(t *testing.T) {
	method, err := provenance.NewMethod("test.Method", "1")
	if err != nil {
		t.Fatal(err)
	}
	record, err := provenance.NewDerivation(method, []string{"a#1"}, nil, nil, provenance.ConfidenceDerived)
	if err != nil {
		t.Fatal(err)
	}

	rebuilt, err := provenance.NewRef(record.Ref().Hash())
	if err != nil {
		t.Fatalf("NewRef on a real address: %v", err)
	}
	if rebuilt.Hash() != record.Ref().Hash() {
		t.Fatalf("round trip changed the address: %s -> %s", record.Ref(), rebuilt)
	}
}

// Malformed addresses are rejected. Validation is shape only: whether the
// record exists is a store's question, and answering it would need I/O the
// kernel must not have.
func TestNewRefRejectsMalformedAddresses(t *testing.T) {
	for name, hash := range map[string]string{
		"empty":     "",
		"too short": "abc123",
		"uppercase": strings.ToUpper(strings.Repeat("ab", 32)),
		"non-hex":   strings.Repeat("zz", 32),
	} {
		if _, err := provenance.NewRef(hash); err == nil {
			t.Errorf("%s: expected rejection, got none", name)
		}
	}
}

// Confidence propagates as the weakest input, never as a product. Multiplying
// ordinal levels would be the arithmetic this type exists to refuse.
func TestConfidencePropagatesAsTheWeakest(t *testing.T) {
	got := provenance.Weakest(
		provenance.ConfidenceAsserted,
		provenance.ConfidenceEstimated,
		provenance.ConfidenceDerived,
	)
	if got != provenance.ConfidenceEstimated {
		t.Fatalf("expected estimated (the weakest), got %s", got)
	}
}

// The grammar ADR-0028 specified, enforced nowhere until an admission path
// exists to enforce it at. These fix what it means, so the admission path has
// something to be right about.
func TestContentAddressGrammar(t *testing.T) {
	const digest = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

	valid := []string{"sha256:" + digest}
	for _, v := range valid {
		s, err := provenance.NewSource(v)
		if err != nil {
			t.Fatalf("source %q: %v", v, err)
		}
		if err := s.CheckContentAddress(); err != nil {
			t.Fatalf("want %q admissible, got %v", v, err)
		}
	}

	// Each of these is something a person would plausibly put in a field named
	// `value`, which is the mistake the grammar exists to refuse.
	invalid := map[string]string{
		"a url":                  "https://broker.example/statement.pdf",
		"an account identifier":  "account-88213",
		"a filesystem path":      "/var/acquisitions/2026-03-01.json",
		"a bare digest":          digest,
		"the wrong algorithm":    "md5:" + digest,
		"a truncated digest":     "sha256:abc123",
		"an over-long digest":    "sha256:" + digest + "ff",
		"uppercase hexadecimal":  "sha256:" + strings.ToUpper(digest),
		"hex-shaped but not hex": "sha256:" + strings.Repeat("z", 64),
		"the prefix and nothing": "sha256:",
	}
	for name, v := range invalid {
		t.Run(name, func(t *testing.T) {
			s, err := provenance.NewSource(v)
			if err != nil {
				t.Fatalf("source: %v", err)
			}
			if err := s.CheckContentAddress(); !errors.Is(err, provenance.ErrMalformedSource) {
				t.Fatalf("want ErrMalformedSource for %q, got %v", v, err)
			}
		})
	}
}

// NewSource must NOT enforce the grammar. The wire codec constructs sources on
// decode, so enforcing there would be enforcement at decode rather than at
// admission — which does not reject bad new data, it makes existing data
// unreadable (ADR-0028).
func TestNewSourceDoesNotEnforceTheGrammar(t *testing.T) {
	if _, err := provenance.NewSource("not-a-digest"); err != nil {
		t.Fatalf("NewSource must accept what the grammar refuses, got %v", err)
	}
}
