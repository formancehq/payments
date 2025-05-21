package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type WebhookEventType string

const (
	AccountTransactionsModified       WebhookEventType = "account-transactions:modified"
	AccountTransactionsDeleted        WebhookEventType = "account-transactions:deleted"
	AccountBookedTransactionsModified WebhookEventType = "account-booked-transactions:modified"
	AccountCreated                    WebhookEventType = "account:created"
	AccountUpdated                    WebhookEventType = "account:updated"
	RefreshFinished                   WebhookEventType = "refresh:finished"
)

type CreateWebhookRequest struct {
	Description   string   `json:"description"`
	EnabledEvents []string `json:"enabledEvents"`
	URL           string   `json:"url"`
}

type CreateWebhookResponse struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
	Disabled      bool      `json:"disabled"`
	EnabledEvents []string  `json:"enabledEvents"`
	URL           string    `json:"url"`
	Secret        string    `json:"secret"`
}

func (p *client) CreateWebhook(ctx context.Context, eventType WebhookEventType, connectorID string, url string) (CreateWebhookResponse, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_webhook")

	body, err := json.Marshal(&CreateWebhookRequest{
		Description:   fmt.Sprintf("%s: %s", connectorID, string(eventType)),
		EnabledEvents: []string{string(eventType)},
		URL:           url,
	})
	if err != nil {
		return CreateWebhookResponse{}, err
	}

	endpoint := fmt.Sprintf("%s/events/v2/webhook-endpoints", p.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return CreateWebhookResponse{}, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	var resp CreateWebhookResponse
	_, err = p.httpClient.Do(ctx, req, &resp, nil)
	if err != nil {
		return CreateWebhookResponse{}, fmt.Errorf("failed to create webhook: %w", err)
	}

	return resp, nil
}
