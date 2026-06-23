package mappers

import (
	"encoding/json"
	"testing"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/pkg/domain/models"
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

// TestAccountOrderAccountReference verifies that AccountOrderDataEventToPSPOrder
// always returns an order and routes accountReference to SourceAccountReference
// when it is the spender, or DestinationAccountReference otherwise.
func TestAccountOrderAccountReference(t *testing.T) {
	t.Parallel()

	currencies := map[string]int{"BTC": 8, "USD": 2}

	makeEvent := func(orderType int) client.AccountOrderDataEvent {
		return client.AccountOrderDataEvent{
			Event:   "order_created",
			EventID: "a1b2c3d4-e5f6-a1b2-c3d4-e5f6a1b2c3d4",
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

	// BUY spends quote (USD): accountReference=USD → srcAccount=&USD, dstAccount=nil.
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

	// BUY spends quote (USD): accountReference=BTC (the destination) → srcAccount=nil, dstAccount=&BTC.
	order, err = AccountOrderDataEventToPSPOrder(currencies, "BTC", "BTC/USD", makeEvent(0))
	if err != nil {
		t.Fatalf("BUY/BTC: unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("BUY/BTC: expected order, got nil")
	}
	if order.SourceAccountReference != nil {
		t.Errorf("BUY/BTC: SourceAccountReference = %v, want nil", order.SourceAccountReference)
	}
	if order.DestinationAccountReference == nil || *order.DestinationAccountReference != "BTC" {
		t.Errorf("BUY/BTC: DestinationAccountReference = %v, want &BTC", order.DestinationAccountReference)
	}

	// SELL spends base (BTC): accountReference=BTC → srcAccount=&BTC, dstAccount=nil.
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

	// SELL spends base (BTC): accountReference=USD (the destination) → srcAccount=nil, dstAccount=&USD.
	order, err = AccountOrderDataEventToPSPOrder(currencies, "USD", "BTC/USD", makeEvent(1))
	if err != nil {
		t.Fatalf("SELL/USD: unexpected error: %v", err)
	}
	if order == nil {
		t.Fatal("SELL/USD: expected order, got nil")
	}
	if order.SourceAccountReference != nil {
		t.Errorf("SELL/USD: SourceAccountReference = %v, want nil", order.SourceAccountReference)
	}
	if order.DestinationAccountReference == nil || *order.DestinationAccountReference != "USD" {
		t.Errorf("SELL/USD: DestinationAccountReference = %v, want &USD", order.DestinationAccountReference)
	}
}

func TestResolveOrderAccounts(t *testing.T) {
	t.Parallel()

	ptrEq := func(p *string, want string) bool { return p != nil && *p == want }

	cases := []struct {
		name             string
		direction        models.OrderDirection
		base, quote, ref string
		wantSrcAccount   string // "" means expect nil
		wantDstAccount   string // "" means expect nil
		wantSrcCurrency  string
		wantDstCurrency  string
	}{
		{
			name: "BUY ref=quote (spender)",
			direction: models.ORDER_DIRECTION_BUY, base: "BTC", quote: "USD", ref: "USD",
			wantSrcAccount: "USD", wantSrcCurrency: "USD", wantDstCurrency: "BTC",
		},
		{
			name: "BUY ref=base (receiver)",
			direction: models.ORDER_DIRECTION_BUY, base: "BTC", quote: "USD", ref: "BTC",
			wantDstAccount: "BTC", wantSrcCurrency: "USD", wantDstCurrency: "BTC",
		},
		{
			name: "SELL ref=base (spender)",
			direction: models.ORDER_DIRECTION_SELL, base: "BTC", quote: "USD", ref: "BTC",
			wantSrcAccount: "BTC", wantSrcCurrency: "BTC", wantDstCurrency: "USD",
		},
		{
			name: "SELL ref=quote (receiver)",
			direction: models.ORDER_DIRECTION_SELL, base: "BTC", quote: "USD", ref: "USD",
			wantDstAccount: "USD", wantSrcCurrency: "BTC", wantDstCurrency: "USD",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srcAccount, dstAccount, srcCurrency, dstCurrency := ResolveOrderAccounts(tc.direction, tc.base, tc.quote, tc.ref)

			if tc.wantSrcAccount != "" {
				if !ptrEq(srcAccount, tc.wantSrcAccount) {
					t.Errorf("srcAccount = %v, want &%s", srcAccount, tc.wantSrcAccount)
				}
			} else if srcAccount != nil {
				t.Errorf("srcAccount = &%s, want nil", *srcAccount)
			}

			if tc.wantDstAccount != "" {
				if !ptrEq(dstAccount, tc.wantDstAccount) {
					t.Errorf("dstAccount = %v, want &%s", dstAccount, tc.wantDstAccount)
				}
			} else if dstAccount != nil {
				t.Errorf("dstAccount = &%s, want nil", *dstAccount)
			}

			if srcCurrency != tc.wantSrcCurrency {
				t.Errorf("srcCurrency = %q, want %q", srcCurrency, tc.wantSrcCurrency)
			}
			if dstCurrency != tc.wantDstCurrency {
				t.Errorf("dstCurrency = %q, want %q", dstCurrency, tc.wantDstCurrency)
			}
		})
	}
}
