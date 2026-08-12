package identity_test

import (
	"testing"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
)

// The defect this framing exists to make unrepresentable, measured on 2026-08-10
// and reproduced here: `scheme + ":" + value` rendered these two claims
// identically, so both derived one EntityId. Two different claims, one identity,
// in an append-only ledger — the silent merge ADR-0007 forbids.
//
// The pair is not exotic. A scheme is only shape-checked (lowercase, unpadded),
// membership in the vocabulary is deliberately open, and a value is verbatim by
// decision — so both of these are well-formed by every rule the constructors
// enforce.
func TestAColonCannotMoveTheClaimBoundary(t *testing.T) {
	left := identity.MustClaim("ticker", "x:y")
	right := identity.MustClaim("ticker:x", "y")

	t.Run("derivation", func(t *testing.T) {
		a, err := identity.DeriveFromClaim(identity.KindInstrument, left)
		if err != nil {
			t.Fatal(err)
		}
		b, err := identity.DeriveFromClaim(identity.KindInstrument, right)
		if err != nil {
			t.Fatal(err)
		}
		if a.Equal(b) {
			t.Fatalf("%s and %s derived one identifier (%s) — a silent merge", left, right, a)
		}
	})

	t.Run("resolution", func(t *testing.T) {
		rs := identity.Canonicalisation()
		if rs.CanonicalPreimage(left) == rs.CanonicalPreimage(right) {
			t.Fatalf("%s and %s share a canonical pre-image — resolution would answer one "+
				"claim with the other's identity", left, right)
		}
	})
}

// Injectivity is a property, not a case. Every separator the old encoding was
// defeated by is generated on both sides of the boundary, and no two distinct
// claims may share a derived identifier or a canonical seed.
func TestClaimDerivationIsInjective(t *testing.T) {
	// The characters that mattered: ':' joined scheme to value, and '\n', '='
	// are what the derivation pre-image used elsewhere. Plus a plain case.
	fragments := []string{"a", "a:", ":a", "a:b", "a\nb", "a=b", "a:b:c", ""}

	seeds := map[string]identity.Claim{}
	ids := map[string]identity.Claim{}
	rs := identity.Canonicalisation()

	for _, scheme := range fragments {
		for _, value := range fragments {
			// A scheme must be lowercase and unpadded, a value non-empty; skip
			// what the constructor legitimately refuses rather than asserting
			// the framing fixes input validation.
			c, err := identity.NewClaim("t"+scheme, "v"+value)
			if err != nil {
				continue
			}

			seed := rs.CanonicalPreimage(c)
			if prior, clash := seeds[seed]; clash {
				t.Errorf("canonical pre-image collision: %q and %q", prior, c)
			}
			seeds[seed] = c

			id, dErr := identity.DeriveFromClaim(identity.KindInstrument, c)
			if dErr != nil {
				t.Fatalf("%q: %v", c, dErr)
			}
			if prior, clash := ids[id.String()]; clash {
				t.Errorf("identifier collision: %q and %q both derived %s", prior, c, id)
			}
			ids[id.String()] = c
		}
	}
	if len(ids) == 0 {
		t.Fatal("the generator produced no valid claims; the test proves nothing")
	}
}

// The generic floor is unchanged, and the asymmetry in it is deliberate.
//
// ADR-0033 decided that canonicalisation is per scheme, that the floor stays
// scheme-blind, and that `ticker` carries no rule because no standard can say
// whether two renderings name one instrument. `resolve_test.go` records the
// consequence PR #51 measured: the folded seed is `scheme:value`, so a *trailing*
// space is stripped while a *leading* one survives as an internal separator the
// floor collapses but cannot remove.
//
// Both directions are pinned here, because framing the pre-image was the obvious
// place to "tidy" that asymmetry away — and doing so would strip a leading space,
// which is a decision that `" PETR4"` and `"PETR4"` name one instrument. That is
// a merge, and ADR-0007 records merges as facts rather than performing them.
func TestFramingPreservesTheGenericFloorAndItsAsymmetry(t *testing.T) {
	derive := func(t *testing.T, value string) identity.ID {
		t.Helper()
		id, err := identity.DeriveFromClaim(identity.KindInstrument, identity.MustClaim("ticker", value))
		if err != nil {
			t.Fatal(err)
		}
		return id
	}

	base := derive(t, "PETR4")

	// Folded by the floor: case, and whitespace runs that are already internal.
	for _, same := range []string{"PETR4 ", "petr4", "PETR4\t"} {
		if got := derive(t, same); !got.Equal(base) {
			t.Errorf("%q derived a different identifier from %q; the floor folds case and "+
				"trailing whitespace (ADR-0033)", same, "PETR4")
		}
	}

	// Not folded, and must not become folded: a leading space is significant
	// because the seed is `scheme:value`, and removing it would be a `ticker`
	// rule that ADR-0033 forbids.
	for _, apart := range []string{" PETR4", "\tPETR4\n", "PETR 4"} {
		if got := derive(t, apart); got.Equal(base) {
			t.Errorf("%q derived the same identifier as %q; framing must not smuggle in a "+
				"canonicalisation change, and folding these is a merge", apart, "PETR4")
		}
	}
}

// The namespace is the one ADR-0040 minted, and provably not the registered DNS
// namespace the code used before. Asserted through behaviour rather than by
// reading the constant, so the test fails if the constant is edited back.
//
// Both literals were computed independently of this implementation — SHA-1 over
// the namespace bytes followed by the framed pre-image, per RFC 9562 §5.5 — so
// they pin the specification rather than the code. Deriving them here instead
// would make the assertion circular and hide exactly the regression it exists
// for.
func TestTheNamespaceIsNotTheRegisteredDNSNamespace(t *testing.T) {
	const (
		// UUIDv5 of frame("fdos.identity.claim-seed.v1", "instrument", "TICKER",
		// "TICKER:PETR4") under the FDOS root recorded in ADR-0040.
		wantUnderFDOSRoot = "48d00d5b-fb3a-589c-944f-e081d96cf449"

		// The same pre-image under 6ba7b810-9dad-11d1-80b4-00c04fd430c8, the
		// registered DNS namespace this package used until ADR-0040. Asserted
		// as a value the derivation must never produce.
		neverUnderDNSNamespace = "dae66628-99ae-5d32-ba29-de46fccf354b"
	)

	got, err := identity.DeriveFromClaim(identity.KindInstrument, identity.MustClaim("ticker", "PETR4"))
	if err != nil {
		t.Fatal(err)
	}
	if got.String() == neverUnderDNSNamespace {
		t.Fatalf("derived %s — that is this pre-image under the RFC 9562 registered DNS "+
			"namespace, so the FDOS root constant has been reverted (ADR-0040)", got)
	}
	if got.String() != wantUnderFDOSRoot {
		t.Fatalf("derived %s, want %s — the namespace constant or the pre-image framing "+
			"changed, and either moves every identifier FDOS will ever assign (ADR-0040)", got, wantUnderFDOSRoot)
	}
}
