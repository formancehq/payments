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

// OrderTypeIntToDirection maps account_order_data.order_type to OrderDirection.
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

// AccountOrderEventToStatus derives a PSPOrder status from the
// account_order_data event type and the order's remaining + traded amounts.
func AccountOrderEventToStatus(event, amountStr, amountTraded string) models.OrderStatus {
	isFullyConsumed := IsZeroAmount(amountStr)
	hasFills := !IsZeroAmount(amountTraded)
	switch event {
	case "order_created":
		if hasFills {
			return models.ORDER_STATUS_PARTIALLY_FILLED
		}
		return models.ORDER_STATUS_OPEN
	case "order_deleted":
		if isFullyConsumed && hasFills {
			return models.ORDER_STATUS_FILLED
		}
		return models.ORDER_STATUS_CANCELLED
	default:
		if hasFills {
			return models.ORDER_STATUS_PARTIALLY_FILLED
		}
		return models.ORDER_STATUS_OPEN
	}
}

// Order subtype (0 - limit; 1 - instant; 2 - market; 3 - daily; 4 - IOC; 5 - MOC; 6 - FOK; 7 - CASH SELL; 8 - GTD; 20 - stop loss; 21 - take profit; 22 - stop loss limit; 23 - take profit limit; 24 - trailing stop loss; 25 - trailing take profit; 26 - stop loss limit; 27 - trailing take profit limit).
const (
	OrderSubtypeLimit                   = 0
	OrderSubtypeInstant                 = 1
	OrderSubtypeMarket                  = 2
	OrderSubtypeDaily                   = 3
	OrderSubtypeIOC                     = 4
	OrderSubtypeMOC                     = 5
	OrderSubtypeFOK                     = 6
	OrderSubtypeCashSell                = 7
	OrderSubtypeGTD                     = 8
	OrderSubtypeStopLoss                = 20
	OrderSubtypeTakeProfit              = 21
	OrderSubtypeStopLossLimit           = 22
	OrderSubtypeTakeProfitLimit         = 23
	OrderSubtypeTrailingStopLoss        = 24
	OrderSubtypeTrailingTakeProfit      = 25
	OrderSubtypeStopLossLimit2          = 26
	OrderSubtypeTrailingTakeProfitLimit = 27
)

func OrderSubtypeToType(subtype int) models.OrderType {
	switch subtype {
	case OrderSubtypeLimit, OrderSubtypeDaily, OrderSubtypeIOC,
		OrderSubtypeFOK, OrderSubtypeGTD:
		return models.ORDER_TYPE_LIMIT
	case OrderSubtypeInstant, OrderSubtypeMarket, OrderSubtypeCashSell:
		return models.ORDER_TYPE_MARKET
	case OrderSubtypeMOC:
		return models.ORDER_TYPE_LIMIT_MAKER
	case OrderSubtypeStopLoss:
		return models.ORDER_TYPE_STOP
	case OrderSubtypeStopLossLimit, OrderSubtypeStopLossLimit2:
		return models.ORDER_TYPE_STOP_LIMIT
	case OrderSubtypeTakeProfit, OrderSubtypeTrailingTakeProfit:
		return models.ORDER_TYPE_TAKE_PROFIT
	case OrderSubtypeTakeProfitLimit, OrderSubtypeTrailingTakeProfitLimit:
		return models.ORDER_TYPE_TAKE_PROFIT_LIMIT
	case OrderSubtypeTrailingStopLoss:
		return models.ORDER_TYPE_TRAILING_STOP
	default:
		return models.ORDER_TYPE_LIMIT
	}
}

func OrderSubtypeToTIF(subtype int) models.TimeInForce {
	switch subtype {
	case OrderSubtypeInstant, OrderSubtypeMarket,
		OrderSubtypeIOC, OrderSubtypeCashSell:
		return models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL
	case OrderSubtypeFOK:
		return models.TIME_IN_FORCE_FILL_OR_KILL
	case OrderSubtypeDaily, OrderSubtypeGTD:
		return models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME
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
