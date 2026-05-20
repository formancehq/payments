package mappers

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)

func TestOrderSubtypeToType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want models.OrderType
	}{
		{OrderSubtypeLimit, models.ORDER_TYPE_LIMIT},
		{OrderSubtypeMarket, models.ORDER_TYPE_MARKET},
		{OrderSubtypeInstant, models.ORDER_TYPE_MARKET},
		{OrderSubtypeStopLimit, models.ORDER_TYPE_STOP_LIMIT},
		{"", models.ORDER_TYPE_UNKNOWN},
		{"FUTURE_SUBTYPE", models.ORDER_TYPE_UNKNOWN},
	}
	for _, tc := range cases {
		if got := OrderSubtypeToType(tc.in); got != tc.want {
			t.Errorf("OrderSubtypeToType(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestOrderSubtypeToTIF(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want models.TimeInForce
	}{
		{OrderSubtypeMarket, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL},
		{OrderSubtypeInstant, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL},
		{OrderSubtypeLimit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeStopLimit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{"", models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
	}
	for _, tc := range cases {
		if got := OrderSubtypeToTIF(tc.in); got != tc.want {
			t.Errorf("OrderSubtypeToTIF(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestWithdrawalRequestTypeToScheme(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   int
		want models.PaymentScheme
	}{
		{WithdrawalRequestTypeSEPA, models.PAYMENT_SCHEME_SEPA_CREDIT},
		{WithdrawalRequestTypeInternationalWire, models.PAYMENT_SCHEME_OTHER},
		{WithdrawalRequestTypeARDI, models.PAYMENT_SCHEME_OTHER},
		{WithdrawalRequestTypeInternationalBIC, models.PAYMENT_SCHEME_OTHER},
		{WithdrawalRequestTypeCrypto, models.PAYMENT_SCHEME_OTHER},
		{99, models.PAYMENT_SCHEME_UNKNOWN},
		{-1, models.PAYMENT_SCHEME_UNKNOWN},
	}
	for _, tc := range cases {
		if got := WithdrawalRequestTypeToScheme(tc.in); got != tc.want {
			t.Errorf("WithdrawalRequestTypeToScheme(%d) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestWithdrawalRequestStatusToPaymentStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   int
		want models.PaymentStatus
	}{
		{WithdrawalRequestStatusOpen, models.PAYMENT_STATUS_PENDING},
		{WithdrawalRequestStatusInProgress, models.PAYMENT_STATUS_PENDING},
		{WithdrawalRequestStatusFinished, models.PAYMENT_STATUS_SUCCEEDED},
		{WithdrawalRequestStatusCanceled, models.PAYMENT_STATUS_CANCELLED},
		{WithdrawalRequestStatusFailed, models.PAYMENT_STATUS_FAILED},
		{99, models.PAYMENT_STATUS_UNKNOWN},
	}
	for _, tc := range cases {
		if got := WithdrawalRequestStatusToPaymentStatus(tc.in); got != tc.want {
			t.Errorf("WithdrawalRequestStatusToPaymentStatus(%d) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

func TestCryptoDepositStatusToPaymentStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want models.PaymentStatus
	}{
		{"PENDING", models.PAYMENT_STATUS_PENDING},
		{"COMPLETED", models.PAYMENT_STATUS_SUCCEEDED},
		{"", models.PAYMENT_STATUS_UNKNOWN},
		{"FUTURE", models.PAYMENT_STATUS_UNKNOWN},
	}
	for _, tc := range cases {
		if got := CryptoDepositStatusToPaymentStatus(tc.in); got != tc.want {
			t.Errorf("CryptoDepositStatusToPaymentStatus(%q) = %v, want %v", tc.in, got, tc.want)
		}
	}
}

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
