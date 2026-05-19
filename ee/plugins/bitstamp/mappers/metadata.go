package mappers

import (
	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
)

// MetadataPrefix namespaces every Bitstamp-specific metadata key per
// the Formance convention (com.<provider>.spec/...). The orchestrator
// re-exports this so plugin code doesn't need to import mappers just
// for the prefix.
const MetadataPrefix = "com.bitstamp.spec/"

// Metadata key catalogue. Kept as typed constants so refactors are
// grep-friendly and the test suite can assert exact key names.
const (
	MetadataKeyType              = MetadataPrefix + "type"
	MetadataKeyFee               = MetadataPrefix + "fee"
	MetadataKeyOrderID           = MetadataPrefix + "order_id"
	MetadataKeyCurrencyPair      = MetadataPrefix + "currency_pair"
	MetadataKeyOrderType         = MetadataPrefix + "order_type"
	MetadataKeyClientOrderID     = MetadataPrefix + "client_order_id"
	MetadataKeyRate              = MetadataPrefix + "rate"
	MetadataKeyRetentionExpired  = MetadataPrefix + "retention_expired"
)

// PaymentMetadata produces the namespaced metadata for a user
// transaction surfaced as a PSPPayment.
func PaymentMetadata(tx client.UserTransaction) map[string]string {
	m := map[string]string{MetadataKeyType: tx.Type}
	if !IsZeroAmount(tx.Fee) {
		m[MetadataKeyFee] = tx.Fee
	}
	if orderID := tx.OrderID.String(); orderID != "" && orderID != "0" {
		m[MetadataKeyOrderID] = orderID
	}
	return m
}

// OrderMetadata produces the namespaced metadata for a PSPOrder.
// retentionExpired is set to "true" only on the forced-final emit
// triggered by the 25-day TrackedOrders eviction policy (MAPPINGS.md
// §3.4.4 step 6).
func OrderMetadata(currencyPair string, clientOrderID string, retentionExpired bool) map[string]string {
	m := map[string]string{
		MetadataKeyCurrencyPair: currencyPair,
		MetadataKeyOrderType:    "limit", // open_orders/ returns limit orders only
	}
	if clientOrderID != "" {
		m[MetadataKeyClientOrderID] = clientOrderID
	}
	if retentionExpired {
		m[MetadataKeyRetentionExpired] = "true"
	}
	return m
}

// ConversionMetadata produces the namespaced metadata for a PSPConversion.
// pairRate is the value behind the dynamic <src>_<dst> rate key on
// the user_transactions row (e.g. "0.86047" for an EUR→USDC instant
// buy). currencyPair is "<src>_<dst>" (lowercase).
func ConversionMetadata(tx client.UserTransaction, currencyPair string, pairRate string) map[string]string {
	m := map[string]string{
		MetadataKeyType:         tx.Type,
		MetadataKeyCurrencyPair: currencyPair,
	}
	if pairRate != "" {
		m[MetadataKeyRate] = pairRate
	}
	if !IsZeroAmount(tx.Fee) {
		m[MetadataKeyFee] = tx.Fee
	}
	return m
}
