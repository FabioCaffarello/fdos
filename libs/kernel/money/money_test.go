package money_test

import (
	"errors"
	"math/rand"
	"strings"
	"testing"

	"pgregory.net/rapid"

	"github.com/FabioCaffarello/fdos/libs/kernel/money"
)

// The property the entire decimal choice exists to guarantee (Constitution §9,
// ADR-0008): a projection that folds the same facts in a different but equally
// valid order must produce an identical total.
//
// This is the test that binary floating point fails, and the failure is not a
// rounding error. Summing the same five `float64` amounts in two orders:
//
//	[0.1, 0.2, 0.3, 1e16, -1e16] -> 0
//	[1e16, 0.1, 0.2, -1e16, 0.3] -> 0.29999999999999999
//
// The entire amount disappears in one ordering. `float64` addition is not
// associative, so the failure is structural rather than a matter of precision,
// and it shows up with ordinary values.
//
// That demonstration cannot live in this repository: `nofloat` refuses `float64`
// in the kernel, which is the point.
func TestFoldOrderDoesNotChangeTheTotal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		texts := rapid.SliceOfN(genDecimalText(), 2, 24).Draw(t, "amounts")

		amounts := make([]money.Money, 0, len(texts))
		for _, text := range texts {
			amounts = append(amounts, money.MustParse(text, "BRL"))
		}

		seed := rapid.Int64().Draw(t, "seed")
		shuffled := make([]money.Money, len(amounts))
		copy(shuffled, amounts)
		// Deterministic shuffle: the property is about order, and the test
		// itself must be reproducible from its inputs.
		r := rand.New(rand.NewSource(seed)) //nolint:gosec // not cryptographic
		r.Shuffle(len(shuffled), func(i, j int) {
			shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
		})

		first, err := sum(amounts)
		if err != nil {
			t.Fatalf("sum: %v", err)
		}
		second, err := sum(shuffled)
		if err != nil {
			t.Fatalf("sum shuffled: %v", err)
		}

		cmp, err := first.Cmp(second)
		if err != nil {
			t.Fatalf("cmp: %v", err)
		}
		if cmp != 0 {
			t.Fatalf("fold order changed the total: %s vs %s", first, second)
		}
	})
}

// Addition is exact, so a value survives being written and read back.
func TestParseRoundTripsThroughCanonicalText(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		text := genDecimalText().Draw(t, "amount")
		original := money.MustParse(text, "USD")

		reparsed, err := money.Parse(original.String(), "USD")
		if err != nil {
			t.Fatalf("reparse %q: %v", original.String(), err)
		}
		cmp, err := original.Cmp(reparsed)
		if err != nil {
			t.Fatalf("cmp: %v", err)
		}
		if cmp != 0 {
			t.Fatalf("round trip changed the value: %s -> %s", original, reparsed)
		}
	})
}

// Currency is part of the value. Mixing is an error, never a number.
func TestCrossCurrencyArithmeticIsRefused(t *testing.T) {
	brl := money.MustParse("10.00", "BRL")
	usd := money.MustParse("10.00", "USD")

	for name, op := range map[string]func() error{
		"Add": func() error { _, err := brl.Add(usd); return err },
		"Sub": func() error { _, err := brl.Sub(usd); return err },
		"Cmp": func() error { _, err := brl.Cmp(usd); return err },
	} {
		if err := op(); !errors.Is(err, money.ErrCurrencyMismatch) {
			t.Errorf("%s: expected ErrCurrencyMismatch, got %v", name, err)
		}
	}
}

// There is no default rounding. A RoundingContext built by struct literal
// without thought is the zero value, and the zero value is invalid — it does
// not silently mean half-even.
func TestRoundingContextHasNoUsableZeroValue(t *testing.T) {
	if _, err := money.NewRoundingContext(2, money.RoundingModeUnspecified); err == nil {
		t.Fatal("expected an unspecified rounding mode to be rejected")
	}
	if _, err := money.NewRoundingContext(0, money.RoundingModeHalfEven); err == nil {
		t.Fatal("expected zero precision to be rejected")
	}
}

