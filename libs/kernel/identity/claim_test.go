package identity_test

import (
	"errors"
	"testing"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
)

// The scheme is FDOS vocabulary and must arrive canonical. ADR-0022 recorded
// that nothing prevented "Ticker" and "ticker" producing two entities; this is
// the half a type can close.
func TestNonCanonicalSchemeIsRejected(t *testing.T) {
	for _, scheme := range []string{"Ticker", "TICKER", " ticker", "ticker ", ""} {
		if _, err := identity.NewClaim(scheme, "PETR4"); !errors.Is(err, identity.ErrMalformedClaim) {
			t.Errorf("scheme %q: expected rejection, got %v", scheme, err)
		}
	}
}

// The value is the provider's data and is never touched. Canonicalising it is a
// resolver's decision to record, not a parser's to make silently.
func TestValueIsVerbatim(t *testing.T) {
	c := identity.MustClaim("ticker", "PETR4 ")
	if c.Value() != "PETR4 " {
		t.Fatalf("value was normalised: %q", c.Value())
	}
}

// Two claims differing only by whitespace are two claims. Deciding they are the
// same thing is resolution, and resolution is a recorded fact — not an equality
// operator.
func TestEqualityIsByteExactOnTheValue(t *testing.T) {
	if identity.MustClaim("ticker", "PETR4").Equal(identity.MustClaim("ticker", "PETR4 ")) {
		t.Fatal("whitespace-differing values compared equal; that is a resolution decision")
	}
	if !identity.MustClaim("ticker", "PETR4").Equal(identity.MustClaim("ticker", "PETR4")) {
		t.Fatal("identical claims compared unequal")
	}
}

func TestEmptyValueIsRejected(t *testing.T) {
	if _, err := identity.NewClaim("ticker", ""); !errors.Is(err, identity.ErrMalformedClaim) {
		t.Fatalf("expected rejection, got %v", err)
	}
}
