package mappers

import (
	"github.com/formancehq/payments/internal/models"
)

// Bitstamp user_transactions.type values.
// 33 (settlement_transfer) and 36 (buy/sell instant) are undocumented
// but observed in production — see MAPPINGS.md §4.1.
const (
	TxTypeDeposit              = "0"
	TxTypeWithdrawal           = "1"
	TxTypeMarketTrade          = "2"
	TxTypeSubAccountTransfer   = "14"
	TxTypeStakingCredit        = "25"
	TxTypeStakingSent          = "26"
	TxTypeStakingReward        = "27"
	TxTypeReferralReward       = "32"
	TxTypeSettlementTransfer   = "33"
	TxTypeInterAccountTransfer = "35"
	TxTypeBuySell              = "36"
)

// transactionTypeMap is the table-driven source of truth for the
// user_transactions.type → PaymentType mapping. Adding a new code is
// a one-line diff; the orchestrator already logs Warn on any code
// missing from this table.
var transactionTypeMap = map[string]models.PaymentType{
	TxTypeDeposit:              models.PAYMENT_TYPE_PAYIN,
	TxTypeWithdrawal:           models.PAYMENT_TYPE_PAYOUT,
	TxTypeSubAccountTransfer:   models.PAYMENT_TYPE_TRANSFER,
	TxTypeStakingCredit:        models.PAYMENT_TYPE_TRANSFER,
	TxTypeStakingSent:          models.PAYMENT_TYPE_TRANSFER,
	TxTypeStakingReward:        models.PAYMENT_TYPE_PAYIN,
	TxTypeReferralReward:       models.PAYMENT_TYPE_PAYIN,
	TxTypeSettlementTransfer:   models.PAYMENT_TYPE_TRANSFER,
	TxTypeInterAccountTransfer: models.PAYMENT_TYPE_TRANSFER,
}

// TransactionTypeToPaymentType returns the PSPPayment type for a
// Bitstamp user_transactions row. Trade and instant-buy/sell rows
// (handled by orders + conversions respectively) return ok=false so
// the payments orchestrator skips them up-front; truly unknown codes
// return PAYMENT_TYPE_OTHER + ok=true with the orchestrator logging
// Warn against the row ID.
func TransactionTypeToPaymentType(txType string) (paymentType models.PaymentType, ok bool) {
	if IsOrderOrConversion(txType) {
		return 0, false
	}
	if t, found := transactionTypeMap[txType]; found {
		return t, true
	}
	return models.PAYMENT_TYPE_OTHER, true
}

// IsOrderOrConversion reports whether a user_transactions row is a
// trade fill (type 2 — feeds orders via order_status.transactions[])
// or an instant buy/sell (type 36 — feeds conversions). Either way
// the payments mapper rejects it.
func IsOrderOrConversion(txType string) bool {
	return txType == TxTypeMarketTrade || txType == TxTypeBuySell
}

// IsKnownTransactionType reports whether a code is in the documented
// (or observed-and-codified) set. Used by the orchestrator to log
// Warn on previously-unseen codes without blocking the row.
func IsKnownTransactionType(txType string) bool {
	if IsOrderOrConversion(txType) {
		return true
	}
	_, ok := transactionTypeMap[txType]
	return ok
}

// Bitstamp order_status.status values, from ccxt's parseOrderStatus
// (cross-checked against live docs). "Expired" does NOT appear in
// the Bitstamp enum — treat any unknown value as OPEN + Warn rather
// than coercing to terminal.
const (
	OrderStatusInQueue       = "In Queue"
	OrderStatusOpen          = "Open"
	OrderStatusFinished      = "Finished"
	OrderStatusCanceled      = "Canceled"
	OrderStatusCancelPending = "Cancel pending"
)

// OrderStatusToPSPStatus derives the Formance order status from the
// raw Bitstamp status plus the number of fills observed so far.
//
//   - "Open" + 0 fills    → OPEN
//   - "Open" + N fills    → PARTIALLY_FILLED
//   - "Finished"          → FILLED  (Bitstamp closes the order)
//   - "Canceled"          → CANCELLED
//   - "Cancel pending"    → CANCELLED (Formance has no transient
//                                     cancelling state)
//   - "In Queue"          → PENDING
//   - anything else       → OPEN  (the safer side of the cycle;
//                                  orchestrator logs Warn)
func OrderStatusToPSPStatus(raw string, fillCount int) models.OrderStatus {
	switch raw {
	case OrderStatusInQueue:
		return models.ORDER_STATUS_PENDING
	case OrderStatusOpen:
		if fillCount > 0 {
			return models.ORDER_STATUS_PARTIALLY_FILLED
		}
		return models.ORDER_STATUS_OPEN
	case OrderStatusFinished:
		return models.ORDER_STATUS_FILLED
	case OrderStatusCanceled, OrderStatusCancelPending:
		return models.ORDER_STATUS_CANCELLED
	default:
		return models.ORDER_STATUS_OPEN
	}
}

// IsKnownOrderStatus reports whether a raw Bitstamp order status is
// in the documented set. Used by the orchestrator to log Warn on
// unknown values without failing the cycle.
func IsKnownOrderStatus(raw string) bool {
	switch raw {
	case OrderStatusInQueue, OrderStatusOpen, OrderStatusFinished,
		OrderStatusCanceled, OrderStatusCancelPending:
		return true
	}
	return false
}

// OrderTypeStringToDirection maps Bitstamp's open_orders.type string
// ("0" = buy, "1" = sell) to a Formance OrderDirection. Returns
// ORDER_DIRECTION_UNKNOWN on any other value; the orchestrator will
// fail validation on the resulting PSPOrder so unknown directions
// are loud, not silent.
func OrderTypeStringToDirection(t string) models.OrderDirection {
	switch t {
	case "0":
		return models.ORDER_DIRECTION_BUY
	case "1":
		return models.ORDER_DIRECTION_SELL
	default:
		return models.ORDER_DIRECTION_UNKNOWN
	}
}

// OrderTypeIntToDirection is the int variant used by order_status's
// transactions[] fills (type 2 = sell-like, 0 = buy-like). Used only
// for fill aggregation diagnostics today; the parent order's
// direction comes from the first-sight open_orders capture.
func OrderTypeIntToDirection(t int) models.OrderDirection {
	switch t {
	case 0:
		return models.ORDER_DIRECTION_BUY
	case 1:
		return models.ORDER_DIRECTION_SELL
	default:
		return models.ORDER_DIRECTION_UNKNOWN
	}
}
