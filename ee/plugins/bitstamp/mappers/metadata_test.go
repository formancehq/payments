package mappers

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
)

func TestPaymentMetadata(t *testing.T) {
	t.Parallel()
	tx := client.UserTransaction{
		Type:    TxTypeWithdrawal,
		Fee:     "0.005",
		OrderID: json.Number("0"),
	}
	got := PaymentMetadata(tx)
	if got[MetadataKeyType] != TxTypeWithdrawal {
		t.Errorf("missing type: %v", got)
	}
	if got[MetadataKeyFee] != "0.005" {
		t.Errorf("missing fee: %v", got)
	}
	if _, ok := got[MetadataKeyOrderID]; ok {
		t.Errorf("order_id 0 should not surface: %v", got)
	}

	tx2 := client.UserTransaction{Type: "0", Fee: "0", OrderID: json.Number("12345")}
	got2 := PaymentMetadata(tx2)
	if _, ok := got2[MetadataKeyFee]; ok {
		t.Errorf("zero fee should not surface: %v", got2)
	}
	if got2[MetadataKeyOrderID] != "12345" {
		t.Errorf("missing order_id: %v", got2)
	}
}

func TestOrderMetadata(t *testing.T) {
	t.Parallel()
	got := OrderMetadata("btcusd", "client-123", true)
	if got[MetadataKeyCurrencyPair] != "btcusd" || got[MetadataKeyOrderType] != "limit" {
		t.Errorf("missing required keys: %v", got)
	}
	if got[MetadataKeyClientOrderID] != "client-123" {
		t.Errorf("missing client_order_id: %v", got)
	}
	if got[MetadataKeyRetentionExpired] != "true" {
		t.Errorf("missing retention_expired flag: %v", got)
	}

	clean := OrderMetadata("btcusd", "", false)
	if _, ok := clean[MetadataKeyClientOrderID]; ok {
		t.Errorf("empty client_order_id should be omitted: %v", clean)
	}
	if _, ok := clean[MetadataKeyRetentionExpired]; ok {
		t.Errorf("retention_expired should be omitted when false: %v", clean)
	}
}

func TestConversionMetadata(t *testing.T) {
	t.Parallel()
	tx := client.UserTransaction{Type: TxTypeBuySell, Fee: "0.000000"}
	got := ConversionMetadata(tx, "eur_usdc", "0.86047")
	if got[MetadataKeyType] != TxTypeBuySell {
		t.Errorf("missing type: %v", got)
	}
	if got[MetadataKeyCurrencyPair] != "eur_usdc" {
		t.Errorf("missing currency_pair: %v", got)
	}
	if got[MetadataKeyRate] != "0.86047" {
		t.Errorf("missing rate: %v", got)
	}
	if _, ok := got[MetadataKeyFee]; ok {
		t.Errorf("zero fee should be omitted: %v", got)
	}
}

func TestMetadataKeysAreNamespaced(t *testing.T) {
	t.Parallel()
	// Guard against accidental drift from the com.bitstamp.spec/* prefix.
	for _, k := range []string{
		MetadataKeyType, MetadataKeyFee, MetadataKeyOrderID,
		MetadataKeyCurrencyPair, MetadataKeyOrderType, MetadataKeyClientOrderID,
		MetadataKeyRate, MetadataKeyRetentionExpired,
	} {
		if !startsWith(k, MetadataPrefix) {
			t.Errorf("%q not under %s", k, MetadataPrefix)
		}
	}
}

func startsWith(s, prefix string) bool {
	if len(s) < len(prefix) {
		return false
	}
	return s[:len(prefix)] == prefix
}
