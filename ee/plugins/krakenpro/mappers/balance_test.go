package mappers

import (
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
)

func TestRawBalanceToPSPBalance_Spot(t *testing.T) {
	t.Parallel()
	entry := client.BalanceExEntry{Balance: "2.0", HoldTrade: "0.5"}
	got, err := RawBalanceToPSPBalance(testCurrencies, "XXBT", entry, time.Now().UTC())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil balance")
	}
	// Reference is the raw code; asset is the normalised symbol.
	if got.AccountReference != "XXBT" {
		t.Errorf("ref=%q want XXBT", got.AccountReference)
	}
	if got.Asset != "BTC/8" {
		t.Errorf("asset=%q want BTC/8", got.Asset)
	}
	want := new(big.Int).SetInt64(150_000_000) // 1.5 BTC at 8 decimals
	if got.Amount.Cmp(want) != 0 {
		t.Errorf("amount=%s want %s", got.Amount, want)
	}
}

func TestRawBalanceToPSPBalance_EarnVariantKeepsRawRef(t *testing.T) {
	t.Parallel()
	got, err := RawBalanceToPSPBalance(testCurrencies, "XBT.M", client.BalanceExEntry{Balance: "0.3"}, time.Now())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil || got.AccountReference != "XBT.M" {
		t.Fatalf("earn variant should keep raw ref XBT.M, got %#v", got)
	}
	if got.Asset != "BTC/8" {
		t.Errorf("asset=%q want BTC/8", got.Asset)
	}
}

func TestRawBalanceToPSPBalance_Credit(t *testing.T) {
	t.Parallel()
	// available = balance + credit - credit_used - hold_trade
	//           = 100 + 50 - 10 - 5 = 135.00 USD (precision 2 → 13_500)
	entry := client.BalanceExEntry{Balance: "100.0", HoldTrade: "5.0", Credit: "50.0", CreditUsed: "10.0"}
	got, err := RawBalanceToPSPBalance(testCurrencies, "ZUSD", entry, time.Now())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil balance")
	}
	if got.Amount.Cmp(big.NewInt(13_500)) != 0 {
		t.Errorf("amount=%s want 13500", got.Amount)
	}
}

func TestRawBalanceToPSPBalance_CreditOnlyEmits(t *testing.T) {
	t.Parallel()
	// Zero spot balance but a credit line → available comes from credit;
	// the row must still emit (not skipped as empty).
	entry := client.BalanceExEntry{Balance: "0", HoldTrade: "0", Credit: "200.0", CreditUsed: "0"}
	got, err := RawBalanceToPSPBalance(testCurrencies, "ZUSD", entry, time.Now())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil || got.Amount.Cmp(big.NewInt(20_000)) != 0 {
		t.Fatalf("credit-only balance should emit 20000, got %#v", got)
	}
}

func TestRawBalanceToPSPBalanceUnknownAssetSkipped(t *testing.T) {
	t.Parallel()
	got, err := RawBalanceToPSPBalance(testCurrencies, "XYZ", client.BalanceExEntry{Balance: "1.0"}, time.Now())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("unknown asset should map to nil, got %#v", got)
	}
}

func TestRawBalanceToPSPBalanceZeroSkipped(t *testing.T) {
	t.Parallel()
	got, err := RawBalanceToPSPBalance(testCurrencies, "XXBT", client.BalanceExEntry{Balance: "0", HoldTrade: "0"}, time.Now())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("zero balance should map to nil, got %#v", got)
	}
}

func TestRawBalanceToPSPBalanceClampsNegative(t *testing.T) {
	t.Parallel()
	// hold_trade > balance should clamp to 0, not crash.
	got, err := RawBalanceToPSPBalance(testCurrencies, "ZUSD", client.BalanceExEntry{Balance: "1.0", HoldTrade: "2.0"}, time.Now())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil")
	}
	if got.Amount.Sign() != 0 {
		t.Errorf("expected clamp to 0, got %s", got.Amount)
	}
}
