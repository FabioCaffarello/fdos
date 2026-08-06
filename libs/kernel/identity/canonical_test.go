package identity

// White box, deliberately. The guard this file exercises — that a fold cannot
// alter a value already in canonical form — is only provable by feeding it a
// fold that violates it, and `Fold` is a closed set with no unsafe member.
// Reaching `foldIsSafe` directly is what lets the check go red.

import (
	"errors"
	"strings"
	"testing"
	"unicode"
)

// The governing rule of ADR-0033, first half: no issuing standard, no rule.
//
// This is what keeps `ticker` out. A rule for it would be a judgement about
// whether two things are the same thing, which is a merge.
func TestARuleWithoutAnIssuingStandardIsRejected(t *testing.T) {
	if _, err := NewRule("ticker", "", 5, StripWhitespace); !errors.Is(err, ErrMalformedRule) {
		t.Fatalf("a rule with no standard was accepted: %v", err)
	}
	if _, err := NewRule("ticker", "   ", 5, StripWhitespace); !errors.Is(err, ErrMalformedRule) {
		t.Fatalf("whitespace passed as a standard: %v", err)
	}
}

// Without a canonical length there is no canonical set, so no fold can be
// checked against anything. A rule that cannot be checked must not exist.
func TestARuleWithoutACanonicalLengthIsRejected(t *testing.T) {
	for _, length := range []int{0, -1} {
		if _, err := NewRule("isin", "ISO 6166", length, StripWhitespace); !errors.Is(err, ErrMalformedRule) {
			t.Errorf("length %d was accepted: %v", length, err)
		}
	}
}

// The governing rule of ADR-0033, second half — and the negative test that
// makes the guard verified rather than merely present.
//
// `Fold`'s only member is safe, so the discriminating case has to be built: a
// predicate that alters an alphanumeric is exactly a rule that could merge two
// valid identifiers, and the guard must reject it.
func TestTheGuardRejectsAFoldThatAltersACanonicalValue(t *testing.T) {
	// A fold that strips a letter. Applied to ISINs it would map the two
	// distinct valid values "BRPETRACNPR0" and "RPETRACNPR0B"'s siblings onto
	// collisions — precisely the ADR-0007 failure.
	stripsALetter := func(r rune) bool { return r == 'B' }
	if foldIsSafe(stripsALetter) {
		t.Error("the guard accepted a fold that alters the letter B")
	}

	stripsADigit := func(r rune) bool { return unicode.IsDigit(r) }
	if foldIsSafe(stripsADigit) {
		t.Error("the guard accepted a fold that alters digits")
	}

	// A suffix-stripping fold — the ticker rule somebody will eventually want —
	// alters '.', which is outside the canonical alphabet, so the guard passes
	// it. That is not a hole: `ticker` cannot carry a rule at all, because it
	// names no standard. Asserted so the two halves of the governing rule are
	// visibly load-bearing rather than one being decorative.
	altersADot := func(r rune) bool { return r == '.' }
	if !foldIsSafe(altersADot) {
		t.Error("fixture is not discriminating: '.' is not in the canonical alphabet")
	}
	if _, err := NewRule("ticker", "", 5, StripWhitespace); err == nil {
		t.Error("ticker was allowed a rule; the standard requirement is what stops it")
	}

	// And the shipped fold passes, or the ruleset below could not exist.
	if !foldIsSafe(StripWhitespace.altersRune) {
		t.Error("StripWhitespace was rejected against the canonical alphabet")
	}
}

// A scheme is FDOS vocabulary and must already be canonical, exactly as
// NewClaim requires. A rule for "ISIN" would silently never match anything.
func TestARuleSchemeMustBeCanonical(t *testing.T) {
	for _, scheme := range []string{"", "ISIN", " isin", "isin "} {
		if _, err := NewRule(scheme, "ISO 6166", 12, StripWhitespace); !errors.Is(err, ErrMalformedRule) {
			t.Errorf("scheme %q was accepted: %v", scheme, err)
		}
	}
}

// A rule with no folds canonicalises nothing while claiming to. It is the
// shape that looks configured and does nothing.
func TestARuleWithNoFoldsIsRejected(t *testing.T) {
	if _, err := NewRule("isin", "ISO 6166", 12); !errors.Is(err, ErrMalformedRule) {
		t.Fatalf("a rule with no folds was accepted: %v", err)
	}
	if _, err := NewRule("isin", "ISO 6166", 12, FoldUnspecified); !errors.Is(err, ErrMalformedRule) {
		t.Fatalf("the zero fold was accepted: %v", err)
	}
}

// Two rules for one scheme make the answer depend on ordering, which is how a
// canonicalisation quietly becomes nondeterministic.
func TestARulesetRejectsADuplicateScheme(t *testing.T) {
	a := MustRule("isin", "ISO 6166", 12, StripWhitespace)
	b := MustRule("isin", "ISO 6166", 12, StripWhitespace)
	if _, err := NewRuleset("1", a, b); !errors.Is(err, ErrMalformedRuleset) {
		t.Fatalf("a duplicate scheme was accepted: %v", err)
	}
	if _, err := NewRuleset("", a); !errors.Is(err, ErrMalformedRuleset) {
		t.Fatalf("an unversioned ruleset was accepted: %v", err)
	}
}

