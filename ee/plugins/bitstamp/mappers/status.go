package mappers

import (
	"github.com/formancehq/payments/internal/models"
)

// Bitstamp user_transactions.type values. 33 + 36 are undocumented
// but observed in production. See MAPPINGS §4.3.
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

// TransactionTypeToPaymentType returns ok=false for trade/conversion
// rows (handled by other capabilities); unknown codes fall back to
// PAYMENT_TYPE_OTHER + ok=true and are logged at Info.
func TransactionTypeToPaymentType(txType string) (paymentType models.PaymentType, ok bool) {
	if IsOrderOrConversion(txType) {
		return 0, false
	}
	if t, found := transactionTypeMap[txType]; found {
		return t, true
	}
	return models.PAYMENT_TYPE_OTHER, true
}

// IsOrderOrConversion: type 2 = trade fill (orders), type 36 = instant
// buy/sell (conversions). Payments mapper rejects both.
func IsOrderOrConversion(txType string) bool {
	return txType == TxTypeMarketTrade || txType == TxTypeBuySell
}

// IsTransferType: types 14 / 33 / 35 — two-legged movements emitted
// as sign-based PAYOUT / PAYIN per MAPPINGS §4.3 cross-account.
func IsTransferType(txType string) bool {
	switch txType {
	case TxTypeSubAccountTransfer, TxTypeSettlementTransfer, TxTypeInterAccountTransfer:
		return true
	}
	return false
}

func IsKnownTransactionType(txType string) bool {
	if IsOrderOrConversion(txType) {
		return true
	}
	_, ok := transactionTypeMap[txType]
	return ok
}

// Bitstamp order_status.status values. Expired does not appear in
// the Bitstamp enum — unknown values default to OPEN + Warn rather
// than coercing to terminal.
const (
	OrderStatusInQueue       = "In Queue"
	OrderStatusOpen          = "Open"
	OrderStatusFinished      = "Finished"
	OrderStatusCanceled      = "Canceled"
	OrderStatusCancelPending = "Cancel pending"
)

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

// OrderTypeStringToDirection maps open_orders.type to OrderDirection.
// Unknown values fail PSPOrder validation downstream (loud, not silent).
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
// transactions[] fills.
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

const (
	OrderSubtypeLimit     = "LIMIT"
	OrderSubtypeMarket    = "MARKET"
	OrderSubtypeInstant   = "INSTANT"
	OrderSubtypeStopLimit = "STOP_LIMIT"
)

// OrderSubtypeToType maps order_status.subtype to OrderType. Formance
// has no INSTANT constant — INSTANT collapses into MARKET with the
// wire subtype preserved under MetadataKeyOrderSubtype.
func OrderSubtypeToType(subtype string) models.OrderType {
	switch subtype {
	case OrderSubtypeLimit:
		return models.ORDER_TYPE_LIMIT
	case OrderSubtypeMarket, OrderSubtypeInstant:
		return models.ORDER_TYPE_MARKET
	case OrderSubtypeStopLimit:
		return models.ORDER_TYPE_STOP_LIMIT
	default:
		return models.ORDER_TYPE_UNKNOWN
	}
}

// OrderSubtypeToTIF infers TimeInForce from subtype — Bitstamp does
// not surface explicit TIF flags. LIMIT/STOP_LIMIT rest on the book
// (GTC); MARKET/INSTANT execute or fail at request time (IOC).
func OrderSubtypeToTIF(subtype string) models.TimeInForce {
	switch subtype {
	case OrderSubtypeMarket, OrderSubtypeInstant:
		return models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL
	default:
		return models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
	}
}

const (
	WithdrawalRequestTypeSEPA              = 0
	WithdrawalRequestTypeInternationalWire = 1
	WithdrawalRequestTypeARDI              = 2
	WithdrawalRequestTypeInternationalBIC  = 3
	WithdrawalRequestTypeCrypto            = 4
)

// WithdrawalRequestTypeToScheme — Formance has no SWIFT / ARDI /
// crypto-withdrawal constants today; non-SEPA preserves the wire
// integer in metadata under MetadataKeyType for downstream
// disambiguation.
func WithdrawalRequestTypeToScheme(t int) models.PaymentScheme {
	switch t {
	case WithdrawalRequestTypeSEPA:
		return models.PAYMENT_SCHEME_SEPA_CREDIT
	case WithdrawalRequestTypeInternationalWire,
		WithdrawalRequestTypeARDI,
		WithdrawalRequestTypeInternationalBIC,
		WithdrawalRequestTypeCrypto:
		return models.PAYMENT_SCHEME_OTHER
	default:
		return models.PAYMENT_SCHEME_UNKNOWN
	}
}

const (
	WithdrawalRequestStatusOpen       = 0
	WithdrawalRequestStatusInProgress = 1
	WithdrawalRequestStatusFinished   = 2
	WithdrawalRequestStatusCanceled   = 3
	WithdrawalRequestStatusFailed     = 4
)

func WithdrawalRequestStatusToPaymentStatus(s int) models.PaymentStatus {
	switch s {
	case WithdrawalRequestStatusOpen, WithdrawalRequestStatusInProgress:
		return models.PAYMENT_STATUS_PENDING
	case WithdrawalRequestStatusFinished:
		return models.PAYMENT_STATUS_SUCCEEDED
	case WithdrawalRequestStatusCanceled:
		return models.PAYMENT_STATUS_CANCELLED
	case WithdrawalRequestStatusFailed:
		return models.PAYMENT_STATUS_FAILED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}

// Withdrawals + ripple IOUs have no status field — both are treated
// as SUCCEEDED by the orchestrator (endpoint surfaces only processed
// rows).
const (
	CryptoDepositStatusPending   = "PENDING"
	CryptoDepositStatusCompleted = "COMPLETED"
)

func CryptoDepositStatusToPaymentStatus(s string) models.PaymentStatus {
	switch s {
	case CryptoDepositStatusPending:
		return models.PAYMENT_STATUS_PENDING
	case CryptoDepositStatusCompleted:
		return models.PAYMENT_STATUS_SUCCEEDED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}
