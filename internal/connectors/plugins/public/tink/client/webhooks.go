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

func (p *client) DeleteWebhook(ctx context.Context, webhookID string) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_webhook")

	endpoint := fmt.Sprintf("%s/events/v2/webhook-endpoints/%s", p.endpoint, webhookID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	_, err = p.httpClient.Do(ctx, req, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}

	return nil
}

type WebhookContext struct {
	UserID         string `json:"userId"`
	ExternalUserID string `json:"externalUserId"`
}

type WebhookTransactions struct {
	EarliestModifiedBookedDate time.Time `json:"earliestModifiedBookedDate"`
	LatestModifiedBookedDate   time.Time `json:"latestModifiedBookedDate"`
	Inserted                   int       `json:"inserted"`
	Updated                    int       `json:"updated"`
	Deleted                    int       `json:"deleted"`
}

func (w *WebhookTransactions) UnmarshalJSON(data []byte) error {
	type webhookTransactions struct {
		EarliestModifiedBookedDate string `json:"earliestModifiedBookedDate"`
		LatestModifiedBookedDate   string `json:"latestModifiedBookedDate"`
		Inserted                   int    `json:"inserted"`
		Updated                    int    `json:"updated"`
		Deleted                    int    `json:"deleted"`
	}

	var tmp webhookTransactions
	if err := json.Unmarshal(data, &tmp); err != nil {
		return err
	}
	var earliestModifiedBookedDate time.Time
	if tmp.EarliestModifiedBookedDate != "" {
		parsed, err := time.Parse(time.DateOnly, tmp.EarliestModifiedBookedDate)
		if err != nil {
			return fmt.Errorf("invalid earliestModifiedBookedDate %q: %w", tmp.EarliestModifiedBookedDate, err)
		}
		earliestModifiedBookedDate = parsed
	}

	var latestModifiedBookedDate time.Time
	if tmp.LatestModifiedBookedDate != "" {
		parsed, err := time.Parse(time.DateOnly, tmp.LatestModifiedBookedDate)
		if err != nil {
			return fmt.Errorf("invalid latestModifiedBookedDate %q: %w", tmp.LatestModifiedBookedDate, err)
		}
		latestModifiedBookedDate = parsed
	}

	*w = WebhookTransactions{
		EarliestModifiedBookedDate: earliestModifiedBookedDate,
		LatestModifiedBookedDate:   latestModifiedBookedDate,
		Inserted:                   tmp.Inserted,
		Updated:                    tmp.Updated,
		Deleted:                    tmp.Deleted,
	}

	return nil
}

func (w WebhookTransactions) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		EarliestModifiedBookedDate string `json:"earliestModifiedBookedDate"`
		LatestModifiedBookedDate   string `json:"latestModifiedBookedDate"`
		Inserted                   int    `json:"inserted"`
		Updated                    int    `json:"updated"`
		Deleted                    int    `json:"deleted"`
	}{
		EarliestModifiedBookedDate: w.EarliestModifiedBookedDate.Format(time.DateOnly),
		LatestModifiedBookedDate:   w.LatestModifiedBookedDate.Format(time.DateOnly),
		Inserted:                   w.Inserted,
		Updated:                    w.Updated,
		Deleted:                    w.Deleted,
	})
}

type AccountTransactionsModifiedWebhook struct {
	UserID         string `json:"userId"`
	ExternalUserID string `json:"externalUserId"`
	Account        struct {
		ID string `json:"id"`
	} `json:"account"`
	Transactions WebhookTransactions `json:"transactions"`
}

func (c *client) GetAccountTransactionsModifiedWebhook(ctx context.Context, payload []byte) (AccountTransactionsModifiedWebhook, error) {
	type baseWebhook struct {
		Content AccountTransactionsModifiedWebhook `json:"content"`
	}

	var base baseWebhook
	if err := json.Unmarshal(payload, &base); err != nil {
		return AccountTransactionsModifiedWebhook{}, err
	}

	return base.Content, nil
}

type WebhookDeletedTransactions struct {
	IDs []string `json:"ids"`
}

type AccountTransactionsDeletedWebhook struct {
	UserID         string `json:"userId"`
	ExternalUserID string `json:"externalUserId"`
	Account        struct {
		ID string `json:"id"`
	} `json:"account"`
	Transactions WebhookDeletedTransactions `json:"transactions"`
}

func (c *client) GetAccountTransactionsDeletedWebhook(ctx context.Context, payload []byte) (AccountTransactionsDeletedWebhook, error) {
	type baseWebhook struct {
		Content AccountTransactionsDeletedWebhook `json:"content"`
	}

	var base baseWebhook
	if err := json.Unmarshal(payload, &base); err != nil {
		return AccountTransactionsDeletedWebhook{}, err
	}

	return base.Content, nil
}

type AccountCreatedWebhook struct {
	UserID         string `json:"userId"`
	ExternalUserID string `json:"externalUserId"`
	ID             string `json:"id"`
}

func (c *client) GetAccountCreatedWebhook(ctx context.Context, payload []byte) (AccountCreatedWebhook, error) {
	type baseWebhook struct {
		Context WebhookContext        `json:"context"`
		Content AccountCreatedWebhook `json:"content"`
	}

	var base baseWebhook
	if err := json.Unmarshal(payload, &base); err != nil {
		return AccountCreatedWebhook{}, err
	}

	base.Content.UserID = base.Context.UserID
	base.Content.ExternalUserID = base.Context.ExternalUserID

	return base.Content, nil
}

type RefreshFinishedWebhook struct {
	UserID            string `json:"userId"`
	ExternalUserID    string `json:"externalUserId"`
	CredentialsID     string `json:"credentialsId"`
	CredentialsStatus string `json:"credentialsStatus"`
	Finished          int64  `json:"finished"`
	DetailedError     struct {
		Type           string `json:"type"`
		DisplayMessage string `json:"displayMessage"`
		Details        struct {
			Reason    string `json:"reason"`
			Retryable bool   `json:"retryable"`
		} `json:"details"`
	} `json:"detailedError"`
}

func (c *client) GetRefreshFinishedWebhook(ctx context.Context, payload []byte) (RefreshFinishedWebhook, error) {
	type baseWebhook struct {
		Context WebhookContext         `json:"context"`
		Content RefreshFinishedWebhook `json:"content"`
	}

	var base baseWebhook
	if err := json.Unmarshal(payload, &base); err != nil {
		return RefreshFinishedWebhook{}, err
	}

	base.Content.UserID = base.Context.UserID
	base.Content.ExternalUserID = base.Context.ExternalUserID

	return base.Content, nil
}
