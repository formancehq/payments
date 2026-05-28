package mappers

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
)

var testCurrencyIndex = map[string]client.Currency{
	"BTC":  {Currency: "BTC", Decimals: 8},
	"ETH":  {Currency: "ETH", Decimals: 18},
	"EUR":  {Currency: "EUR", Decimals: 2},
	"USD":  {Currency: "USD", Decimals: 2},
	"USDC": {Currency: "USDC", Decimals: 6},
}

func TestAccountBalanceToPSPAccount(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "btc", Total: "1.5", Available: "1.5", Reserved: "0"}
	got, err := AccountBalanceToPSPAccount(testCurrencyIndex, bal)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Reference != "BTC" {
		t.Errorf("reference=%q, want BTC", got.Reference)
	}
	if got.Name == nil || *got.Name != "BTC" {
		t.Errorf("Name = %v, want pointer to \"BTC\" (currency ticker)", got.Name)
	}
	if !got.CreatedAt.Equal(BitstampGenesis) {
		t.Errorf("createdAt=%v, want BitstampGenesis %v", got.CreatedAt, BitstampGenesis)
	}
	if got.DefaultAsset == nil || *got.DefaultAsset != "BTC/8" {
		t.Errorf("defaultAsset=%v, want BTC/8", got.DefaultAsset)
	}
	if len(got.Metadata) != 0 {
		t.Errorf("Metadata must be empty when no enrichment is provided, got %+v", got.Metadata)
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

func TestAccountBalanceToPSPAccountEnriched(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "btc", Total: "1.5", Available: "1.5", Reserved: "0"}
	enrich := AccountEnrichment{
		Networks: []client.CurrencyNetwork{
			{Network: "xrpl", Deposit: "Disabled", Withdrawal: "Disabled"},
			{Network: "bitcoin", Deposit: "Enabled", Withdrawal: "Enabled", WithdrawalMinimumAmount: "0.00006"},
		},
		WithdrawalFees: []client.WithdrawalFee{
			{Currency: "btc", Network: "xrpl", Fee: "0"},
			{Currency: "btc", Network: "bitcoin", Fee: "0.00008"},
		},
		TradableMarkets: []client.MyMarket{
			{Name: "BTC/USD", URLSymbol: "btcusd"},
			{Name: "BTC/EUR", URLSymbol: "btceur"},
		},
		MakerFee:      "0.300",
		TakerFee:      "0.400",
		MinOrderValue: "10",
		MarketType:    "SPOT",
	}
	got, err := AccountBalanceToPSPAccountEnriched(testCurrencyIndex, bal, enrich)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Metadata[MetadataKeyFeeTierMaker] != "0.300" || got.Metadata[MetadataKeyFeeTierTaker] != "0.400" {
		t.Errorf("fee tier metadata missing: %+v", got.Metadata)
	}
	if got.Metadata[MetadataKeyMinOrderValue] != "10" || got.Metadata[MetadataKeyMarketSymbol] != "SPOT" {
		t.Errorf("market metadata missing: %+v", got.Metadata)
	}
	// Networks must be deterministically ordered by network name.
	if got.Metadata[MetadataKeyNetworks] != `[{"network":"bitcoin","deposit":"Enabled","withdrawal":"Enabled","withdrawal_minimum_amount":"0.00006"},{"network":"xrpl","deposit":"Disabled","withdrawal":"Disabled"}]` {
		t.Errorf("networks JSON not deterministically ordered: %s", got.Metadata[MetadataKeyNetworks])
	}
	// Withdrawal fees must be sorted by (currency, network).
	wantFees := `[{"currency":"btc","fee":"0.00008","network":"bitcoin"},{"currency":"btc","fee":"0","network":"xrpl"}]`
	if got.Metadata[MetadataKeyWithdrawalFees] != wantFees {
		t.Errorf("withdrawal fees JSON not deterministic: %s\nwant: %s", got.Metadata[MetadataKeyWithdrawalFees], wantFees)
	}
	// Tradable markets sorted alphabetically by market name (slash format).
	if got.Metadata[MetadataKeyTradableMarkets] != `["BTC/EUR","BTC/USD"]` {
		t.Errorf("tradable markets JSON not deterministic: %s", got.Metadata[MetadataKeyTradableMarkets])
	}
}

func TestAccountBalanceToPSPAccountEnriched_EmptyEnrichmentOmitsMetadata(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "btc"}
	got, err := AccountBalanceToPSPAccountEnriched(testCurrencyIndex, bal, AccountEnrichment{})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if len(got.Metadata) != 0 {
		t.Errorf("empty enrichment must leave Metadata nil/empty, got %+v", got.Metadata)
	}
}

func TestAccountBalanceToPSPAccountUnknownCurrency(t *testing.T) {
	t.Parallel()
	bal := client.AccountBalance{Currency: "xyz", Available: "1.0"}
	got, err := AccountBalanceToPSPAccount(testCurrencyIndex, bal)
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
	got, err := AccountBalanceToPSPAccount(testCurrencyIndex, client.AccountBalance{Currency: ""})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("empty currency should map to nil, got %#v", got)
	}
}
