package core

import (
	"time"
)

const (
	TopicPayments = "payments"

	EventPaymentsSavedPayment = "SAVED_PAYMENT"
)

type EventPaymentsMessage[P any] struct {
	Date    time.Time `json:"date"`
	Type    string    `json:"type"`
	Payload P         `json:"payload"`
}

type SavedPayment ComputedPayment

func NewEventPaymentsSavedPayment(payload SavedPayment) EventPaymentsMessage[SavedPayment] {
	return EventPaymentsMessage[SavedPayment]{
		Date:    time.Now().UTC(),
		Type:    EventPaymentsSavedPayment,
		Payload: payload,
	}
}
