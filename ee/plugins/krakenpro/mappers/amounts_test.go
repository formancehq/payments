package mappers

import (
	"math/big"
	"testing"
)

func TestParseDecimalAmount(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in        string
		precision int
		want      *big.Int
		expectErr bool
	}{
		{"1.50000000", 8, big.NewInt(150000000), false},
		{"100.00", 2, big.NewInt(10000), false},
		{"0", 8, big.NewInt(0), false},
		{"1.123456789", 6, big.NewInt(1123456), false}, // truncates to precision
		{"", 8, nil, true},
	}
	for _, c := range cases {
		got, err := ParseDecimalAmount(c.in, c.precision)
		if c.expectErr {
			if err == nil {
				t.Errorf("ParseDecimalAmount(%q,%d) expected error", c.in, c.precision)
			}
			continue
		}
		if err != nil {
			t.Errorf("ParseDecimalAmount(%q,%d) err=%v", c.in, c.precision, err)
			continue
		}
		if got.Cmp(c.want) != 0 {
			t.Errorf("ParseDecimalAmount(%q,%d) = %s, want %s", c.in, c.precision, got, c.want)
		}
	}
}

func TestSubDecimalClampsToZero(t *testing.T) {
	t.Parallel()
	got, err := SubDecimal("1.0", "2.0", 2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Sign() != 0 {
		t.Errorf("expected 0 (clamped) got %s", got)
	}
}

func TestSubDecimal(t *testing.T) {
	t.Parallel()
	got, err := SubDecimal("100.00", "25.00", 2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	want := big.NewInt(7500)
	if got.Cmp(want) != 0 {
		t.Errorf("got %s want %s", got, want)
	}
}

func TestIsZeroAmount(t *testing.T) {
	t.Parallel()
	if !IsZeroAmount("") {
		t.Error("empty must be zero")
	}
	if !IsZeroAmount("0") {
		t.Error("zero literal must be zero")
	}
	if !IsZeroAmount("0.0000") {
		t.Error("padded zero must be zero")
	}
	if IsZeroAmount("0.01") {
		t.Error("non-zero must not be zero")
	}
}

func TestIsNegative(t *testing.T) {
	t.Parallel()
	if !IsNegative("-1") {
		t.Error("-1 is negative")
	}
	if IsNegative("1") {
		t.Error("1 is not negative")
	}
	if IsNegative("") {
		t.Error("empty is not negative")
	}
}
