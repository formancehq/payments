package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type EventCategory string

const (
	EventCategoryBookTransferUpdated        EventCategory = "book.transfer.updated"
	EventCategoryWireTransferCompleted      EventCategory = "wire.outgoing_transfer.completed"
	EventCategoryACHTransferSettled         EventCategory = "ach.outgoing_transfer.settled"
	EventCategoryInternationalWireCompleted EventCategory = "swift.outgoing_transfer.completed"
	EventCategoryRealtimeTransferCompleted  EventCategory = "realtime.outgoing_transfer.completed"
)

type WebhookEvent[t any] struct {
	ID        string `json:"id"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
	Data      t      `json:"data"`
}

type EventSubscription struct {
	ID            string   `json:"id"`
	URL           string   `json:"url"`
	CreatedAt     string   `json:"created_at"`
	UpdatedAt     string   `json:"updated_at"`
	Description   string   `json:"description"`
	EnabledEvents []string `json:"enabled_events"`
	Secret        string   `json:"secret"`
	IsDisabled    bool     `json:"is_disabled"`
}

type CreateEventSubscriptionRequest struct {
	EnabledEvents []string `json:"enabled_events"`
	URL           string   `json:"url"`
}

type ListWebhookResponseWrapper[t any] struct {
	WebhookEndpoints t `json:"webhook_endpoints"`
}

func (c *client) CreateEventSubscription(ctx context.Context, es *CreateEventSubscriptionRequest) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_event_subscription")

	body, err := json.Marshal(es)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "webhook-endpoints", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}

	var res EventSubscription
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}

func (c *client) ListEventSubscriptions(ctx context.Context) ([]*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_event_subscription")

	req, err := c.newRequest(ctx, http.MethodGet, "webhook-endpoints", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}

	var res ListWebhookResponseWrapper[[]*EventSubscription]
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to list web hooks: %w %w", err, errRes.Error())
	}
	return res.WebhookEndpoints, nil
}

func (c *client) DeleteEventSubscription(ctx context.Context, eventID string) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_event_subscription")

	req, err := c.newRequest(ctx, http.MethodDelete, fmt.Sprintf("webhook-endpoints/%s", eventID), http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}

	var res EventSubscription
	var errRes columnError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to delete web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}
