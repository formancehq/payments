package routable

import (
	"strings"

	"github.com/formancehq/payments/internal/models"
)

// payableStatus maps Routable's documented payable/receivable status strings
// onto Formance PaymentStatus values. Lifted from the rules already validated
// in connector-routable/internal/mapper/status.go and kept narrow on purpose:
// any value the API surfaces that we have not seen lands on UNKNOWN so the
// engine logs it instead of silently treating it as another known state.
func payableStatus(s string) models.PaymentStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "draft", "ready_to_send", "pending", "scheduled",
		"initiated", "processing", "in_transit", "awaiting_delivery":
		return models.PAYMENT_STATUS_PENDING
	case "completed", "paid", "delivered":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "failed", "returned", "nsf":
		return models.PAYMENT_STATUS_FAILED
	case "stopped", "canceled", "cancelled", "voided":
		return models.PAYMENT_STATUS_CANCELLED
	case "expired":
		return models.PAYMENT_STATUS_EXPIRED
	case "":
		return models.PAYMENT_STATUS_UNKNOWN
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}

// isTerminalStatus returns true once the payable has reached a state from
// which it will not advance, which is the signal PollPayoutStatus uses to
// stop polling.
func isTerminalStatus(s models.PaymentStatus) bool {
	switch s {
	case models.PAYMENT_STATUS_SUCCEEDED,
		models.PAYMENT_STATUS_FAILED,
		models.PAYMENT_STATUS_CANCELLED,
		models.PAYMENT_STATUS_EXPIRED,
		models.PAYMENT_STATUS_REFUNDED:
		return true
	default:
		return false
	}
}
