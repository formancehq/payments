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

	deref := func(p *string) string {
		if p == nil {
			return "<nil>"
		}
		return *p
	}

	// BUY: src=quote(USD), dst=base(BTC). accountReference=USD → src non-nil, dst nil.
	src, dst := accountReferencesForDirection(models.ORDER_DIRECTION_BUY, "BTC", "USD", "USD")
	if src == nil || *src != "USD" {
		t.Errorf("BUY/USD: src = %s, want USD", deref(src))
	}
	if dst != nil {
		t.Errorf("BUY/USD: dst = %s, want nil", deref(dst))
	}

	// BUY: accountReference=BTC → src nil, dst non-nil.
	src, dst = accountReferencesForDirection(models.ORDER_DIRECTION_BUY, "BTC", "USD", "BTC")
	if src != nil {
		t.Errorf("BUY/BTC: src = %s, want nil", deref(src))
	}
	if dst == nil || *dst != "BTC" {
		t.Errorf("BUY/BTC: dst = %s, want BTC", deref(dst))
	}

	// SELL: src=base(BTC), dst=quote(USD). accountReference=BTC → src non-nil, dst nil.
	src, dst = accountReferencesForDirection(models.ORDER_DIRECTION_SELL, "BTC", "USD", "BTC")
	if src == nil || *src != "BTC" {
		t.Errorf("SELL/BTC: src = %s, want BTC", deref(src))
	}
	if dst != nil {
		t.Errorf("SELL/BTC: dst = %s, want nil", deref(dst))
	}

	// SELL: accountReference=USD → src nil, dst non-nil.
	src, dst = accountReferencesForDirection(models.ORDER_DIRECTION_SELL, "BTC", "USD", "USD")
	if src != nil {
		t.Errorf("SELL/USD: src = %s, want nil", deref(src))
	}
	if dst == nil || *dst != "USD" {
		t.Errorf("SELL/USD: dst = %s, want USD", deref(dst))
	}

	// accountReference matches neither → both nil.
	src, dst = accountReferencesForDirection(models.ORDER_DIRECTION_BUY, "BTC", "USD", "ETH")
	if src != nil || dst != nil {
		t.Errorf("BUY/ETH: got (%s, %s), want (nil, nil)", deref(src), deref(dst))
	}
}


