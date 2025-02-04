package client

import (
	"context"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/Increase/increase-go"
)

type WebhookEvent struct {
	ID                  string          `json:"id"`
	Type                string          `json:"type"`
	CreatedAt           time.Time       `json:"created_at"`
	Category            string          `json:"category"`
	AssociatedObjectID  string          `json:"associated_object_id"`
	AssociatedObjectType string         `json:"associated_object_type"`
	Data                map[string]any  `json:"data"`
}

type EventSubscription struct {
	ID        string    `json:"id"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
	Status    string    `json:"status"`
}

type CreateEventSubscriptionRequest struct {
	URL string `json:"url"`
}

func mapEventSubscription(s *increase.EventSubscription) *EventSubscription {
	return &EventSubscription{
		ID:        s.ID,
		URL:       s.URL,
		CreatedAt: s.CreatedAt,
		Status:    string(s.Status),
	}
}

func (c *client) CreateEventSubscription(ctx context.Context, req *CreateEventSubscriptionRequest) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, api.MetricOperationContextKey, "create_event_subscription")

	params := &increase.EventSubscriptionCreateParams{
		URL: req.URL,
	}

	subscription, err := c.sdk.EventSubscriptions.New(ctx, params)
	if err != nil {
		return nil, err
	}

	return mapEventSubscription(subscription), nil
}

func (c *client) VerifyWebhookSignature(payload []byte, header string) error {
	return increase.ValidateWebhookSignature(payload, header, c.apiKey)
}
