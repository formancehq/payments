package mappers

import (
	"math/big"
	"testing"
)

var testCurrencies = map[string]int{
	"BTC":  8,
	"ETH":  18,
	"EUR":  2,
	"USD":  2,
	"USDC": 6,
}

func TestNormalizeCurrency(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"btc":   "BTC",
		" eur ": "EUR",
		"":      "",
	}
	for in, want := range cases {
		if got := NormalizeCurrency(in); got != want {
			t.Errorf("normalize %q = %q, want %q", in, got, want)
		}
	}
}

func TestIsZeroAmount(t *testing.T) {
	t.Parallel()
	cases := map[string]bool{
		"":               true,
		"0":              true,
		"0.0":            true,
		"0.00000000":     true,
		"-0":             true,
		"not-a-number":   true, // unparseable → treated as zero
		"0.0000001":      false,
		"1":              false,
		"-5.00":          false,
	}
	for in, want := range cases {
		if got := IsZeroAmount(in); got != want {
			t.Errorf("isZero %q = %v, want %v", in, got, want)
		}
	}
}

func TestAbsAmount(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		"-5.00": "5.00",
		"5.00":  "5.00",
		"":      "",
	}
	for in, want := range cases {
		if got := AbsAmount(in); got != want {
			t.Errorf("abs %q = %q, want %q", in, got, want)
		}
	}
}

func TestResolveSinglePaymentAsset(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		amounts   map[string]string
		wantAsset string
		wantMinor int64
		wantOk    bool
	}{
		{
			name:      "deposit: single non-zero known currency",
			amounts:   map[string]string{"btc": "1.5", "eur": "0", "usd": "0.0"},
			wantAsset: "BTC/8",
			wantMinor: 150000000,
			wantOk:    true,
		},
		{
			name:      "withdrawal: signed negative is absoluted",
			amounts:   map[string]string{"eur": "-25.50"},
			wantAsset: "EUR/2",
			wantMinor: 2550,
			wantOk:    true,
		},
		{
			name:    "two non-zero known currencies → not a payment",
			amounts: map[string]string{"eur": "-5.00", "usdc": "5.810770"},
		},
		{
			name:    "all zero",
			amounts: map[string]string{"eur": "0", "btc": "0"},
		},
		{
			name:    "unknown currency only",
			amounts: map[string]string{"xyz": "1.0"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			asset, amount, ok, err := ResolveSinglePaymentAsset(testCurrencies, tc.amounts)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ok != tc.wantOk {
				t.Fatalf("ok=%v want %v", ok, tc.wantOk)
			}
			if !ok {
				return
			}
			if asset != tc.wantAsset {
				t.Errorf("asset=%q want %q", asset, tc.wantAsset)
			}
			if amount.Cmp(big.NewInt(tc.wantMinor)) != 0 {
				t.Errorf("amount=%s want %d", amount, tc.wantMinor)
			}
		})
	}
}

func TestResolveTwoAssetConversion(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		amounts     map[string]string
		wantSrcSym  string
		wantSrcAmt  int64
		wantDstSym  string
		wantDstAmt  int64
		wantOk      bool
	}{
		{
			name: "Quentin #679 EUR -> USDC fixture",
			amounts: map[string]string{
				"eur":  "-5.00",
				"usdc": "5.810770",
				"usd":  "0.0",
				"btc":  "0.0",
			},
			wantSrcSym: "EUR", wantSrcAmt: 500,
			wantDstSym: "USDC", wantDstAmt: 5810770,
			wantOk: true,
		},
		{
			name:    "single non-zero → not a conversion",
			amounts: map[string]string{"eur": "-5.00"},
		},
		{
			name:    "two same-sign → not a conversion",
			amounts: map[string]string{"eur": "5.00", "usdc": "5.810770"},
		},
		{
			name:    "three non-zero → not a conversion",
			amounts: map[string]string{"eur": "-5", "usdc": "5.8", "btc": "0.0001"},
		},
		{
			name:    "unknown currency → not a conversion",
			amounts: map[string]string{"eur": "-5.00", "xyz": "5.0"},
		},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			src, dst, ok, err := ResolveTwoAssetConversion(testCurrencies, tc.amounts)
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if ok != tc.wantOk {
				t.Fatalf("ok=%v want %v", ok, tc.wantOk)
			}
			if !ok {
				return
			}
			if src.Symbol != tc.wantSrcSym || src.Amount.Cmp(big.NewInt(tc.wantSrcAmt)) != 0 {
				t.Errorf("src=%s/%s want %s/%d", src.Symbol, src.Amount, tc.wantSrcSym, tc.wantSrcAmt)
			}
			if dst.Symbol != tc.wantDstSym || dst.Amount.Cmp(big.NewInt(tc.wantDstAmt)) != 0 {
				t.Errorf("dst=%s/%s want %s/%d", dst.Symbol, dst.Amount, tc.wantDstSym, tc.wantDstAmt)
			}
		})
	}
}

func TestPrecisionFor(t *testing.T) {
	t.Parallel()
	if p, err := PrecisionFor(testCurrencies, "eur"); err != nil || p != 2 {
		t.Errorf("PrecisionFor eur = (%d, %v), want (2, nil)", p, err)
	}
	if _, err := PrecisionFor(testCurrencies, "xyz"); err == nil {
		t.Error("PrecisionFor unknown should error")
	}
}
