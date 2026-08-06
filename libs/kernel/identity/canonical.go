package identity

import (
	"fmt"
	"strings"
	"unicode"
)

// Errors returned by the canonicalisation machinery.
var (
	// ErrMalformedRule is returned when a canonicalisation rule is not
	// well-formed, or when its folds are not provably safe for its scheme.
	ErrMalformedRule = fmt.Errorf("identity: malformed canonicalisation rule")

	// ErrMalformedRuleset is returned when a ruleset is not well-formed.
	ErrMalformedRuleset = fmt.Errorf("identity: malformed canonicalisation ruleset")
)

// Fold is one declared transform a canonicalisation rule may apply.
//
// **A closed set, and that is the point.** A rule composes folds; it does not
// carry a function. A `func(string) string` can express anything, including a
// particular provider's padding habit — and "rules are keyed to schemes, never
// to providers" is the boundary this whole design turns on. Making the wrong
// rule unwritable is rung 1; discouraging it in review is rung 6 (ADR-0033).
//
// Exactly one member today, which is honest: exactly one transform is currently
// defensible. Adding one is an ADR.
type Fold uint8

// Folds. The zero value is invalid.
const (
	FoldUnspecified Fold = iota

	// StripWhitespace removes every whitespace rune.
	//
	// Safe only for schemes whose standard defines the value as contiguous
	// alphanumerics — which is checked, not assumed. See [NewRule].
	StripWhitespace
)

// String returns the fold name.
func (f Fold) String() string {
	if f == StripWhitespace {
		return "strip_whitespace"
	}
	return "unspecified"
}

// Valid reports whether the fold is one of the defined ones.
func (f Fold) Valid() bool { return f == StripWhitespace }

// apply performs the transform.
func (f Fold) apply(value string) string {
	if f == StripWhitespace {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, value)
	}
	return value
}

// altersRune reports whether a value containing r can be changed by this fold.
//
// This is what makes the safety property checkable rather than asserted: a fold
// that cannot alter any rune of a scheme's canonical alphabet cannot alter a
// value already in canonical form, and therefore cannot merge two valid
// distinct values.
func (f Fold) altersRune(r rune) bool {
	if f == StripWhitespace {
		return unicode.IsSpace(r)
	}
	return true // an unknown fold is assumed to alter everything
}

// canonicalAlphabet is the set a standard-defined identifier is drawn from:
// uppercase alphanumerics. Enumerated rather than described, because the safety
// check below is an exhaustive test over it rather than an argument about it.
const canonicalAlphabet = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// foldIsSafe reports whether a fold can alter a value drawn from the canonical
// alphabet.
//
// Exhaustive over the 36 runes, so for a rune-wise fold this is a proof and not
// a sample. Split out from [NewRule] so the guard itself can be tested with a
// predicate that is deliberately unsafe — a check that has never gone red is
// unverified.
func foldIsSafe(alters func(rune) bool) bool {
	for _, r := range canonicalAlphabet {
		if alters(r) {
			return false
		}
	}
	return true
}

// Rule is the canonicalisation FDOS applies to one scheme's values before
// deriving an identity from them (ADR-0033).
//
// The governing rule, which every field exists to enforce:
//
//	A canonicalisation rule may exist only for a scheme whose issuing standard
//	defines a canonical form, and a rule may never alter a value already in
//	that form.
//
// The first half is why `standard` and `length` are required: without an
// issuing standard there is no canonical form to recover, so any fold would be
// a judgement about whether two things are the same thing — which is a merge,
// and ADR-0007 decided merges are recorded as EntitiesIdentified, never
// performed. That is why `ticker` has no rule and cannot have one.
//
// The second half is why folds are checked against the canonical alphabet.
type Rule struct {
	scheme   string
	standard string
	length   int
	folds    []Fold
}

