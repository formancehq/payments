package mappers

import (
	"strings"

	"github.com/formancehq/payments/internal/models"
)

// DeliveryMethodToScheme covers every Routable v1 delivery_method
// explicitly so a new value lands on a known case rather than the
// default. PAYMENT_SCHEME_OTHER is used wherever the Formance enum has
// no dedicated constant (wire, check, internal transfer, wallet).
func DeliveryMethodToScheme(deliveryMethod string) models.PaymentScheme {
	switch strings.ToLower(strings.TrimSpace(deliveryMethod)) {
	case "ach", "ach_standard", "ach_same_day", "same_day_ach", "ach_expedited":
		return models.PAYMENT_SCHEME_ACH
	case "wire", "international_wire", "swift", "international",
		"check",
		"payout", // Routable balance / internal transfer
		"wallet": // digital wallet
		return models.PAYMENT_SCHEME_OTHER
	default:
		return models.PAYMENT_SCHEME_OTHER
	}
}
