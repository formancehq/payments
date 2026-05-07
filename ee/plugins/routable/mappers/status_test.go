package mappers

import (
	"testing"

	"github.com/formancehq/payments/internal/models"
)

func TestPayableStatus(t *testing.T) {
	cases := map[string]models.PaymentStatus{
		"draft":         models.PAYMENT_STATUS_PENDING,
		"pending":       models.PAYMENT_STATUS_PENDING,
		"processing":    models.PAYMENT_STATUS_PENDING,
		"completed":     models.PAYMENT_STATUS_SUCCEEDED,
		"paid":          models.PAYMENT_STATUS_SUCCEEDED,
		"failed":        models.PAYMENT_STATUS_FAILED,
		"returned":      models.PAYMENT_STATUS_FAILED,
		"canceled":      models.PAYMENT_STATUS_CANCELLED,
		"cancelled":     models.PAYMENT_STATUS_CANCELLED,
		"expired":       models.PAYMENT_STATUS_EXPIRED,
		"unknown_state": models.PAYMENT_STATUS_UNKNOWN,
		"":              models.PAYMENT_STATUS_UNKNOWN,
		"COMPLETED":     models.PAYMENT_STATUS_SUCCEEDED, // case-insensitive
		"  pending  ":   models.PAYMENT_STATUS_PENDING,   // trims whitespace
	}
	for in, want := range cases {
		if got := PayableStatus(in); got != want {
			t.Errorf("PayableStatus(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestIsTerminalStatus(t *testing.T) {
	terminal := []models.PaymentStatus{
		models.PAYMENT_STATUS_SUCCEEDED,
		models.PAYMENT_STATUS_FAILED,
		models.PAYMENT_STATUS_CANCELLED,
		models.PAYMENT_STATUS_EXPIRED,
		models.PAYMENT_STATUS_REFUNDED,
	}
	for _, s := range terminal {
		if !IsTerminalStatus(s) {
			t.Errorf("expected %v to be terminal", s)
		}
	}
	if IsTerminalStatus(models.PAYMENT_STATUS_PENDING) {
		t.Error("PENDING should not be terminal")
	}
}
