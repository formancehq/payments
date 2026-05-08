package mappers

import (
	"strings"

	"github.com/formancehq/payments/internal/models"
)

// PayableStatus maps Routable status strings; unknown values fall
// through to UNKNOWN so the engine logs them rather than coercing them.
func PayableStatus(s string) models.PaymentStatus {
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
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}

func IsTerminalStatus(s models.PaymentStatus) bool {
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