// Division reports inexactness rather than swallowing it, so a caller can
// record the rounding in a derivation record (ADR-0010).
func TestDivisionReportsInexactness(t *testing.T) {
	rc := money.MustRoundingContext(10, money.RoundingModeHalfEven)

	exact, inexact, err := money.MustParse("10.00", "BRL").
		Div(money.MustParseQuantity("2", "share"), rc)
	if err != nil {
		t.Fatalf("exact division: %v", err)
	}
	if inexact {
		t.Errorf("10.00 / 2 reported inexact, got %s", exact)
	}

	_, inexact, err = money.MustParse("10.00", "BRL").
		Div(money.MustParseQuantity("3", "share"), rc)
	if err != nil {
		t.Fatalf("inexact division: %v", err)
	}
	if !inexact {
		t.Error("10.00 / 3 did not report inexact; a silent loss of precision is unauditable")
	}
}

func TestDivisionByZeroIsAnError(t *testing.T) {
	rc := money.MustRoundingContext(10, money.RoundingModeHalfEven)
	_, _, err := money.MustParse("1.00", "BRL").Div(money.MustParseQuantity("0", "share"), rc)
	if !errors.Is(err, money.ErrDivideByZero) {
		t.Fatalf("expected ErrDivideByZero, got %v", err)
	}
}

// Canonical form is enforced. A ledger that accepts "1e3" for a thousand is a
// ledger nobody can audit by reading.
func TestNonCanonicalTextIsRejected(t *testing.T) {
	for _, text := range []string{"", "1e3", "+1", " 1", "1.", ".1", "1.2.3", "1,000", "abc", "-"} {
		if _, err := money.Parse(text, "BRL"); err == nil {
			t.Errorf("Parse(%q) should have failed", text)
		}
	}
}

func TestInvalidCurrencyIsRejected(t *testing.T) {
	for _, code := range []string{"", "BR", "BRLL", "brl", "BR1"} {
		if _, err := money.Parse("1.00", code); !errors.Is(err, money.ErrInvalidCurrency) {
			t.Errorf("Parse with currency %q: expected ErrInvalidCurrency, got %v", code, err)
		}
	}
}

func sum(amounts []money.Money) (money.Money, error) {
	total := money.Zero(amounts[0].Currency())
	for _, a := range amounts {
		var err error
		if total, err = total.Add(a); err != nil {
			return money.Money{}, err
		}
	}
	return total, nil
}

// genDecimalText produces canonical decimal text with varied scale, which is
// what makes the fold-order property meaningful: equal scales would hide the
// associativity failure that motivates the whole design.
func genDecimalText() *rapid.Generator[string] {
	return rapid.Custom(func(t *rapid.T) string {
		negative := rapid.Bool().Draw(t, "negative")
		whole := rapid.StringMatching(`[0-9]{1,12}`).Draw(t, "whole")
		scale := rapid.IntRange(0, 8).Draw(t, "scale")

		out := whole
		if scale > 0 {
			frac := rapid.StringMatching(`[0-9]{1,8}`).Draw(t, "frac")
			if len(frac) > scale {
				frac = frac[:scale]
			}
			out = whole + "." + frac
		}
		if negative {
			out = "-" + out
		}
		return out
	})
}

