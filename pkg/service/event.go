package service

import (
	"github.com/numary/payments/pkg"
	"time"
)

const (
	TopicSavedPayment = "SAVED_PAYMENT"
)

type SavedPaymentEvent struct {
	Date    time.Time       `json:"date"`
	Type    string          `json:"type"`
	Payload payment.Payment `json:"payload"`
}
