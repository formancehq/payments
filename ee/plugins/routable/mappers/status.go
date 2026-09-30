package mappers

import (
	"strings"

	"github.com/formancehq/payments/pkg/domain/models"
)

// PayableStatus maps a Routable status onto a Formance payment status.
// Payables and receivables share one enum upstream (`items.common.Status`),
// so one mapper serves both. Unknown values fall through to UNKNOWN so the
// engine records them rather than coercing them.
//
// Enum: https://developers.routable.com/reference/list-payables
// Semantics: https://developers.routable.com/docs/payment-statuses
func PayableStatus(s string) models.PaymentStatus {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "created", "needs_approval", "ready_to_send", "scheduled",
		"compliance_hold", "po_discrepancy_hold",
		"pending", "processing", "initiated":
		return models.PAYMENT_STATUS_PENDING
	case "completed", "externally_paid":
		return models.PAYMENT_STATUS_SUCCEEDED
	case "issue", "failed":
		return models.PAYMENT_STATUS_FAILED
	case "canceled", "cancelled":
		return models.PAYMENT_STATUS_CANCELLED
	default:
		return models.PAYMENT_STATUS_UNKNOWN
	}
}

func IsTerminalStatus(s models.PaymentStatus) bool {
	switch s {
	case models.PAYMENT_STATUS_SUCCEEDED,
		models.PAYMENT_STATUS_FAILED,
		models.PAYMENT_STATUS_CANCELLED:
		return true
	default:
		return false
	}
}
