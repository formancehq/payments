package mappers

import (
	"testing"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
)

func TestRawBalanceToPSPAccount_Spot(t *testing.T) {
	t.Parallel()
	got, err := RawBalanceToPSPAccount(testCurrencies, "XXBT", client.BalanceExEntry{Balance: "1.5", HoldTrade: "0.1"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil {
		t.Fatal("expected account, got nil")
	}
	// Reference is the raw code; DefaultAsset is the normalised symbol.
	if got.Reference != "XXBT" {
		t.Errorf("reference=%q want XXBT", got.Reference)
	}
	if *got.DefaultAsset != "BTC/8" {
		t.Errorf("defaultAsset=%q want BTC/8", *got.DefaultAsset)
	}
	if got.Name == nil || *got.Name != "BTC Spot" {
		t.Errorf("name=%v want 'BTC Spot'", got.Name)
	}
	if got.Metadata[MetadataPrefix+"wallet_type"] != WalletClassSpot {
		t.Errorf("wallet_type=%q want spot", got.Metadata[MetadataPrefix+"wallet_type"])
	}
	// The raw code is the Reference, so it isn't duplicated in metadata.
	if _, ok := got.Metadata[MetadataPrefix+"kraken_asset"]; ok {
		t.Error("kraken_asset must not be duplicated in account metadata")
	}
}

func TestRawBalanceToPSPAccount_EarnVariant(t *testing.T) {
	t.Parallel()
	got, err := RawBalanceToPSPAccount(testCurrencies, "XBT.M", client.BalanceExEntry{Balance: "0.3"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil || got.Reference != "XBT.M" {
		t.Fatalf("earn variant should keep raw ref, got %#v", got)
	}
	if got.Metadata[MetadataPrefix+"wallet_type"] != "rewards" {
		t.Errorf("wallet_type=%q want rewards", got.Metadata[MetadataPrefix+"wallet_type"])
	}
	if got.Name == nil || *got.Name != "BTC Rewards" {
		t.Errorf("name=%v want 'BTC Rewards'", got.Name)
	}
	if *got.DefaultAsset != "BTC/8" {
		t.Errorf("defaultAsset=%q want BTC/8", *got.DefaultAsset)
	}
}

func TestRawBalanceToPSPAccount_ZeroSpotStillEmits(t *testing.T) {
	t.Parallel()
	// The builder does NOT skip zero — the orchestrator owns that policy
	// so it can force-emit a zero spot account.
	got, err := RawBalanceToPSPAccount(testCurrencies, "ZUSD", client.BalanceExEntry{Balance: "0", HoldTrade: "0"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil || got.Reference != "ZUSD" {
		t.Fatalf("zero spot should still build an account, got %#v", got)
	}
}

func TestRawBalanceToPSPAccountUnknownAsset(t *testing.T) {
	t.Parallel()
	got, err := RawBalanceToPSPAccount(testCurrencies, "XYZ", client.BalanceExEntry{Balance: "1.0"})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("unknown asset should map to nil, got %#v", got)
	}
}
