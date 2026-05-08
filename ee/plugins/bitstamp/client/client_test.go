package client

import (
	"encoding/json"
	"testing"
)

func TestUserTransactionUnmarshalJSONExtractsOnlyCurrencyAmountStrings(t *testing.T) {
	payload := []byte(`{
		"id": 458254264,
		"datetime": "2025-09-25 14:42:59.894846",
		"type": "36",
		"fee": "0.000000",
		"eur": "-5.00",
		"usdc": "5.810770",
		"usdc_eur": 0.86047000,
		"usd": 0.0
	}`)

	var tx UserTransaction
	if err := json.Unmarshal(payload, &tx); err != nil {
		t.Fatalf("unmarshal transaction: %v", err)
	}

	if tx.CurrencyAmounts["eur"] != "-5.00" {
		t.Fatalf("expected eur amount, got %q", tx.CurrencyAmounts["eur"])
	}
	if tx.CurrencyAmounts["usdc"] != "5.810770" {
		t.Fatalf("expected usdc amount, got %q", tx.CurrencyAmounts["usdc"])
	}
	if _, ok := tx.CurrencyAmounts["usdc_eur"]; ok {
		t.Fatalf("did not expect pair rate usdc_eur to be treated as a currency amount")
	}
	if _, ok := tx.CurrencyAmounts["usd"]; ok {
		t.Fatalf("did not expect numeric usd field to be treated as a currency amount")
	}
}

func TestNewDefaultsEmptyEndpoint(t *testing.T) {
	c, ok := New("bitstamp", "api-key", "api-secret", "").(*client)
	if !ok {
		t.Fatalf("expected concrete client")
	}

	if c.endpoint != DefaultEndpoint {
		t.Fatalf("expected default endpoint %q, got %q", DefaultEndpoint, c.endpoint)
	}
}