// NewRule builds a canonicalisation rule, rejecting one that cannot be shown
// safe.
//
// `standard` names the issuing standard whose canonical form this recovers —
// "ISO 6166", not a provider. Nothing can stop a caller writing a provider's
// name here; that residue is rung 6 and ADR-0033 records it. What is enforced
// is that *something* must be named, which makes an unjustifiable rule
// conspicuous rather than routine.
//
// `length` is the canonical length the standard defines. It exists so the folds
// have a canonical set to be checked against, and for no other purpose — see
// [Rule.IsCanonical] for what it is deliberately not used for.
func NewRule(scheme, standard string, length int, folds ...Fold) (Rule, error) {
	if scheme == "" || scheme != strings.ToLower(scheme) || scheme != strings.TrimSpace(scheme) {
		return Rule{}, fmt.Errorf(
			"%w: scheme %q is not canonical — schemes are lowercase and unpadded", ErrMalformedRule, scheme)
	}
	if strings.TrimSpace(standard) == "" {
		return Rule{}, fmt.Errorf(
			"%w: scheme %q names no issuing standard; a scheme without one cannot carry a rule",
			ErrMalformedRule, scheme)
	}
	if length <= 0 {
		return Rule{}, fmt.Errorf(
			"%w: scheme %q declares no canonical length, so no fold can be checked against it",
			ErrMalformedRule, scheme)
	}
	if len(folds) == 0 {
		return Rule{}, fmt.Errorf("%w: scheme %q declares no folds", ErrMalformedRule, scheme)
	}
	for _, f := range folds {
		if !f.Valid() {
			return Rule{}, fmt.Errorf("%w: %d is not a defined fold", ErrMalformedRule, f)
		}
		if !foldIsSafe(f.altersRune) {
			return Rule{}, fmt.Errorf(
				"%w: fold %s can alter a value already in %s canonical form, so it could merge two valid identifiers",
				ErrMalformedRule, f, standard)
		}
	}
	return Rule{
		scheme:   scheme,
		standard: standard,
		length:   length,
		folds:    append([]Fold(nil), folds...),
	}, nil
}

// MustRule is NewRule for constants and tests.
func MustRule(scheme, standard string, length int, folds ...Fold) Rule {
	r, err := NewRule(scheme, standard, length, folds...)
	if err != nil {
		panic(err)
	}
	return r
}

// Scheme returns the scheme this rule canonicalises.
func (r Rule) Scheme() string { return r.scheme }

// Standard returns the issuing standard whose canonical form this recovers.
func (r Rule) Standard() string { return r.standard }

// Length returns the canonical length the standard defines.
func (r Rule) Length() int { return r.length }

// IsCanonical reports whether a value is already in the standard's canonical
// form.
//
// **A guard on rules, never a filter on claims.** A claim whose value is
// malformed for its scheme — a nine-character ISIN — is still admitted, still
// folded and still mintable. RFC-0007 decided that claims reach the ledger and
// that resolution is not a precondition of appending; using this to reject one
// would move a truth decision to the door, which is what Constitution §6 exists
// to prevent.
func (r Rule) IsCanonical(value string) bool {
	if len([]rune(value)) != r.length {
		return false
	}
	for _, c := range value {
		if !strings.ContainsRune(canonicalAlphabet, c) {
			return false
		}
	}
	return true
}

// fold applies the rule's transforms in declaration order.
func (r Rule) fold(value string) string {
	for _, f := range r.folds {
		value = f.apply(value)
	}
	return value
}

// Ruleset is a versioned set of per-scheme canonicalisation rules (ADR-0033).
//
// Versioned because changing a shipped rule changes what *resolves*, and does
// so retroactively — resolution folds at read time. That is a
// `contract/breaking`-class act needing its own ADR, and where it merges two
// previously distinct identities it needs recorded EntitiesIdentified facts.
// The version is what makes such a change nameable in a derivation record.
//
// The zero value applies no rules, which is the generic floor and is safe. It
// is nonetheless refused by NewLedger: the deployed ruleset determines what
// merges, so it is a decision to be made rather than defaulted into.
type Ruleset struct {
	version string
	rules   []Rule // a slice, not a map: four entries, and nothing here may iterate nondeterministically
}

