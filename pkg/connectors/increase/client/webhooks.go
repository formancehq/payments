package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/pkg/connector/metrics"
)

type EventCategory string

const (
	EventCategoryPendingTransactionUpdated  EventCategory = "pending_transaction.updated"
	EventCategoryPendingTransactionCreated  EventCategory = "pending_transaction.created"
	EventCategoryTransactionCreated         EventCategory = "transaction.created"
	EventCategoryDeclinedTransactionCreated EventCategory = "declined_transaction.created"
	EventCategoryCheckTransferUpdated       EventCategory = "check_transfer.updated"
)

type WebhookEvent struct {
	ID                   string `json:"id"`
	Type                 string `json:"type"`
	CreatedAt            string `json:"created_at"`
	Category             string `json:"category"`
	AssociatedObjectID   string `json:"associated_object_id"`
	AssociatedObjectType string `json:"associated_object_type"`
}

type EventSubscription struct {
	ID                    string `json:"id"`
	URL                   string `json:"url"`
	CreatedAt             string `json:"created_at"`
	Status                string `json:"status"`
	SelectedEventCategory string `json:"selected_event_category"`
}

type UpdateEventSubscriptionRequest struct {
	Status string `json:"status"`
}

type CreateEventSubscriptionRequest struct {
	SelectedEventCategory string `json:"selected_event_category"`
	SharedSecret          string `json:"shared_secret"`
	URL                   string `json:"url"`
}

func (c *client) CreateEventSubscription(ctx context.Context, es *CreateEventSubscriptionRequest, idempotencyKey string) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_event_subscription")

	if es.SharedSecret == "" {
		es.SharedSecret = c.webhookSharedSecret
	}

	body, err := json.Marshal(es)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPost, "event_subscriptions", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}
	req.Header.Add("Idempotency-Key", idempotencyKey)

	var res EventSubscription
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}

func (c *client) ListEventSubscriptions(ctx context.Context) ([]*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_event_subscription")

	req, err := c.newRequest(ctx, http.MethodGet, "event_subscriptions", http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}

	var res ResponseWrapper[[]*EventSubscription]
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to list web hooks: %w %w", err, errRes.Error())
	}
	return res.Data, nil
}

func (c *client) UpdateEventSubscription(ctx context.Context, es *UpdateEventSubscriptionRequest, eventID string) (*EventSubscription, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "update_event_subscription")

	body, err := json.Marshal(es)
	if err != nil {
		return nil, err
	}

	req, err := c.newRequest(ctx, http.MethodPatch, fmt.Sprintf("event_subscriptions/%s", eventID), bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrWebhookRequestFailed, err)
	}

	var res EventSubscription
	var errRes increaseError
	_, err = c.httpClient.Do(ctx, req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to update web hooks: %w %w", err, errRes.Error())
	}
	return &res, nil
}
