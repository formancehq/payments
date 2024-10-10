package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type WebhookDelivery struct {
	Version string `json:"version"`
	URL     string `json:"url"`
}

type webhookSubscription struct {
	Name      string          `json:"name"`
	TriggerOn string          `json:"trigger_on"`
	Delivery  WebhookDelivery `json:"delivery"`
}

type WebhookSubscriptionResponse struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Delivery  WebhookDelivery `json:"delivery"`
	TriggerOn string          `json:"trigger_on"`
	Scope     struct {
		Domain string `json:"domain"`
	} `json:"scope"`
	CreatedBy struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

func (c *client) CreateWebhook(ctx context.Context, profileID uint64, name, triggerOn, url, version string) (*WebhookSubscriptionResponse, error) {
	reqBody, err := json.Marshal(webhookSubscription{
		Name:      name,
		TriggerOn: triggerOn,
		Delivery: struct {
			Version string `json:"version"`
			URL     string `json:"url"`
		}{
			Version: version,
			URL:     url,
		},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		c.endpoint(fmt.Sprintf("/v3/profiles/%d/subscriptions", profileID)),
		bytes.NewBuffer(reqBody),
	)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	var res WebhookSubscriptionResponse
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w %w", err, errRes.Error(statusCode).Error())
	}
	return &res, nil
}

func (c *client) ListWebhooksSubscription(ctx context.Context, profileID uint64) ([]WebhookSubscriptionResponse, error) {
	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet, c.endpoint(fmt.Sprintf("/v3/profiles/%d/subscriptions", profileID)), http.NoBody)
	if err != nil {
		return nil, err
	}

	var res []WebhookSubscriptionResponse
	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(req, &res, &errRes)
	if err != nil {
		return nil, fmt.Errorf("failed to get subscription: %w %w", err, errRes.Error(statusCode).Error())
	}
	return res, nil
}

func (c *client) DeleteWebhooks(ctx context.Context, profileID uint64, subscriptionID string) error {
	req, err := http.NewRequestWithContext(ctx,
		http.MethodDelete, c.endpoint(fmt.Sprintf("/v3/profiles/%d/subscriptions/%s", profileID, subscriptionID)), http.NoBody)
	if err != nil {
		return err
	}

	var errRes wiseErrors
	statusCode, err := c.httpClient.Do(req, nil, &errRes)
	if err != nil {
		return fmt.Errorf("failed to delete webhooks: %w %w", err, errRes.Error(statusCode).Error())
	}
	return nil
}

type transferStateChangedWebhookPayload struct {
	Data struct {
		Resource struct {
			Type      string `json:"type"`
			ID        uint64 `json:"id"`
			ProfileID uint64 `json:"profile_id"`
			AccountID uint64 `json:"account_id"`
		} `json:"resource"`
		CurrentState  string `json:"current_state"`
		PreviousState string `json:"previous_state"`
		OccurredAt    string `json:"occurred_at"`
	} `json:"data"`
	SubscriptionID string `json:"subscription_id"`
	EventType      string `json:"event_type"`
	SchemaVersion  string `json:"schema_version"`
	SentAt         string `json:"sent_at"`
}

func (c *client) TranslateTransferStateChangedWebhook(ctx context.Context, payload []byte) (Transfer, error) {
	var transferStatedChangedEvent transferStateChangedWebhookPayload
	err := json.Unmarshal(payload, &transferStatedChangedEvent)
	if err != nil {
		return Transfer{}, err
	}

	transfer, err := c.GetTransfer(ctx, fmt.Sprint(transferStatedChangedEvent.Data.Resource.ID))
	if err != nil {
		return Transfer{}, err
	}

	transfer.Created = transferStatedChangedEvent.Data.OccurredAt
	transfer.CreatedAt, err = time.Parse("2006-01-02 15:04:05", transfer.Created)
	if err != nil {
		return Transfer{}, fmt.Errorf("failed to parse created time: %w", err)
	}

	return *transfer, nil
}

type BalanceUpdateWebhookPayload struct {
	Data           BalanceUpdateWebhookData `json:"data"`
	SubscriptionID string                   `json:"subscription_id"`
	EventType      string                   `json:"event_type"`
	SchemaVersion  string                   `json:"schema_version"`
	SentAt         string                   `json:"sent_at"`
}

type BalanceUpdateWebhookData struct {
	Resource          BalanceUpdateWebhookResource `json:"resource"`
	Amount            json.Number                  `json:"amount"`
	BalanceID         uint64                       `json:"balance_id"`
	Currency          string                       `json:"currency"`
	TransactionType   string                       `json:"transaction_type"`
	OccurredAt        string                       `json:"occurred_at"`
	TransferReference string                       `json:"transfer_reference"`
	ChannelName       string                       `json:"channel_name"`
}

type BalanceUpdateWebhookResource struct {
	ID        uint64 `json:"id"`
	ProfileID uint64 `json:"profile_id"`
	Type      string `json:"type"`
}

func (c *client) TranslateBalanceUpdateWebhook(ctx context.Context, payload []byte) (BalanceUpdateWebhookPayload, error) {
	var balanceUpdateEvent BalanceUpdateWebhookPayload
	err := json.Unmarshal(payload, &balanceUpdateEvent)
	if err != nil {
		return BalanceUpdateWebhookPayload{}, err
	}

	return balanceUpdateEvent, nil
}
