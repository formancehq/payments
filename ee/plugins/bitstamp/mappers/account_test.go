package mappers

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
)

func TestAccountBalanceToPSPAccount(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "btc", Total: "1.5", Available: "1.5", Reserved: "0"}
	got, err := AccountBalanceToPSPAccount(testCurrencies, bal)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Reference != "BTC" {
		t.Errorf("reference=%q, want BTC", got.Reference)
	}
	if !got.CreatedAt.Equal(BitstampGenesis) {
		t.Errorf("createdAt=%v, want BitstampGenesis %v", got.CreatedAt, BitstampGenesis)
	}
	if got.DefaultAsset == nil || *got.DefaultAsset != "BTC/8" {
		t.Errorf("defaultAsset=%v, want BTC/8", got.DefaultAsset)
	}
	// Raw must round-trip back to the original AccountBalance.
	var roundTrip client.AccountBalance
	if err := json.Unmarshal(got.Raw, &roundTrip); err != nil {
		t.Fatalf("raw unmarshal: %v", err)
	}
	if roundTrip != bal {
		t.Errorf("raw round-trip mismatch: %#v vs %#v", roundTrip, bal)
	}
}

func TestAccountBalanceToPSPAccountUnknownCurrency(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "xyz", Available: "1.0"}
	got, err := AccountBalanceToPSPAccount(testCurrencies, bal)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got == nil {
		t.Fatal("expected account, got nil")
	}
	if got.DefaultAsset != nil {
		t.Errorf("unknown currency should have nil DefaultAsset, got %v", got.DefaultAsset)
	}
}

func TestAccountBalanceToPSPAccountEmptyCurrency(t *testing.T) {
	t.Parallel()
	got, err := AccountBalanceToPSPAccount(testCurrencies, client.AccountBalance{Currency: ""})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("empty currency should map to nil, got %#v", got)
	}
}
