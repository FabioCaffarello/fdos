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

// The collisions the framed pre-image exists to make unrepresentable, measured
// on 2026-08-10 and reproduced here.
//
// The old rendering joined components with "\ninput=", "\nparam=" and "=", none
// of which is safe when a component may contain them. Two structurally different
// derivations shared the address `881ab834…`: a parameter whose value contained
// `\nparam=b=2` produced the same bytes as two separate parameters, and the
// inputs/parameters boundary was crossable the same way.
//
// A content address that two different derivations share is not an address. It is
// what a corrected fact points at, so a collision means a correction that names
// the wrong derivation.
func TestADerivationAddressCannotBeForged(t *testing.T) {
	method := mustMethod(t, "test.Method", "1")

	address := func(t *testing.T, inputs []string, params []provenance.Parameter) string {
		t.Helper()
		d, err := provenance.NewDerivation(
			method, inputs, params, nil, provenance.ConfidenceDerived)
		if err != nil {
			t.Fatal(err)
		}
		return d.Ref().Hash()
	}

	t.Run("a parameter value cannot forge another parameter", func(t *testing.T) {
		forged := address(t, nil, []provenance.Parameter{{Name: "a", Value: "1\nparam=b=2"}})
		real := address(t, nil, []provenance.Parameter{{Name: "a", Value: "1"}, {Name: "b", Value: "2"}})
		if forged == real {
			t.Fatalf("one parameter carrying %q addressed the same as two real parameters (%s)",
				"1\nparam=b=2", forged)
		}
	})

	t.Run("an input cannot cross into the parameters", func(t *testing.T) {
		forged := address(t, []string{"x\nparam=k=v"}, nil)
		real := address(t, []string{"x"}, []provenance.Parameter{{Name: "k", Value: "v"}})
		if forged == real {
			t.Fatalf("an input carrying %q addressed the same as an input plus a parameter (%s)",
				"x\nparam=k=v", forged)
		}
	})

	t.Run("a name and a value cannot swap the boundary", func(t *testing.T) {
		left := address(t, nil, []provenance.Parameter{{Name: "a", Value: "b=c"}})
		right := address(t, nil, []provenance.Parameter{{Name: "a=b", Value: "c"}})
		if left == right {
			t.Fatalf("moving the = between a parameter's name and value did not change the "+
				"address (%s)", left)
		}
	})
}

// Structure is part of the address, not only content. The nesting is what carries
// this: a flat component list could not distinguish two inputs from one input and
// one parameter, because the counts would not be encoded.
func TestDerivationStructureIsPartOfTheAddress(t *testing.T) {
	method := mustMethod(t, "test.Method", "1")

	address := func(t *testing.T, inputs []string, params []provenance.Parameter) string {
		t.Helper()
		d, err := provenance.NewDerivation(method, inputs, params, nil, provenance.ConfidenceDerived)
		if err != nil {
			t.Fatal(err)
		}
		return d.Ref().Hash()
	}

	twoInputs := address(t, []string{"a", "b"}, nil)
	oneEach := address(t, []string{"a"}, []provenance.Parameter{{Name: "b", Value: ""}})
	if twoInputs == oneEach {
		t.Errorf("two inputs addressed the same as one input and one parameter (%s)", twoInputs)
	}

	// And the caller's ordering of parameters still cannot change the address,
	// because NewDerivation sorts them. Framing must not have reintroduced
	// order-dependence.
	forward := address(t, nil, []provenance.Parameter{{Name: "a", Value: "1"}, {Name: "b", Value: "2"}})
	backward := address(t, nil, []provenance.Parameter{{Name: "b", Value: "2"}, {Name: "a", Value: "1"}})
	if forward != backward {
		t.Errorf("parameter order changed the address: %s vs %s", forward, backward)
	}
}

func mustMethod(t *testing.T, name, version string) provenance.Method {
	t.Helper()
	m, err := provenance.NewMethod(name, version)
	if err != nil {
		t.Fatal(err)
	}
	return m
}
