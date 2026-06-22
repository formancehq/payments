package mappers

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)

func TestClassifyLedgerType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in       string
		wantKind LedgerTypeKind
		wantType models.PaymentType
	}{
		{"deposit", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
		{"withdrawal", LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT},
		// transfer / custodytransfer are internal movements -> TRANSFER.
		{"transfer", LedgerKindPayment, models.PAYMENT_TYPE_TRANSFER},
		{"custodytransfer", LedgerKindPayment, models.PAYMENT_TYPE_TRANSFER},
		{"staking", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
		{"reward", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
		{"adjustment", LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
		{"trade", LedgerKindOrder, models.PAYMENT_TYPE_OTHER},
		{"conversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
		{"sale", LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
		{"marginconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
		{"derivativesflexconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
		{"derivativestaxconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
		{"derivativesconversioncredit", LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
		{"collateralconversion", LedgerKindConversion, models.PAYMENT_TYPE_OTHER},
		// Newer Kraken ledger types (Codex review).
		{"spend", LedgerKindPayment, models.PAYMENT_TYPE_PAYOUT},
		{"receive", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
		{"nftrebate", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
		{"nft_rebate", LedgerKindPayment, models.PAYMENT_TYPE_PAYIN},
		{"future_unknown_type", LedgerKindPayment, models.PAYMENT_TYPE_OTHER},
	}
	for _, c := range cases {
		gotKind, gotType := ClassifyLedgerType(c.in)
		if gotKind != c.wantKind || gotType != c.wantType {
			t.Errorf("ClassifyLedgerType(%q) = (%v,%v), want (%v,%v)",
				c.in, gotKind, gotType, c.wantKind, c.wantType)
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