// NewRuleset builds a versioned ruleset, rejecting a duplicate scheme.
func NewRuleset(version string, rules ...Rule) (Ruleset, error) {
	if strings.TrimSpace(version) == "" {
		return Ruleset{}, fmt.Errorf("%w: version is empty", ErrMalformedRuleset)
	}
	for i, r := range rules {
		if r.scheme == "" {
			return Ruleset{}, fmt.Errorf("%w: rule %d is unset", ErrMalformedRuleset, i)
		}
		for _, earlier := range rules[:i] {
			if earlier.scheme == r.scheme {
				return Ruleset{}, fmt.Errorf(
					"%w: two rules for scheme %q; which one applies would depend on ordering",
					ErrMalformedRuleset, r.scheme)
			}
		}
	}
	return Ruleset{version: version, rules: append([]Rule(nil), rules...)}, nil
}

// Version returns the ruleset version, for recording in a derivation.
func (rs Ruleset) Version() string { return rs.version }

// IsZero reports whether the ruleset is unset.
func (rs Ruleset) IsZero() bool { return rs.version == "" }

// RuleFor returns the rule for a scheme, if there is one.
//
// Most schemes have none, and that is the safe answer rather than a gap: an
// unregistered scheme gets the generic floor inside [Derive] and nothing more.
func (rs Ruleset) RuleFor(scheme string) (Rule, bool) {
	for _, r := range rs.rules {
		if r.scheme == scheme {
			return r, true
		}
	}
	return Rule{}, false
}

// Fold canonicalises a claim's value for its scheme, leaving the scheme alone.
//
// Applied **before** [Derive], never inside it: `canonicaliseSeed` stays the
// generic, scheme-blind floor it is (ADR-0033). A scheme-aware
// `canonicaliseSeed` would make every caller of Derive — including
// `ledger_stream`, which has no schemes at all — depend on the identifier
// vocabulary.
//
// A fold that would empty a value returns the claim unchanged. A fold's job is
// to recover a standard's form, and there is no standard under which the empty
// string is that form; folding "   " to "" would also make every all-whitespace
// value one value, which is a merge performed rather than recorded.
func (rs Ruleset) Fold(c Claim) Claim {
	rule, ok := rs.RuleFor(c.scheme)
	if !ok {
		return c
	}
	folded := rule.fold(c.value)
	if folded == "" {
		return c
	}
	return Claim{scheme: c.scheme, value: folded}
}

// CanonicalSeed returns the seed [Derive] will hash for this claim under this
// ruleset: the per-scheme fold composed with the generic floor.
//
// This exists so that resolution can be made to agree with minting **by
// construction** rather than by argument. Comparing folded claims alone is not
// enough: `ticker` has no rule, so "PETR4" and "PETR4 " fold to themselves and
// compare unequal — while Derive folds whitespace generically and assigns them
// one identity. Matching on the seed closes that gap, because it is literally
// what Derive hashes.
//
// The invariant it buys (ADR-0033): two claims resolve to the same identity if
// and only if minting them under the same kind would derive the same identity.
func (rs Ruleset) CanonicalSeed(c Claim) string {
	return canonicaliseSeed(rs.Fold(c).String())
}

// canonicalisationVersion versions the ruleset below.
//
// Bumped when a rule changes, which is an ADR-class act — see [Ruleset].
const canonicalisationVersion = "1"

// Canonicalisation returns the per-scheme ruleset this build ships (ADR-0033).
//
// Four schemes, one transform, and the reason each qualifies is the same: its
// issuing standard defines the value as a fixed run of contiguous
// alphanumerics, so whitespace in a rendering is presentation rather than
// content. Removing it is therefore the identity on every valid value, which is
// what makes it unable to merge two real instruments.
//
// **`ticker`, `symbol` and `account_number` are absent deliberately.** They have
// no issuing standard. "PETR4" versus "PETR4.SA" is a venue distinction and
// "BRK.B" versus "BRK B" is a guess, and a guess about whether two things are
// the same thing is a merge — recorded as EntitiesIdentified, never performed
// (ADR-0007).
func Canonicalisation() Ruleset {
	return Ruleset{
		version: canonicalisationVersion,
		rules: []Rule{
			MustRule("isin", "ISO 6166", 12, StripWhitespace),
			MustRule("cusip", "CUSIP Global Services", 9, StripWhitespace),
			MustRule("sedol", "London Stock Exchange", 7, StripWhitespace),
			MustRule("figi", "OMG FIGI", 12, StripWhitespace),
		},
	}
}
