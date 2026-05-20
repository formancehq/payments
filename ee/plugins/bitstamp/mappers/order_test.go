package mappers

import (
	"encoding/json"
	"math/big"
	"testing"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

func TestSplitCurrencyPair(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		base string
		quote string
		err  bool
	}{
		{"btcusd", "BTC", "USD", false},
		{"btcusdc", "BTC", "USDC", false},
		{"BTC/USD", "BTC", "USD", false},
		{"", "", "", true},
		{"toolongtoknow", "", "", true},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			b, q, err := splitCurrencyPair(tc.in)
			if (err != nil) != tc.err {
				t.Fatalf("err: %v wantErr=%v", err, tc.err)
			}
			if !tc.err && (b != tc.base || q != tc.quote) {
				t.Errorf("got (%s, %s), want (%s, %s)", b, q, tc.base, tc.quote)
			}
		})
	}
}

func TestAccountReferencesForDirection(t *testing.T) {
	t.Parallel()
	src, dst := accountReferencesForDirection(models.ORDER_DIRECTION_BUY, "BTC", "USD")
	if src != "USD" || dst != "BTC" {
		t.Errorf("BUY: got (%s, %s), want (USD, BTC)", src, dst)
	}
	src, dst = accountReferencesForDirection(models.ORDER_DIRECTION_SELL, "BTC", "USD")
	if src != "BTC" || dst != "USD" {
		t.Errorf("SELL: got (%s, %s), want (BTC, USD)", src, dst)
	}
}

func TestAverageFillPrice(t *testing.T) {
	t.Parallel()
	// Filled 0.5 BTC for 30,000 USD total → avg price 60,000 USD/BTC.
	// USD precision 2, BTC precision 8.
	quote := big.NewInt(3_000_000)         // 30,000 USD in cents
	base := big.NewInt(50_000_000)         // 0.5 BTC at 8 decimals
	got := averageFillPrice(quote, base, 8, 2)
	want := big.NewInt(6_000_000) // 60,000 USD in cents
	if got.Cmp(want) != 0 {
		t.Errorf("avg fill price = %s, want %s", got, want)
	}

	// No fills: returns zero.
	if averageFillPrice(big.NewInt(0), big.NewInt(0), 8, 2).Sign() != 0 {
		t.Error("no fills should return zero, not error")
	}
}

func TestOrderStatusToPSPOrderFinishedFilled(t *testing.T) {
	t.Parallel()
	tracked := TrackedOrderInput{
		Price:       "60000.00",
		FirstSeenAt: time.Date(2025, 9, 25, 14, 0, 0, 0, time.UTC),
	}
	status := client.OrderStatus{
		ID:              json.Number("123"),
		Datetime:        "2025-09-25 14:42:59.000000",
		Type:            "0", // BUY
		Subtype:         OrderSubtypeLimit,
		Market:          "BTC/USD",
		AmountRemaining: "0",
		Status:          OrderStatusFinished,
		Transactions: []client.OrderTransaction{{
			TID:             1,
			Type:            0,
			Datetime:        "2025-09-25 14:42:59.000000",
			Price:           "60000.00",
			Fee:             "15.00",
			CurrencyAmounts: map[string]string{"btc": "0.5", "usd": "30000.00"},
		}},
	}
	got, err := OrderStatusToPSPOrder(testCurrencies, OrderMapInput{Status: status, Tracked: tracked})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Reference != "123" {
		t.Errorf("reference=%q, want 123", got.Reference)
	}
	if got.Direction != models.ORDER_DIRECTION_BUY {
		t.Errorf("direction=%v, want BUY", got.Direction)
	}
	if got.Status != models.ORDER_STATUS_FILLED {
		t.Errorf("status=%v, want FILLED", got.Status)
	}
	if got.BaseQuantityOrdered == nil || got.BaseQuantityOrdered.Cmp(big.NewInt(50_000_000)) != 0 {
		t.Errorf("baseOrdered=%s, want 50_000_000 (filled + remaining)", got.BaseQuantityOrdered)
	}
	if got.BaseQuantityFilled.Cmp(big.NewInt(50_000_000)) != 0 {
		t.Errorf("baseFilled=%s, want 50_000_000", got.BaseQuantityFilled)
	}
	if got.QuoteAmount.Cmp(big.NewInt(3_000_000)) != 0 {
		t.Errorf("quoteAmount=%s, want 3_000_000", got.QuoteAmount)
	}
	if got.Fee.Cmp(big.NewInt(1500)) != 0 {
		t.Errorf("fee=%s, want 1500", got.Fee)
	}
	if got.SourceAsset != "USD/2" || got.DestinationAsset != "BTC/8" {
		t.Errorf("BUY assets: got (%s, %s), want (USD/2, BTC/8)", got.SourceAsset, got.DestinationAsset)
	}
	if got.Type != models.ORDER_TYPE_LIMIT {
		t.Errorf("type=%v, want LIMIT", got.Type)
	}
	if got.TimeInForce != models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED {
		t.Errorf("tif=%v, want GTC", got.TimeInForce)
	}
	if got.LimitPrice == nil || got.LimitPrice.Cmp(big.NewInt(6_000_000)) != 0 {
		t.Errorf("LimitPrice = %v, want 6_000_000 (60000.00 USD at quotePrec=2)", got.LimitPrice)
	}
	// CreatedAt prefers the wire datetime over the first-sight value.
	wantCreated := time.Date(2025, 9, 25, 14, 42, 59, 0, time.UTC)
	if !got.CreatedAt.Equal(wantCreated) {
		t.Errorf("CreatedAt = %v, want %v", got.CreatedAt, wantCreated)
	}
}

