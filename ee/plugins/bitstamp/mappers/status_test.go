package mappers

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)

func TestTransactionTypeToPaymentType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		txType   string
		want     models.PaymentType
		wantOk   bool
		wantKnown bool
	}{
		{TxTypeDeposit, models.PAYMENT_TYPE_PAYIN, true, true},
		{TxTypeWithdrawal, models.PAYMENT_TYPE_PAYOUT, true, true},
		{TxTypeSubAccountTransfer, models.PAYMENT_TYPE_TRANSFER, true, true},
		{TxTypeStakingCredit, models.PAYMENT_TYPE_TRANSFER, true, true},
		{TxTypeStakingSent, models.PAYMENT_TYPE_TRANSFER, true, true},
		{TxTypeStakingReward, models.PAYMENT_TYPE_PAYIN, true, true},
		{TxTypeReferralReward, models.PAYMENT_TYPE_PAYIN, true, true},
		{TxTypeSettlementTransfer, models.PAYMENT_TYPE_TRANSFER, true, true},
		{TxTypeInterAccountTransfer, models.PAYMENT_TYPE_TRANSFER, true, true},
		// Orders + conversions are filtered up-front.
		{TxTypeMarketTrade, 0, false, true},
		{TxTypeBuySell, 0, false, true},
		// Unknown code: OTHER + ok=true so the row is emitted with a
		// Warn log against tx.id (orchestrator concern).
		{"999", models.PAYMENT_TYPE_OTHER, true, false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.txType, func(t *testing.T) {
			t.Parallel()
			got, ok := TransactionTypeToPaymentType(tc.txType)
			if ok != tc.wantOk || got != tc.want {
				t.Errorf("got (%v, %v), want (%v, %v)", got, ok, tc.want, tc.wantOk)
			}
			if known := IsKnownTransactionType(tc.txType); known != tc.wantKnown {
				t.Errorf("IsKnownTransactionType(%q) = %v, want %v", tc.txType, known, tc.wantKnown)
			}
		})
	}
}

func TestOrderStatusToPSPStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name      string
		raw       string
		fillCount int
		want      models.OrderStatus
	}{
		{"in queue", OrderStatusInQueue, 0, models.ORDER_STATUS_PENDING},
		{"open, no fills", OrderStatusOpen, 0, models.ORDER_STATUS_OPEN},
		{"open, partial fills", OrderStatusOpen, 1, models.ORDER_STATUS_PARTIALLY_FILLED},
		{"finished", OrderStatusFinished, 3, models.ORDER_STATUS_FILLED},
		{"canceled", OrderStatusCanceled, 1, models.ORDER_STATUS_CANCELLED},
		{"cancel pending", OrderStatusCancelPending, 0, models.ORDER_STATUS_CANCELLED},
		{"unknown defaults to open + Warn", "Some New Bitstamp State", 0, models.ORDER_STATUS_OPEN},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := OrderStatusToPSPStatus(tc.raw, tc.fillCount); got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestNoBitstampExpiredStatus(t *testing.T) {
	t.Parallel()
	// Bitstamp does not emit "Expired" — the connector must never
	// coerce an unknown status to ORDER_STATUS_EXPIRED.
	if OrderStatusToPSPStatus("Expired", 0) == models.ORDER_STATUS_EXPIRED {
		t.Error("ORDER_STATUS_EXPIRED must not be reachable from any Bitstamp status string")
	}
	if IsKnownOrderStatus("Expired") {
		t.Error("Expired should not be in the documented Bitstamp set")
	}
}

func TestOrderTypeDirection(t *testing.T) {
	t.Parallel()
	if OrderTypeStringToDirection("0") != models.ORDER_DIRECTION_BUY {
		t.Error("0 should map to BUY")
	}
	if OrderTypeStringToDirection("1") != models.ORDER_DIRECTION_SELL {
		t.Error("1 should map to SELL")
	}
	if OrderTypeStringToDirection("99") != models.ORDER_DIRECTION_UNKNOWN {
		t.Error("unknown should map to UNKNOWN")
	}
	if OrderTypeIntToDirection(0) != models.ORDER_DIRECTION_BUY {
		t.Error("int 0 should map to BUY")
	}
	if OrderTypeIntToDirection(1) != models.ORDER_DIRECTION_SELL {
		t.Error("int 1 should map to SELL")
	}
}
