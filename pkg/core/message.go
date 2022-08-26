package core

import (
	"time"
)

const (
	TopicPayments = "payments"

	EventVersion = "v1"
	EventApp     = "payments"

	EventPaymentsTypeSavedPayment = "SAVED_PAYMENT"
)

type EventPaymentsMessage[T any] struct {
	Date    time.Time `json:"date"`
	App     string    `json:"app"`
	Version string    `json:"version"`
	Type    string    `json:"type"`
	Payload T         `json:"payload"`
}

type SavedPayment ComputedPayment

func NewEventPaymentsSavedPayment(payload SavedPayment) EventPaymentsMessage[SavedPayment] {
	return EventPaymentsMessage[SavedPayment]{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventPaymentsTypeSavedPayment,
		Payload: payload,
	}
}
