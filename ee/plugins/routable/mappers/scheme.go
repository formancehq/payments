package mappers

import (
	"strings"

	"github.com/formancehq/payments/internal/models"
)

// DeliveryMethodToScheme maps Routable's delivery_method onto the Formance
// PaymentScheme enum.
//
// Routable's v1 delivery_method values (per their reference docs):
// ach, ach_same_day, ach_expedited, wire, international_wire, check,
// payout (Routable balance / internal transfer), wallet. Of these, only
// ACH-family rails have a dedicated Formance scheme constant
// (PAYMENT_SCHEME_ACH). Wire, check, internal transfer, and wallet all
// land on PAYMENT_SCHEME_OTHER because the Formance enum does not (yet)
// expose dedicated values for them. Listing each case explicitly rather
// than falling through to default keeps the intent visible and lets table
// tests pin every documented Routable value.
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
