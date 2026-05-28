package mappers

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)


func TestTransactionTypeToPaymentType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		txType    string
		want      models.PaymentType
		wantOk    bool
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

func TestOrderSubtypeToType(t *testing.T) {
	t.Parallel()
	cases := []struct {
		subtype int
		want    models.OrderType
	}{
		{OrderSubtypeLimit, models.ORDER_TYPE_LIMIT},
		{OrderSubtypeInstant, models.ORDER_TYPE_MARKET},
		{OrderSubtypeMarket, models.ORDER_TYPE_MARKET},
		{OrderSubtypeDaily, models.ORDER_TYPE_LIMIT},
		{OrderSubtypeIOC, models.ORDER_TYPE_LIMIT},
		{OrderSubtypeMOC, models.ORDER_TYPE_LIMIT_MAKER},
		{OrderSubtypeFOK, models.ORDER_TYPE_LIMIT},
		{OrderSubtypeCashSell, models.ORDER_TYPE_MARKET},
		{OrderSubtypeGTD, models.ORDER_TYPE_LIMIT},
		{OrderSubtypeStopLoss, models.ORDER_TYPE_STOP},
		{OrderSubtypeTakeProfit, models.ORDER_TYPE_TAKE_PROFIT},
		{OrderSubtypeStopLossLimit, models.ORDER_TYPE_STOP_LIMIT},
		{OrderSubtypeTakeProfitLimit, models.ORDER_TYPE_TAKE_PROFIT_LIMIT},
		{OrderSubtypeTrailingStopLoss, models.ORDER_TYPE_TRAILING_STOP},
		{OrderSubtypeTrailingTakeProfit, models.ORDER_TYPE_TAKE_PROFIT},
		{OrderSubtypeStopLossLimit2, models.ORDER_TYPE_STOP_LIMIT},
		{OrderSubtypeTrailingTakeProfitLimit, models.ORDER_TYPE_TAKE_PROFIT_LIMIT},
		{99, models.ORDER_TYPE_LIMIT},
	}
	for _, tc := range cases {
		if got := OrderSubtypeToType(tc.subtype); got != tc.want {
			t.Errorf("OrderSubtypeToType(%d) = %v, want %v", tc.subtype, got, tc.want)
		}
	}
}

func TestOrderSubtypeToTIF(t *testing.T) {
	t.Parallel()
	cases := []struct {
		subtype int
		want    models.TimeInForce
	}{
		{OrderSubtypeLimit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeInstant, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL},
		{OrderSubtypeMarket, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL},
		{OrderSubtypeDaily, models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME},
		{OrderSubtypeIOC, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL},
		{OrderSubtypeMOC, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeFOK, models.TIME_IN_FORCE_FILL_OR_KILL},
		{OrderSubtypeCashSell, models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL},
		{OrderSubtypeGTD, models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME},
		{OrderSubtypeStopLoss, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeTakeProfit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeStopLossLimit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeTakeProfitLimit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeTrailingStopLoss, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeTrailingTakeProfit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeStopLossLimit2, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{OrderSubtypeTrailingTakeProfitLimit, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
		{99, models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED},
	}
	for _, tc := range cases {
		if got := OrderSubtypeToTIF(tc.subtype); got != tc.want {
			t.Errorf("OrderSubtypeToTIF(%d) = %v, want %v", tc.subtype, got, tc.want)
		}
	}
}

func TestOrderTypeDirection(t *testing.T) {
	t.Parallel()
	if OrderTypeIntToDirection(0) != models.ORDER_DIRECTION_BUY {
		t.Error("int 0 should map to BUY")
	}
	if OrderTypeIntToDirection(1) != models.ORDER_DIRECTION_SELL {
		t.Error("int 1 should map to SELL")
	}
	if OrderTypeIntToDirection(99) != models.ORDER_DIRECTION_UNKNOWN {
		t.Error("unknown should map to UNKNOWN")
	}
}
