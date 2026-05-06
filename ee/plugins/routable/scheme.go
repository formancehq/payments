package routable

import (
	"strings"

	"github.com/formancehq/payments/internal/models"
)

// deliveryMethodToScheme maps Routable's delivery_method string onto the
// Formance PaymentScheme enum. Anything not explicitly modeled lands on
// PAYMENT_SCHEME_OTHER, which is the same fallback used by the other EE
// plugins for non-card rails.
func deliveryMethodToScheme(deliveryMethod string) models.PaymentScheme {
	switch strings.ToLower(deliveryMethod) {
	case "ach", "ach_standard", "ach_same_day", "ach_expedited":
		return models.PAYMENT_SCHEME_ACH
	case "wire", "international_wire", "swift", "international":
		return models.PAYMENT_SCHEME_OTHER
	case "check":
		return models.PAYMENT_SCHEME_OTHER
	default:
		return models.PAYMENT_SCHEME_OTHER
	}
}
