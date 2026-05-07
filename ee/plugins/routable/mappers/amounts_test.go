package mappers

import (
	"math/big"
	"testing"
)

func TestToMinorUnitsRoundTrip(t *testing.T) {
	cases := []struct {
		amount    string
		precision int
		minor     string
	}{
		{"100.00", 2, "10000"},
		{"100.50", 2, "10050"},
		{"100.555", 2, "10056"}, // half-up rounding
		{"100", 0, "100"},
		{"-25.30", 2, "-2530"},
		{"-25.305", 2, "-2531"},
		{"0.001", 3, "1"},
	}
	for _, c := range cases {
		t.Run(c.amount, func(t *testing.T) {
			got, err := ToMinorUnits(c.amount, c.precision)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.String() != c.minor {
				t.Fatalf("ToMinorUnits(%s, %d) = %s, want %s", c.amount, c.precision, got.String(), c.minor)
			}
		})
	}
}

func TestFromMinorUnits(t *testing.T) {
	cases := []struct {
		minor     string
		precision int
		want      string
	}{
		{"10050", 2, "100.50"},
		{"10000", 2, "100.00"},
		{"100", 0, "100"},
		{"-2530", 2, "-25.30"},
		{"7", 3, "0.007"},
	}
	for _, c := range cases {
		t.Run(c.minor, func(t *testing.T) {
			n, _ := new(big.Int).SetString(c.minor, 10)
			got := FromMinorUnits(n, c.precision)
			if got != c.want {
				t.Fatalf("FromMinorUnits(%s, %d) = %s, want %s", c.minor, c.precision, got, c.want)
			}
		})
	}
}

// PrecisionFor must reject unknown ISO codes loudly (no silent default to
// 2). At 200k tx/wk an unknown currency means a misconfigured PSP entry,
// which we want surfaced — not coerced into wrong precision.
func TestPrecisionForRejectsUnknown(t *testing.T) {
	if _, err := PrecisionFor("XYZ"); err == nil {
		t.Fatal("expected error for unknown currency")
	}
	if _, err := PrecisionFor("usd"); err != nil {
		t.Fatalf("expected USD to be supported, got %v", err)
	}
}

// Currency-specific precisions: JPY (0), USD (2), KWD (3), BHD (3),
// VND (0). Catches a regression where someone defaults to 2 minor units.
func TestPrecisionForCoversNonTwoDigitCurrencies(t *testing.T) {
	cases := map[string]int{
		"USD": 2,
		"EUR": 2,
		"JPY": 0,
		"VND": 0,
		"KWD": 3,
		"BHD": 3,
		"OMR": 3,
	}
	for code, want := range cases {
		t.Run(code, func(t *testing.T) {
			got, err := PrecisionFor(code)
			if err != nil {
				t.Fatalf("PrecisionFor(%q) unexpected error: %v", code, err)
			}
			if got != want {
				t.Fatalf("PrecisionFor(%q) = %d, want %d", code, got, want)
			}
		})
	}
}

// JPY and KWD round-trip checks: a JPY 100 must serialize/round to "100"
// (no decimal), and KWD 1.234 must serialize to "1234" minor units.
func TestRoundTripNonTwoDigitCurrencies(t *testing.T) {
	cases := []struct {
		amount    string
		precision int
		minor     string
	}{
		{"100", 0, "100"},                 // JPY
		{"99999", 0, "99999"},             // JPY large
		{"1.234", 3, "1234"},              // KWD
		{"0.001", 3, "1"},                 // KWD smallest unit
		{"1234.567", 3, "1234567"},        // BHD
	}
	for _, c := range cases {
		t.Run(c.amount, func(t *testing.T) {
			got, err := ToMinorUnits(c.amount, c.precision)
			if err != nil {
				t.Fatalf("ToMinorUnits err: %v", err)
			}
			if got.String() != c.minor {
				t.Fatalf("ToMinorUnits(%s, %d) = %s, want %s", c.amount, c.precision, got.String(), c.minor)
			}
		})
	}
}