// The exact context must refuse, not approximate. ADR-0008 decides two things
// this guards: "there is no default rounding context and no privileged mode",
// and "precision loss is signalled and recorded in the computation trace".
//
// Before the traps were added, every case below returned a nil error and a
// silently rounded value, because the context inherited apd.BaseContext's
// half-up rounding. Add, Sub and Mul are the three operations ADR-0008 and
// fdos.kernel.v1.Decimal both describe as exact, so a silent loss here is a
// wrong number with no trace of having been one.
func TestExactArithmeticRefusesToLosePrecision(t *testing.T) {
	// maxPrecision is 96 significant digits. A 1 in the 97th place is the
	// smallest input that cannot be represented, which makes it the case a
	// half-up default discards most quietly.
	huge := "1" + repeat("0", 96)
	one := "1"

	t.Run("Add", func(t *testing.T) {
		sum, err := money.MustParse(huge, "BRL").Add(money.MustParse(one, "BRL"))
		if !errors.Is(err, money.ErrInexact) {
			t.Fatalf("10^96 + 1 returned (%s, %v); want ErrInexact — a discarded unit "+
				"that reports success is unauditable", sum, err)
		}
	})

	t.Run("Sub", func(t *testing.T) {
		_, err := money.MustParse(huge, "BRL").Sub(money.MustParse("0."+one, "BRL"))
		if !errors.Is(err, money.ErrInexact) {
			t.Fatalf("10^96 - 0.1 returned %v; want ErrInexact", err)
		}
	})

	t.Run("Mul", func(t *testing.T) {
		big := "1." + repeat("1", 60)
		_, err := money.MustParse(big, "BRL").Mul(money.MustParseQuantity(big, "share"))
		if !errors.Is(err, money.ErrInexact) {
			t.Fatalf("a 61-digit product needing 121 digits returned %v; want ErrInexact", err)
		}
	})

	t.Run("QuantityAdd", func(t *testing.T) {
		_, err := money.MustParseQuantity(huge, "share").
			Add(money.MustParseQuantity(one, "share"))
		if !errors.Is(err, money.ErrInexact) {
			t.Fatalf("quantity 10^96 + 1 returned %v; want ErrInexact", err)
		}
	})
}

// The guard must not fire on arithmetic that is genuinely exact, or it would
// simply move the defect: a kernel that refuses 1.50 + 2.25 is no more usable
// than one that silently rounds 10^96 + 1.
//
// The zero case is the one worth stating. "1.500 - 1.500" is 0.000, not 0:
// trailing zeros are significant because they record the precision of the value
// (fdos.kernel.v1.Decimal), so a context that discarded them would be losing
// information the ledger is required to keep.
func TestExactArithmeticStillPermitsExactResults(t *testing.T) {
	for _, tc := range []struct {
		name, a, b, want string
	}{
		{"different scales", "1.50", "2.25", "3.75"},
		{"trailing zeros survive", "1.500", "-1.500", "0.000"},
		{"long but representable", "0." + repeat("1", 40), "0." + repeat("1", 40),
			"0." + repeat("2", 40)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			sum, err := money.MustParse(tc.a, "BRL").Add(money.MustParse(tc.b, "BRL"))
			if err != nil {
				t.Fatalf("%s + %s: %v", tc.a, tc.b, err)
			}
			if sum.String() != tc.want {
				t.Errorf("%s + %s = %s; want %s", tc.a, tc.b, sum, tc.want)
			}
		})
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for range n {
		out = append(out, s...)
	}
	return string(out)
}

// "Round to the cent" was inexpressible until now, and that is the defect
// ADR-0040 closed. Precision is a significant-digit budget: one context yields
// three different decimal places for 1/3, 1000/3 and 0.001/3, so it cannot say
// what a currency's minor units require.
//
// ISO 4217 publishes those minor units per currency — 0 for JPY, 3 for KWD, 4 for
// CLF — and Council Regulation (EC) No 1103/97 Article 5 requires rounding to the
// sub-unit. Each case below is one of those, which is why they are currencies
// rather than arbitrary numbers.
func TestQuantizeRoundsToACurrencysMinorUnits(t *testing.T) {
	for _, tc := range []struct {
		name, amount, currency, want string
		scale                        int32
		mode                         money.RoundingMode
		inexact                      bool
	}{
		{"BRL to the cent", "10.567", "BRL", "10.57", 2, money.RoundingModeHalfEven, true},
		{"JPY has no minor unit", "1234.62", "JPY", "1235", 0, money.RoundingModeHalfEven, true},
		{"KWD has three", "1.23456", "KWD", "1.235", 3, money.RoundingModeHalfEven, true},
		{"already at scale is exact", "10.50", "BRL", "10.50", 2, money.RoundingModeHalfEven, false},
		{"trailing zeros are added, not implied", "10.5", "BRL", "10.50", 2, money.RoundingModeHalfEven, false},
		{"negative scale rounds to tens", "1236", "BRL", "1240", -1, money.RoundingModeHalfEven, true},
		{"half-up and half-even differ", "10.565", "BRL", "10.57", 2, money.RoundingModeHalfUp, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			rc := money.MustRoundingContext(20, tc.mode).WithScale(tc.scale)
			got, inexact, err := money.MustParse(tc.amount, tc.currency).Quantize(rc)
			if err != nil {
				t.Fatalf("quantize: %v", err)
			}
			if got.String() != tc.want {
				t.Errorf("%s at scale %d = %s, want %s", tc.amount, tc.scale, got, tc.want)
			}
			if inexact != tc.inexact {
				t.Errorf("inexact = %v, want %v; a rounding nobody is told about is unauditable",
					inexact, tc.inexact)
			}
		})
	}
}