// The property the whole safety argument rests on, asserted over the shipped
// ruleset rather than argued: folding a value already in canonical form
// returns it unchanged.
func TestFoldingIsTheIdentityOnCanonicalValues(t *testing.T) {
	rules := Canonicalisation()

	corpus := map[string][]string{
		"isin":  {"BRPETRACNPR0", "US0378331005", "GB0002634946"},
		"cusip": {"037833100", "594918104", "68389X105"},
		"sedol": {"0263494", "B1YW440", "2000019"},
		"figi":  {"BBG000B9XRY4", "BBG000BLNNH6", "BBG001S5N8V8"},
	}

	for scheme, values := range corpus {
		rule, ok := rules.RuleFor(scheme)
		if !ok {
			t.Fatalf("the shipped ruleset has no rule for %q", scheme)
		}
		for _, v := range values {
			if !rule.IsCanonical(v) {
				t.Fatalf("fixture is not discriminating: %q is not canonical for %s", v, scheme)
			}
			if got := rule.fold(v); got != v {
				t.Errorf("%s: folding the canonical %q produced %q", scheme, v, got)
			}
		}
	}
}

// What the rules are for: a vendor's spaced rendering and a clean one are one
// value, so they seed one identity.
func TestVendorSpacingFoldsAwayForStandardSchemes(t *testing.T) {
	rules := Canonicalisation()

	spaced := MustClaim("isin", "BR PETR ACNPR0")
	clean := MustClaim("isin", "BRPETRACNPR0")

	if rules.Fold(spaced).Value() != clean.Value() {
		t.Fatalf("folded to %q, want %q", rules.Fold(spaced).Value(), clean.Value())
	}

	// The consequence that matters: one identity, not two.
	a := MustDerive(KindInstrument, rules.Fold(spaced).String())
	b := MustDerive(KindInstrument, rules.Fold(clean).String())
	if !a.Equal(b) {
		t.Errorf("two renderings of one ISIN derived %s and %s", a, b)
	}
}

// The other half, and the one that protects real instruments: a scheme with no
// issuing standard is folded by nothing, so variants stay distinct.
func TestSchemesWithoutAStandardAreLeftAlone(t *testing.T) {
	rules := Canonicalisation()

	for _, c := range []Claim{
		MustClaim("ticker", "PETR4.SA"),
		MustClaim("ticker", "BRK B"),
		MustClaim("symbol", "PETR 4"),
		MustClaim("account_number", "0001234-5"),
		MustClaim("something_a_private_connector_invented", "x y"),
	} {
		if folded := rules.Fold(c); !folded.Equal(c) {
			t.Errorf("%s was folded to %s; it names no standard and must be left alone", c, folded)
		}
	}

	// PETR4 and PETR4.SA are a venue distinction. Deciding they are one
	// instrument is a merge, recorded as EntitiesIdentified and never performed.
	base := MustDerive(KindInstrument, rules.Fold(MustClaim("ticker", "PETR4")).String())
	suffixed := MustDerive(KindInstrument, rules.Fold(MustClaim("ticker", "PETR4.SA")).String())
	if base.Equal(suffixed) {
		t.Error("a ticker and its venue-suffixed form derived one identity")
	}
}

// A fold that empties a value would make every all-whitespace claim one claim,
// which is a merge performed rather than recorded.
func TestFoldingNeverEmptiesAValue(t *testing.T) {
	rules := Canonicalisation()
	c := MustClaim("isin", "   ")
	if folded := rules.Fold(c); folded.Value() != c.Value() {
		t.Errorf("an all-whitespace value folded to %q", folded.Value())
	}
}

// IsCanonical is a guard on rules, never a filter on claims: a malformed ISIN
// is still a claim FDOS admits, folds and can mint. Asserted here so that
// wiring it into admission later has to break a test that says why not.
func TestACanonicalityCheckDoesNotRejectAClaim(t *testing.T) {
	rules := Canonicalisation()
	rule, _ := rules.RuleFor("isin")

	malformed := "BR PETR"
	if rule.IsCanonical(strings.ReplaceAll(malformed, " ", "")) {
		t.Fatal("fixture is not discriminating: the value is canonical after folding")
	}

	c := MustClaim("isin", malformed)
	folded := rules.Fold(c)
	if folded.Value() != "BRPETR" {
		t.Errorf("a malformed value was not folded: %q", folded.Value())
	}
	if _, err := Derive(KindInstrument, folded.String()); err != nil {
		t.Errorf("a malformed but non-empty claim could not be minted: %v", err)
	}
}

// The zero ruleset is the generic floor and must be safe rather than panic:
// NewLedger refuses it, but nothing here may crash on it.
func TestTheZeroRulesetFoldsNothing(t *testing.T) {
	var rs Ruleset
	if !rs.IsZero() {
		t.Error("the zero ruleset does not report itself as zero")
	}
	c := MustClaim("isin", "BR PETR ACNPR0")
	if folded := rs.Fold(c); !folded.Equal(c) {
		t.Errorf("the zero ruleset folded %s to %s", c, folded)
	}
}
