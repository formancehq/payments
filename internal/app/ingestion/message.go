package ingestion

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/app/models"

	"github.com/formancehq/go-libs/sharedlogging"
)

const (
	TopicPayments = "payments"
	TopicAccounts = "payments"

	EventVersion = "v1"
	EventApp     = "payments"

	EventTypeSavedPayments = "SAVED_PAYMENTS"
	EventTypeSavedAccounts = "SAVED_ACCOUNTS"
)

type EventMessage struct {
	Date    time.Time `json:"date"`
	App     string    `json:"app"`
	Version string    `json:"version"`
	Type    string    `json:"type"`
	Payload any       `json:"payload"`
}

func NewEventSavedPayments(payments []*models.Payment) EventMessage {
	return EventMessage{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventTypeSavedPayments,
		Payload: payments,
	}
}

func NewEventSavedAccounts(accounts []models.Account) EventMessage {
	return EventMessage{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventTypeSavedAccounts,
		Payload: accounts,
	}
}

func (i *DefaultIngester) publish(ctx context.Context, topic string, ev EventMessage) {
	if err := i.publisher.Publish(ctx, topic, ev); err != nil {
		sharedlogging.GetLogger(ctx).Errorf("Publishing message: %s", err)

		return
	}
}
