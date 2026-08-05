package provenance_test

import (
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
