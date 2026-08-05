package identity_test

import (
	"strings"
	"testing"

	"github.com/FabioCaffarello/fdos/libs/kernel/identity"
)

// A decoder must rebuild an identifier from its opaque value. Re-deriving would
// produce a different one for an entity whose natural key has since changed —
// which is the whole reason Derive is used once and never repeated (ADR-0007).
func TestRestoreRoundTripsADerivedID(t *testing.T) {
	original := identity.MustDerive(identity.KindAccount, "acct-1")

	back, err := identity.Restore(original.Kind(), original.String())
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if !back.Equal(original) {
		t.Fatalf("round trip changed the identifier: %s -> %s", original, back)
	}
}

// Derivation is deterministic: replaying the same input must yield the same
// identifier, or no report is byte-reproducible (Constitution §9).
func TestDeriveIsDeterministicAndSeparatesKinds(t *testing.T) {
	first := identity.MustDerive(identity.KindAccount, "x")
	second := identity.MustDerive(identity.KindAccount, "x")
	if !first.Equal(second) {
		t.Fatalf("Derive is not deterministic: %s vs %s", first, second)
	}
	if identity.MustDerive(identity.KindAccount, "x").Equal(identity.MustDerive(identity.KindParty, "x")) {
		t.Error("the same seed under different kinds produced the same identifier")
	}
	// Canonicalisation folds case and whitespace, so a statement quoting an
	// identifier differently still resolves to one entity.
	if !identity.MustDerive(identity.KindInstrument, "  br petra  ").
		Equal(identity.MustDerive(identity.KindInstrument, "BR PETRA")) {
		t.Error("seed canonicalisation did not fold case and whitespace")
	}
}

func TestRestoreRejectsMalformedValues(t *testing.T) {
	valid := identity.MustDerive(identity.KindParty, "p").String()
	for name, tc := range map[string]struct {
		kind  identity.Kind
		value string
	}{
		"unknown kind": {identity.KindUnspecified, valid},
		"empty":        {identity.KindParty, ""},
		"not a uuid":   {identity.KindParty, "not-a-uuid"},
		"uppercase":    {identity.KindParty, strings.ToUpper(valid)},
	} {
		if _, err := identity.Restore(tc.kind, tc.value); err == nil {
			t.Errorf("%s: expected rejection, got none", name)
		}
	}
}
