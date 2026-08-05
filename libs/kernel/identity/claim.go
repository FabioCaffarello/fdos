package identity

import (
	"fmt"
	"strings"
)

// ErrMalformedClaim is returned when a claim is not well-formed.
var ErrMalformedClaim = fmt.Errorf("identity: malformed claim")

// Claim is a {scheme, value} pair exactly as a provider stated it (ADR-0022).
//
// It is **not** an identity and deliberately carries no ID. This is the shape a
// connector can fully populate: it reads "PETR4" off a page and knows
// {scheme: "ticker", value: "PETR4"} and nothing more.
//
// A connector must not derive an ID from one. Deriving an identity from a
// ticker makes the ticker the primary key, and the day the external world
// reuses one, two different things merge silently inside an append-only ledger
// (ADR-0007).
type Claim struct {
	scheme string
	value  string
}

// NewClaim builds a claim.
//
// The **scheme** is FDOS vocabulary and must already be canonical: lowercase,
// no surrounding whitespace. A non-canonical scheme is rejected rather than
// normalised, because silently folding "Ticker" into "ticker" hides that a
// connector is emitting something the vocabulary does not contain.
//
// ADR-0022 recorded that nothing prevented `"Ticker"` and `"ticker"` producing
// two entities. This closes that half. The other half — two *different* schemes
// for the same concept, "ticker" and "symbol" — is vocabulary governance, and
// no type can solve it.
//
// The **value** is verbatim and is never touched. "PETR4 " with a trailing
// space is what the provider said; canonicalising it is a resolver's decision
// to record, not a parser's to make silently.
func NewClaim(scheme, value string) (Claim, error) {
	if scheme == "" {
		return Claim{}, fmt.Errorf("%w: scheme is empty", ErrMalformedClaim)
	}
	if scheme != strings.ToLower(scheme) || scheme != strings.TrimSpace(scheme) {
		return Claim{}, fmt.Errorf(
			"%w: scheme %q is not canonical — schemes are lowercase and unpadded", ErrMalformedClaim, scheme)
	}
	if value == "" {
		return Claim{}, fmt.Errorf("%w: value is empty", ErrMalformedClaim)
	}
	return Claim{scheme: scheme, value: value}, nil
}

// MustClaim is NewClaim for constants and tests.
func MustClaim(scheme, value string) Claim {
	c, err := NewClaim(scheme, value)
	if err != nil {
		panic(err)
	}
	return c
}

// Scheme returns the governed vocabulary term: "isin", "ticker",
// "account_number".
func (c Claim) Scheme() string { return c.scheme }

// Value returns the provider's value, verbatim.
func (c Claim) Value() string { return c.value }

// IsZero reports whether the claim is unset.
func (c Claim) IsZero() bool { return c.scheme == "" }

// Equal reports whether two claims are the same claim.
//
// Byte equality on the value, deliberately. Two claims differing only by
// whitespace are two claims: deciding they are the same thing is resolution,
// and resolution is recorded as a fact rather than performed by an equality
// operator.
func (c Claim) Equal(other Claim) bool {
	return c.scheme == other.scheme && c.value == other.value
}

// String renders "scheme:value".
func (c Claim) String() string { return c.scheme + ":" + c.value }
