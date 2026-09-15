package mappers

import (
	"testing"

	"github.com/formancehq/payments/pkg/domain/models"
)

// allRoutableStatuses is Routable's published `items.common.Status` enum,
// shared by payables and receivables. Every value must map to something other
// than UNKNOWN — an UNKNOWN here is the bug this table exists to catch.
var allRoutableStatuses = []string{
	"canceled",
	"completed",
	"compliance_hold",
	"created",
	"externally_paid",
	"failed",
	"initiated",
	"issue",
	"needs_approval",
	"pending",
	"po_discrepancy_hold",
	"processing",
	"ready_to_send",
	"scheduled",
}

func TestPayableStatusCoversPublishedEnum(t *testing.T) {
	for _, s := range allRoutableStatuses {
		if got := PayableStatus(s); got == models.PAYMENT_STATUS_UNKNOWN {
			t.Errorf("PayableStatus(%q) = UNKNOWN, want a concrete status", s)
		}
	}
}

func TestPayableStatus(t *testing.T) {
	cases := map[string]models.PaymentStatus{
		// Pre-send: nothing has moved yet, but the item is still live.
		"created":             models.PAYMENT_STATUS_PENDING,
		"needs_approval":      models.PAYMENT_STATUS_PENDING,
		"ready_to_send":       models.PAYMENT_STATUS_PENDING,
		"scheduled":           models.PAYMENT_STATUS_PENDING,
		"compliance_hold":     models.PAYMENT_STATUS_PENDING,
		"po_discrepancy_hold": models.PAYMENT_STATUS_PENDING,

		// In flight.
		"pending":    models.PAYMENT_STATUS_PENDING,
		"processing": models.PAYMENT_STATUS_PENDING,
		"initiated":  models.PAYMENT_STATUS_PENDING,

		// Succeeded.
		"completed":       models.PAYMENT_STATUS_SUCCEEDED,
		"externally_paid": models.PAYMENT_STATUS_SUCCEEDED,

		// Failed. `issue` = Routable could not initiate the transfer (bad bank
		// account, etc.); `failed` = it broke after initiation. Neither moved
		// money, so neither may sit in PENDING.
		"issue":  models.PAYMENT_STATUS_FAILED,
		"failed": models.PAYMENT_STATUS_FAILED,

		// Cancelled, plus the defensive spelling variant.
		"canceled":  models.PAYMENT_STATUS_CANCELLED,
		"cancelled": models.PAYMENT_STATUS_CANCELLED,

		// Outside the enum entirely.
		"unknown_state": models.PAYMENT_STATUS_UNKNOWN,
		"":              models.PAYMENT_STATUS_UNKNOWN,

		"COMPLETED":   models.PAYMENT_STATUS_SUCCEEDED, // case-insensitive
		"  pending  ": models.PAYMENT_STATUS_PENDING,   // trims whitespace
		"  ISSUE  ":   models.PAYMENT_STATUS_FAILED,
	}
	for in, want := range cases {
		if got := PayableStatus(in); got != want {
			t.Errorf("PayableStatus(%q) = %v, want %v", in, got, want)
		}
	}
}

// `issue` and `failed` are terminal for the 201-create branch in
// createPayout/createTransfer: the workflow ends with the failed payment
// instead of polling an item that will not move without human intervention.
func TestIssueIsTerminalForCreateBranch(t *testing.T) {
	if !IsTerminalStatus(PayableStatus("issue")) {
		t.Error("issue should map to a terminal status so createPayout stops polling")
	}
}

func TestIsTerminalStatus(t *testing.T) {
	terminal := []models.PaymentStatus{
		models.PAYMENT_STATUS_SUCCEEDED,
		models.PAYMENT_STATUS_FAILED,
		models.PAYMENT_STATUS_CANCELLED,
	}
	for _, s := range terminal {
		if !IsTerminalStatus(s) {
			t.Errorf("expected %v to be terminal", s)
		}
	}
	// UNKNOWN is deliberately non-terminal: createPayout keeps polling
	// rather than closing out a payment it could not read.
	for _, s := range []models.PaymentStatus{
		models.PAYMENT_STATUS_PENDING,
		models.PAYMENT_STATUS_UNKNOWN,
	} {
		if IsTerminalStatus(s) {
			t.Errorf("expected %v to be non-terminal", s)
		}
	}
}
