package mappers

import (
	"strconv"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
)

// MetadataPrefix namespaces every Bitstamp-specific metadata key.
// Re-exported from the plugin package so orchestrator code doesn't
// need to import mappers just for the prefix.
const MetadataPrefix = "com.bitstamp.spec/"

// Metadata key catalogue. Typed constants so refactors are
// grep-friendly and tests can assert exact key names.
const (
	MetadataKeyType          = MetadataPrefix + "type"
	MetadataKeyFee           = MetadataPrefix + "fee"
	MetadataKeyOrderID       = MetadataPrefix + "order_id"
	MetadataKeyCurrencyPair  = MetadataPrefix + "currency_pair"
	MetadataKeyClientOrderID = MetadataPrefix + "client_order_id"
	MetadataKeyRate          = MetadataPrefix + "rate"

	MetadataKeySource                     = MetadataPrefix + "source"
	MetadataKeyTransferPairID             = MetadataPrefix + "transfer_pair_id"
	MetadataKeyTransferDirection          = MetadataPrefix + "transfer_direction"
	MetadataKeyCounterpartySubAccountID   = MetadataPrefix + "counterparty_sub_account_id"
	MetadataKeyCounterpartySubAccountName = MetadataPrefix + "counterparty_sub_account_name"

	MetadataKeyNetworks        = MetadataPrefix + "networks"
	MetadataKeyWithdrawalFees  = MetadataPrefix + "withdrawal_fees"
	MetadataKeyTradableMarkets = MetadataPrefix + "tradable_markets"
	MetadataKeyFeeTierMaker    = MetadataPrefix + "fee_tier_maker"
	MetadataKeyFeeTierTaker    = MetadataPrefix + "fee_tier_taker"
	MetadataKeyMinOrderValue   = MetadataPrefix + "min_order_value"
	MetadataKeyMarketSymbol    = MetadataPrefix + "market_symbol"

	MetadataKeyOrderEventType      = MetadataPrefix + "order_event_type"
	MetadataKeyOrderEventID        = MetadataPrefix + "order_event_id"
	MetadataKeyOrderDatetimeSecs   = MetadataPrefix + "order_status_datetime_s"
	MetadataKeyOrderDatetimeMicros = MetadataPrefix + "order_status_datetime_ms"
)

const (
	PaymentSourceUserTransactions = "user_transactions"
)

const (
	TransferDirectionOutgoing = "outgoing"
	TransferDirectionIncoming = "incoming"
)

// setIfNonEmpty is the shared "omit empty values" pattern across
// every metadata builder below.
func setIfNonEmpty(m map[string]string, key, value string) {
	if value != "" {
		m[key] = value
	}
}

// PaymentMetadata for user_transactions rows. MetadataKeySource is
// always set so downstream can distinguish settled-history from
// PENDING crypto deposits or fiat withdrawal-request lifecycles.
func PaymentMetadata(tx client.UserTransaction) map[string]string {
	m := map[string]string{
		MetadataKeySource: PaymentSourceUserTransactions,
		MetadataKeyType:   tx.Type,
	}
	if !IsZeroAmount(tx.Fee) {
		m[MetadataKeyFee] = tx.Fee
	}
	if orderID := tx.OrderID.String(); orderID != "" && orderID != "0" {
		m[MetadataKeyOrderID] = orderID
	}
	return m
}

// TransferPairMetadata correlates the two halves of types 14/33/35.
// Each connector emits one leg; downstream joins on (transfer_pair_id,
// asset). Counterparty fields are omitted when absent.
func TransferPairMetadata(txID int64, direction, counterpartyID, counterpartyName string) map[string]string {
	m := map[string]string{
		MetadataKeyTransferPairID:    strconv.FormatInt(txID, 10),
		MetadataKeyTransferDirection: direction,
	}
	setIfNonEmpty(m, MetadataKeyCounterpartySubAccountID, counterpartyID)
	setIfNonEmpty(m, MetadataKeyCounterpartySubAccountName, counterpartyName)
	return m
}

// MergeMetadata folds maps left-to-right; later values override.
func MergeMetadata(maps ...map[string]string) map[string]string {
	out := map[string]string{}
	for _, m := range maps {
		for k, v := range m {
			out[k] = v
		}
	}
	return out
}

// ConversionMetadata for PSPConversion. pairRate is the value behind
// the dynamic <src>_<dst> rate key on the user_transactions row.
func ConversionMetadata(tx client.UserTransaction, currencyPair, pairRate string) map[string]string {
	m := map[string]string{
		MetadataKeyType:         tx.Type,
		MetadataKeyCurrencyPair: currencyPair,
	}
	setIfNonEmpty(m, MetadataKeyRate, pairRate)
	if !IsZeroAmount(tx.Fee) {
		m[MetadataKeyFee] = tx.Fee
	}
	if orderID := tx.OrderID.String(); orderID != "" && orderID != "0" {
		m[MetadataKeyOrderID] = orderID
	}
	return m
}
