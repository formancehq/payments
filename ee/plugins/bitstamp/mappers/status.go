package mappers

import (
	"github.com/formancehq/payments/internal/models"
)

// Bitstamp user_transactions.type values. See MAPPINGS §4.3.
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

	// Small balance conversion: two legs in one row (debit/credit),
	// handled identically to type 36 by the conversions mapper.
	TxTypeSmallBalanceConversionSrc = "53"
	TxTypeSmallBalanceConversionDst = "55"

	// Derivatives operations — spot-only connector skips these with a
	// warning log regardless of whether a derivatives marker is present
	// in the raw JSON.
	TxTypeDerivativesPeriodicSettlement = "58"
	TxTypeInsuranceFundClaim            = "59"
	TxTypeInsuranceFundPremium          = "60"
	TxTypeCollateralLiquidation         = "61"
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
	if IsConversionType(txType) {
		return 0, false
	}
	if t, found := transactionTypeMap[txType]; found {
		return t, true
	}
	return models.PAYMENT_TYPE_OTHER, true
}

// IsConversionType reports whether a transaction type should be handled
// by the conversions mapper: type 2 (market trade fill), 36 (instant
// buy/sell), 53/55 (small-balance conversion).
func IsConversionType(txType string) bool {
	switch txType {
	case TxTypeMarketTrade, TxTypeBuySell, TxTypeSmallBalanceConversionSrc, TxTypeSmallBalanceConversionDst:
		return true
	}
	return false
}

// IsDerivativesType reports whether a transaction type is an inherently
// derivatives operation. Spot-only connector skips these with a warning.
func IsDerivativesType(txType string) bool {
	switch txType {
	case TxTypeDerivativesPeriodicSettlement, TxTypeInsuranceFundClaim,
		TxTypeInsuranceFundPremium, TxTypeCollateralLiquidation:
		return true
	}
	return false
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
	if IsConversionType(txType) || IsDerivativesType(txType) {
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