func TestOrderStatusToPSPOrderPartialFill(t *testing.T) {
	t.Parallel()
	tracked := TrackedOrderInput{Price: "60000.00", FirstSeenAt: time.Now().UTC()}
	status := client.OrderStatus{
		ID:              json.Number("124"),
		Type:            "1", // SELL
		Subtype:         OrderSubtypeLimit,
		Market:          "BTC/USD",
		AmountRemaining: "0.75000000",
		Status:          OrderStatusOpen,
		Transactions: []client.OrderTransaction{{
			TID: 2, Type: 1, Price: "60000.00", Fee: "7.50",
			CurrencyAmounts: map[string]string{"btc": "0.25", "usd": "15000.00"},
		}},
	}
	got, err := OrderStatusToPSPOrder(testCurrencies, OrderMapInput{Status: status, Tracked: tracked})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Status != models.ORDER_STATUS_PARTIALLY_FILLED {
		t.Errorf("status=%v, want PARTIALLY_FILLED", got.Status)
	}
	if got.SourceAsset != "BTC/8" || got.DestinationAsset != "USD/2" {
		t.Errorf("SELL assets: got (%s, %s), want (BTC/8, USD/2)", got.SourceAsset, got.DestinationAsset)
	}
	if got.BaseQuantityOrdered == nil || got.BaseQuantityOrdered.Cmp(big.NewInt(100_000_000)) != 0 {
		t.Errorf("baseOrdered = %s, want 100_000_000 (0.25 filled + 0.75 remaining)", got.BaseQuantityOrdered)
	}
}

func TestOrderStatusToPSPOrderMarketSubtypeIsIOC(t *testing.T) {
	t.Parallel()
	// MARKET orders that fully filled within one cycle may never appear
	// in open_orders/; Tracked.Price is empty and LimitPrice must be nil.
	status := client.OrderStatus{
		ID: json.Number("126"), Type: "0", Subtype: OrderSubtypeMarket, Market: "BTC/USD",
		Status: OrderStatusFinished,
		Transactions: []client.OrderTransaction{{
			TID: 3, Type: 0, Price: "60000.00", Fee: "15.00",
			CurrencyAmounts: map[string]string{"btc": "0.5", "usd": "30000.00"},
		}},
	}
	got, err := OrderStatusToPSPOrder(testCurrencies, OrderMapInput{Status: status})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Type != models.ORDER_TYPE_MARKET {
		t.Errorf("Type = %v, want MARKET", got.Type)
	}
	if got.TimeInForce != models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL {
		t.Errorf("TIF = %v, want IOC", got.TimeInForce)
	}
	if got.LimitPrice != nil {
		t.Errorf("LimitPrice must be nil for MARKET orders, got %v", got.LimitPrice)
	}
	if got.Metadata[MetadataKeyOrderSubtype] != OrderSubtypeMarket {
		t.Errorf("order_subtype metadata missing")
	}
}

func TestOrderStatusToPSPOrderInstantSubtypeIsMarketIOC(t *testing.T) {
	t.Parallel()
	status := client.OrderStatus{
		ID: json.Number("127"), Type: "0", Subtype: OrderSubtypeInstant, Market: "BTC/USD",
		Status: OrderStatusFinished,
	}
	got, err := OrderStatusToPSPOrder(testCurrencies, OrderMapInput{Status: status})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Type != models.ORDER_TYPE_MARKET {
		t.Errorf("Type = %v, want MARKET (INSTANT subtype)", got.Type)
	}
	if got.TimeInForce != models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL {
		t.Errorf("TIF = %v, want IOC", got.TimeInForce)
	}
}

