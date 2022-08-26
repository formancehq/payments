package ingestion

import (
	"time"

	"github.com/numary/payments/pkg/core"
)

const (
	TopicPayments             = "payments"
	EventPaymentsSavedPayment = "SAVED_PAYMENT"
)

type EventPaymentsMessage struct {
	Date    time.Time            `json:"date"`
	Type    string               `json:"type"`
	Payload core.ComputedPayment `json:"payload"`
}