// A context with no scale cannot quantize, and must say so rather than pick one.
// Defaulting a scale would be the privileged default ADR-0008 forbids, and here
// it would additionally be a guess about a currency.
func TestQuantizeRefusesAContextWithoutAScale(t *testing.T) {
	rc := money.MustRoundingContext(10, money.RoundingModeHalfEven)
	if _, _, err := money.MustParse("1.005", "BRL").Quantize(rc); !errors.Is(err, money.ErrNoScale) {
		t.Fatalf("want ErrNoScale, got %v", err)
	}

	// And presence is not the same as zero: scale 0 is a real instruction.
	whole, _, err := money.MustParse("1.6", "JPY").Quantize(rc.WithScale(0))
	if err != nil {
		t.Fatalf("scale 0: %v", err)
	}
	if whole.String() != "2" {
		t.Errorf("scale 0 gave %s, want 2 — zero is JPY, not an unset field", whole)
	}
}

// The trap ADR-0040 named: quantize is total on scale and *partial* on precision.
// The decimal specification raises Invalid Operation when the quantized
// coefficient would exceed precision, and the specification's own example is
// 35236450.6 to two places at precision 9.
//
// It must surface as a domain error. A NaN a caller could propagate is the same
// failure the exact context had before its conditions were trapped: an arithmetic
// condition nobody is told about becomes a wrong number.
func TestQuantizeRefusesWhatPrecisionCannotHold(t *testing.T) {
	rc := money.MustRoundingContext(9, money.RoundingModeHalfEven).WithScale(2)

	got, _, err := money.MustParse("35236450.6", "BRL").Quantize(rc)
	if !errors.Is(err, money.ErrInexact) {
		t.Fatalf("quantize returned (%s, %v); want ErrInexact — a NaN a caller can propagate "+
			"is a wrong number with no trace of having been one", got, err)
	}
	if !strings.Contains(err.Error(), "precision 9") {
		t.Errorf("the error does not name the precision that bound: %v", err)
	}

	// The same amount fits once precision is sized for it, which is what a
	// caller must do rather than lowering the scale.
	roomy := money.MustRoundingContext(20, money.RoundingModeHalfEven).WithScale(2)
	if _, _, err := money.MustParse("35236450.6", "BRL").Quantize(roomy); err != nil {
		t.Errorf("precision 20 should hold 35236450.60: %v", err)
	}
}

// The context renders into derivation records, so its rendering is a hash
// pre-image. A context without a scale must render exactly as it did before the
// field existed, or every derivation that never used a scale would move address
// for a cosmetic reason.
func TestARoundingContextWithoutAScaleRendersUnchanged(t *testing.T) {
	rc := money.MustRoundingContext(10, money.RoundingModeHalfEven)
	if got := rc.String(); got != "precision=10,mode=half_even" {
		t.Errorf("rendered %q; adding the scale field moved a pre-image it should not touch", got)
	}
	if got := rc.WithScale(2).String(); got != "precision=10,mode=half_even,scale=2" {
		t.Errorf("a context with a scale rendered %q", got)
	}
}
