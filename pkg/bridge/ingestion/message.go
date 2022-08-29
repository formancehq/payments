package ingestion

import (
	"context"
	"encoding/json"
	"time"

	"github.com/numary/go-libs/sharedlogging"
	payments "github.com/numary/payments/pkg"
)

const (
	TopicPayments = "payments"

	EventVersion = "v1"
	EventApp     = "payments"

	EventTypeSavedPayment = "SAVED_PAYMENT"
)

type EventMessage struct {
	Date    time.Time       `json:"date"`
	App     string          `json:"app"`
	Version string          `json:"version"`
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

type SavedPayment payments.ComputedPayment

func NewEventSavedPayment(payment SavedPayment) EventMessage {
	payload, err := json.Marshal(payment)
	if err != nil {
		panic(err)
	}

	return EventMessage{
		Date:    time.Now().UTC(),
		App:     EventApp,
		Version: EventVersion,
		Type:    EventTypeSavedPayment,
		Payload: payload,
	}
}

func (i *defaultIngester) publish(ctx context.Context, topic string, ev EventMessage) {
	if err := i.publisher.Publish(ctx, topic, ev); err != nil {
		sharedlogging.GetLogger(ctx).Errorf("Publishing message: %s", err)
		return
	}
}
