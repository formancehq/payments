package mappers

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)

func TestClassifyLedgerType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in         string
		wantKind   LedgerTypeKind
		wantType   models.PaymentType
		signDriven bool
	}{
		{"deposit", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
		{"withdrawal", LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT, false},
		{"transfer", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, true},
		{"staking", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
		{"reward", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
		{"adjustment", LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
		{"trade", LedgerKindOrder, models.PAYMENT_TYPE_OTHER, false},
		{"conversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
		{"sale", LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
		{"marginconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
		{"derivativesflexconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
		{"derivativestaxconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
		{"derivativesconversioncredit", LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
		{"collateralconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER, false},
		// Newer Kraken ledger types (Codex review).
		{"spend", LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT, false},
		{"receive", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
		{"custodytransfer", LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
		{"nftrebate", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
		{"nft_rebate", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN, false},
		{"future_unknown_type", LedgerKindPayment, models.PAYMENT_TYPE_OTHER, false},
	}
	for _, c := range cases {
		gotKind, gotType, gotSign := ClassifyLedgerType(c.in)
		if gotKind != c.wantKind || gotType != c.wantType || gotSign != c.signDriven {
			t.Errorf("ClassifyLedgerType(%q) = (%v,%v,%v), want (%v,%v,%v)",
				c.in, gotKind, gotType, gotSign, c.wantKind, c.wantType, c.signDriven)
		}
	}
}

func TestMapOrderType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in        string
		want      models.OrderType
		wantKnown bool
	}{
		{"market", models.ORDER_TYPE_MARKET, true},
		{"limit", models.ORDER_TYPE_LIMIT, true},
		{"stop-loss-limit", models.ORDER_TYPE_STOP_LIMIT, true},
		{"trailing-stop", models.ORDER_TYPE_TRAILING_STOP, true},
		{"take-profit", models.ORDER_TYPE_TAKE_PROFIT, true},
		{"iceberg", models.ORDER_TYPE_MARKET, true},
		{"settle-position", models.ORDER_TYPE_MARKET, true},
		{"some-new-type", models.ORDER_TYPE_UNKNOWN, false},
	}
	for _, c := range cases {
		got, known := MapOrderType(c.in)
		if got != c.want || known != c.wantKnown {
			t.Errorf("MapOrderType(%q) = (%v,%v), want (%v,%v)", c.in, got, known, c.want, c.wantKnown)
		}
	}
}
