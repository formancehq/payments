package mappers

import (
	"testing"

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


