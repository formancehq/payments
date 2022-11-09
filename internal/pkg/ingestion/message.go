package ingestion

import (
	"context"
	"time"

	"github.com/numary/payments/internal/pkg/payments"

	"github.com/numary/go-libs/sharedlogging"
)

const (
	TopicPayments = "payments"

	EventVersion = "v1"
	EventApp     = "payments"

	EventTypeSavedPayment = "SAVED_PAYMENT"
	EventTypeSavedAccount = "SAVED_ACCOUNT"
)

type EventMessage struct {
	Date    time.Time `json:"date"`
	App     string    `json:"app"`
	Version string    `json:"version"`
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
}

func NewEventSavedPayment(payment payments.SavedPayment) EventMessage {
	return EventMessage{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventTypeSavedPayment,
		Payload: payment,
	}
}

func NewEventSavedAccount(account payments.Account) EventMessage {
	return EventMessage{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventTypeSavedAccount,
		Payload: account,
	}
}

func (i *DefaultIngester) publish(ctx context.Context, topic string, ev EventMessage) {
	if err := i.publisher.Publish(ctx, topic, ev); err != nil {
		sharedlogging.GetLogger(ctx).Errorf("Publishing message: %s", err)

		return
	}
}