func TestOrderStatusToPSPOrderMissingMarketIsError(t *testing.T) {
	t.Parallel()
	status := client.OrderStatus{ID: json.Number("128"), Type: "0", Status: OrderStatusOpen}
	_, err := OrderStatusToPSPOrder(testCurrencies, OrderMapInput{Status: status})
	if err == nil {
		t.Error("expected error when order_status omits market field")
	}
}

func TestOrderStatusToPSPOrderUnknownDirectionIsError(t *testing.T) {
	t.Parallel()
	status := client.OrderStatus{
		ID: json.Number("129"), Type: "9", Subtype: OrderSubtypeLimit, Market: "BTC/USD", Status: OrderStatusOpen,
	}
	_, err := OrderStatusToPSPOrder(testCurrencies, OrderMapInput{Status: status})
	if err == nil {
		t.Error("expected error on unknown direction value")
	}
}

func TestOrderStatusToPSPOrderEvictionFlag(t *testing.T) {
	t.Parallel()
	tracked := TrackedOrderInput{Price: "60000.00", FirstSeenAt: time.Now().UTC()}
	status := client.OrderStatus{
		ID: json.Number("125"), Type: "0", Subtype: OrderSubtypeLimit, Market: "BTC/USD",
		Status: OrderStatusOpen,
	}
	got, err := OrderStatusToPSPOrder(testCurrencies, OrderMapInput{
		Status:           status,
		Tracked:          tracked,
		RetentionExpired: true,
	})
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Metadata[MetadataKeyRetentionExpired] != "true" {
		t.Errorf("retention_expired metadata missing: %v", got.Metadata)
	}
}

func TestOrderCreatedAtFallback(t *testing.T) {
	t.Parallel()
	firstSeen := time.Date(2025, 9, 25, 14, 0, 0, 0, time.UTC)
	// Wire datetime present → wins.
	got := orderCreatedAt("2025-09-25 14:42:59.000000", firstSeen)
	want := time.Date(2025, 9, 25, 14, 42, 59, 0, time.UTC)
	if !got.Equal(want) {
		t.Errorf("wire datetime should win, got %v want %v", got, want)
	}
	// Wire datetime empty → fall back to first-sight.
	if got := orderCreatedAt("", firstSeen); !got.Equal(firstSeen) {
		t.Errorf("empty wire datetime should fall back to firstSeen, got %v", got)
	}
	// Wire datetime malformed → fall back rather than fail the cycle.
	if got := orderCreatedAt("not-a-date", firstSeen); !got.Equal(firstSeen) {
		t.Errorf("malformed wire datetime should fall back to firstSeen, got %v", got)
	}
}

func TestComputeBaseQuantityOrdered(t *testing.T) {
	t.Parallel()
	filled := big.NewInt(50_000_000)
	got, err := computeBaseQuantityOrdered(filled, "0.5", 8)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got.Cmp(big.NewInt(100_000_000)) != 0 {
		t.Errorf("got %s, want 100_000_000 (filled + remaining)", got)
	}

	// Missing amount_remaining → nil (not zero) so PSPOrder surfaces honestly.
	got, err = computeBaseQuantityOrdered(filled, "", 8)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if got != nil {
		t.Errorf("absent amount_remaining must yield nil, got %s", got)
	}

	// Bad amount_remaining → wrapped error.
	if _, err := computeBaseQuantityOrdered(filled, "not-a-number", 8); err == nil {
		t.Error("expected error on bad amount_remaining")
	}
}

func TestAggregateFillsDedupesSelfTrades(t *testing.T) {
	t.Parallel()
	fills := []client.OrderTransaction{
		{TID: 100, CurrencyAmounts: map[string]string{"btc": "0.5", "usd": "30000.00"}, Fee: "15"},
		// Duplicate tid (self-trade pair) — must be ignored to avoid
		// double-counting both legs of the same parent order.
		{TID: 100, CurrencyAmounts: map[string]string{"btc": "0.5", "usd": "30000.00"}, Fee: "15"},
		{TID: 101, CurrencyAmounts: map[string]string{"btc": "0.5", "usd": "30000.00"}, Fee: "15"},
	}
	base, quote, fee, count, err := aggregateFills(fills, "BTC", "USD", 8, 2)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	if count != 2 {
		t.Errorf("fill count = %d, want 2 (after dedupe)", count)
	}
	if base.Cmp(big.NewInt(100_000_000)) != 0 {
		t.Errorf("base = %s, want 100_000_000", base)
	}
	if quote.Cmp(big.NewInt(6_000_000)) != 0 {
		t.Errorf("quote = %s, want 6_000_000", quote)
	}
	if fee.Cmp(big.NewInt(3000)) != 0 {
		t.Errorf("fee = %s, want 3000", fee)
	}
}
