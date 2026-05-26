package mappers

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
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
			b, q, err := SplitCurrencyPair(tc.in)
			if (err != nil) != tc.err {
				t.Fatalf("err: %v wantErr=%v", err, tc.err)
			}
			if !tc.err && (b != tc.base || q != tc.quote) {
				t.Errorf("got (%s, %s), want (%s, %s)", b, q, tc.base, tc.quote)
			}
		})
	}
}

// TestAccountOrderFilterAndReference verifies that AccountOrderDataEventToPSPOrder
// filters orders whose source currency doesn't match accountReference and that
// only SourceAccountReference is set (DestinationAccountReference is always nil).
func TestAccountOrderFilterAndReference(t *testing.T) {
	t.Parallel()

	currencies := map[string]int{"BTC": 8, "USD": 2}

	makeEvent := func(orderType int) client.AccountOrderDataEvent {
		return client.AccountOrderDataEvent{
			Event:   "order_created",
			EventID: "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4",
			Data: client.AccountOrderDataItem{
				IDStr:          "1000",
				OrderType:      orderType,
				Amount:         json.Number("0.001"),
				AmountAtCreate: "0.001",
				AmountTraded:   "0",
				AmountStr:      "0.001",
				PriceStr:       "50000",
				Microtimestamp: "1779709892000000",
			},
		}
	}

	// BUY (orderType=0): source is quote (USD). accountReference=USD → keep.
	order, err := AccountOrderDataEventToPSPOrder(currencies, "USD", "BTC/USD", makeEvent(0))
	if err != nil {
		t.Fatalf("BUY/USD: unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("BUY/USD: expected order, got nil")
	}
	if order.SourceAccountReference == nil || *order.SourceAccountReference != "USD" {
		t.Errorf("BUY/USD: SourceAccountReference = %v, want &USD", order.SourceAccountReference)
	}
	if order.DestinationAccountReference != nil {
		t.Errorf("BUY/USD: DestinationAccountReference = %v, want nil", order.DestinationAccountReference)
	}

	// BUY: accountReference=BTC (the base, not the source) → filter out.
	order, err = AccountOrderDataEventToPSPOrder(currencies, "BTC", "BTC/USD", makeEvent(0))
	if err != nil {
		t.Fatalf("BUY/BTC: unexpected error: %v", err)
	}
	if order != nil {
		t.Errorf("BUY/BTC: expected nil (filtered), got order %+v", order)
	}

	// SELL (orderType=1): source is base (BTC). accountReference=BTC → keep.
	order, err = AccountOrderDataEventToPSPOrder(currencies, "BTC", "BTC/USD", makeEvent(1))
	if err != nil {
		t.Fatalf("SELL/BTC: unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("SELL/BTC: expected order, got nil")
	}
	if order.SourceAccountReference == nil || *order.SourceAccountReference != "BTC" {
		t.Errorf("SELL/BTC: SourceAccountReference = %v, want &BTC", order.SourceAccountReference)
	}
	if order.DestinationAccountReference != nil {
		t.Errorf("SELL/BTC: DestinationAccountReference = %v, want nil", order.DestinationAccountReference)
	}

	// SELL: accountReference=USD (the quote, not the source) → filter out.
	order, err = AccountOrderDataEventToPSPOrder(currencies, "USD", "BTC/USD", makeEvent(1))
	if err != nil {
		t.Fatalf("SELL/USD: unexpected error: %v", err)
	}
	if order != nil {
		t.Errorf("SELL/USD: expected nil (filtered), got order %+v", order)
	}
}


