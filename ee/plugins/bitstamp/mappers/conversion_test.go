package mappers

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

// TestUserTransactionToPSPConversion_Quentin679Fixture locks the exact
// regression case raised on the original Bitstamp PR by Quentin:
// {eur: -5.00, usdc: 5.810770, usdc_eur: 0.86047, type: 36}
// must produce EUR → USDC with the right minor units AND surface the
// rate in metadata.
func TestUserTransactionToPSPConversion_Quentin679Fixture(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 458254264,
		"datetime": "2025-09-25 14:42:59.894846",
		"type": "36",
		"fee": "0.000000",
		"eur": "-5.00",
		"usdc": "5.810770",
		"usdc_eur": 0.86047,
		"usd": "0.00",
		"btc": "0.00000000"
	}`)
	res, err := UserTransactionToPSPConversion(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Skip || res.Conversion == nil {
		t.Fatalf("expected conversion, got %#v", res)
	}
	c := res.Conversion
	if c.Reference != "458254264" {
		t.Errorf("reference=%q", c.Reference)
	}
	if c.SourceAsset != "EUR/2" || c.DestinationAsset != "USDC/6" {
		t.Errorf("assets: got (%s, %s), want (EUR/2, USDC/6)", c.SourceAsset, c.DestinationAsset)
	}
	if c.SourceAmount.Cmp(big.NewInt(500)) != 0 {
		t.Errorf("sourceAmount=%s, want 500", c.SourceAmount)
	}
	if c.DestinationAmount.Cmp(big.NewInt(5810770)) != 0 {
		t.Errorf("destinationAmount=%s, want 5810770", c.DestinationAmount)
	}
	if c.Status != models.CONVERSION_STATUS_COMPLETED {
		t.Errorf("status=%v, want COMPLETED", c.Status)
	}
	if c.Fee != nil {
		t.Errorf("zero fee should map to nil, got %v", c.Fee)
	}
	if c.Metadata[MetadataKeyRate] != "0.86047" {
		t.Errorf("rate metadata=%q, want 0.86047", c.Metadata[MetadataKeyRate])
	}
	if c.Metadata[MetadataKeyCurrencyPair] != "usdc_eur" && c.Metadata[MetadataKeyCurrencyPair] != "eur_usdc" {
		t.Errorf("currency_pair metadata=%q, want eur_usdc or usdc_eur", c.Metadata[MetadataKeyCurrencyPair])
	}
	if src, dst := c.SourceAccountReference, c.DestinationAccountReference; src == nil || dst == nil || *src != "EUR" || *dst != "USDC" {
		t.Errorf("account refs: got (%v, %v), want (EUR, USDC)", src, dst)
	}
	// Raw must round-trip back to a UserTransaction.
	var roundTrip client.UserTransaction
	if err := json.Unmarshal(c.Raw, &roundTrip); err != nil {
		t.Fatalf("raw round-trip: %v", err)
	}
	if roundTrip.ID != 458254264 {
		t.Errorf("raw id=%d, want 458254264", roundTrip.ID)
	}
}

func TestUserTransactionToPSPConversionWithFee(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 600,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "36",
		"fee": "0.05",
		"eur": "-100.00",
		"usdc": "116.21",
		"usdc_eur": "0.86047"
	}`)
	res, err := UserTransactionToPSPConversion(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if res.Conversion == nil {
		t.Fatal("expected conversion")
	}
	// pair key is "usdc_eur" → quote = EUR = source; fee 0.05 at EUR/2 = 5 minor units.
	if res.Conversion.Fee == nil || res.Conversion.Fee.Cmp(big.NewInt(5)) != 0 {
		t.Errorf("fee=%v, want 5 minor units at EUR/2", res.Conversion.Fee)
	}
	if res.Conversion.FeeAsset == nil || *res.Conversion.FeeAsset != "EUR/2" {
		t.Errorf("feeAsset=%v, want EUR/2", res.Conversion.FeeAsset)
	}
}

func TestUserTransactionToPSPConversionSkipsNonType36(t *testing.T) {
	t.Parallel()
	for _, txType := range []string{TxTypeDeposit, TxTypeWithdrawal, TxTypeMarketTrade, TxTypeStakingReward} {
		tx := newTx(`{
			"id": 700,
			"datetime": "2025-09-25 14:42:59.000000",
			"type": "` + txType + `",
			"fee": "0",
			"eur": "-5.00",
			"usdc": "5.81"
		}`)
		res, err := UserTransactionToPSPConversion(testCurrencies, tx)
		if err != nil {
			t.Fatalf("type %s err: %v", txType, err)
		}
		if !res.Skip || res.Conversion != nil {
			t.Errorf("type %s must be skipped, got %#v", txType, res)
		}
	}
}

func TestUserTransactionToPSPConversionSkipsDerivatives(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 800,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "36",
		"fee": "0",
		"eur": "-5.00",
		"usdc": "5.81",
		"margin_mode": "FLEXIBLE"
	}`)
	res, err := UserTransactionToPSPConversion(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Skip || !res.DerivativesRow {
		t.Errorf("expected DerivativesRow skip, got %#v", res)
	}
}

func TestUserTransactionToPSPConversionSkipsSingleAsset(t *testing.T) {
	t.Parallel()
	tx := newTx(`{
		"id": 900,
		"datetime": "2025-09-25 14:42:59.000000",
		"type": "36",
		"fee": "0",
		"eur": "-5.00",
		"usdc": "0",
		"btc": "0"
	}`)
	res, err := UserTransactionToPSPConversion(testCurrencies, tx)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if !res.Skip || res.Conversion != nil {
		t.Errorf("single-asset type-36 row must skip, got %#v", res)
	}
}
