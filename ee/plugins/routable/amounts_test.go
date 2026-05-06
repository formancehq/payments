package routable

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
			got, err := toMinorUnits(c.amount, c.precision)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got.String() != c.minor {
				t.Fatalf("toMinorUnits(%s, %d) = %s, want %s", c.amount, c.precision, got.String(), c.minor)
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
			got := fromMinorUnits(n, c.precision)
			if got != c.want {
				t.Fatalf("fromMinorUnits(%s, %d) = %s, want %s", c.minor, c.precision, got, c.want)
			}
		})
	}
}

func TestPrecisionForRejectsUnknown(t *testing.T) {
	if _, err := precisionFor("XYZ"); err == nil {
		t.Fatal("expected error for unknown currency")
	}
	if _, err := precisionFor("usd"); err != nil {
		t.Fatalf("expected USD to be supported, got %v", err)
	}
}
