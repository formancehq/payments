package mappers

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

func TestAccountBalanceToPSPBalance(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "btc", Total: "1.5", Available: "1.25000000", Reserved: "0.25"}
	raw, _ := json.Marshal(bal)
	asset := "BTC/8"
	parent := models.PSPAccount{
		Reference:    "BTC",
		DefaultAsset: &asset,
		Raw:          raw,
	}
	observedAt := time.Date(2025, 9, 25, 14, 42, 59, 0, time.UTC)

	got, err := AccountBalanceToPSPBalance(testCurrencies, parent, observedAt)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.AccountReference != "BTC" {
		t.Errorf("reference=%q, want BTC", got.AccountReference)
	}
	if got.Asset != "BTC/8" {
		t.Errorf("asset=%q, want BTC/8", got.Asset)
	}
	want := big.NewInt(125000000)
	if got.Amount.Cmp(want) != 0 {
		t.Errorf("amount=%s, want %s", got.Amount, want)
	}
	if !got.CreatedAt.Equal(observedAt) {
		t.Errorf("createdAt=%v, want %v", got.CreatedAt, observedAt)
	}
}

func TestAccountBalanceToPSPBalanceMissingRaw(t *testing.T) {
	t.Parallel()
	_, err := AccountBalanceToPSPBalance(testCurrencies, models.PSPAccount{Reference: "BTC"}, time.Now())
	if err == nil {
		t.Fatal("expected error on missing Raw")
	}
}

func TestAccountBalanceToPSPBalanceUnknownCurrency(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "xyz", Available: "1.0"}
	raw, _ := json.Marshal(bal)
	parent := models.PSPAccount{Reference: "XYZ", Raw: raw}
	got, err := AccountBalanceToPSPBalance(testCurrencies, parent, time.Now())
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("unknown currency should map to nil, got %#v", got)
	}
}
