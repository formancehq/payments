package payment

import (
	"time"
)

const (
	TopicSavedPayment = "SAVED_PAYMENT"
)

type SavedPaymentEvent struct {
	Date    time.Time `json:"date"`
	Type    string    `json:"type"`
	Payload Payment   `json:"payload"`
}
