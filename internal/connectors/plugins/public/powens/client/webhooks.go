package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type WebhookEventType string

const (
	WebhookEventTypeUserCreated                 WebhookEventType = "USER_CREATED"
	WebhookEventTypeUserDeleted                 WebhookEventType = "USER_DELETED"
	WebhookEventTypeConnectionSynced            WebhookEventType = "CONNECTION_SYNCED"
	WebhookEventTypeConnectionDeleted           WebhookEventType = "CONNECTION_DELETED"
	WebhookEventTypeAccountsFetched             WebhookEventType = "ACCOUNTS_FETCHED"
	WebhookEventTypeAccountSynced               WebhookEventType = "ACCOUNT_SYNCED"
	WebhookEventTypeAccountDisabled             WebhookEventType = "ACCOUNT_DISABLED"
	WebhookEventTypeAccountEnabled              WebhookEventType = "ACCOUNT_ENABLED"
	WebhookEventTypeAccountFound                WebhookEventType = "ACCOUNT_FOUND"
	WebhookEventTypeAccountOwnerhipsFound       WebhookEventType = "ACCOUNT_OWNERSHIPS_FOUND"
	WebhookEventTypeAccountCategorized          WebhookEventType = "ACCOUNT_CATEGORIZED"
	WebhookEventTypeSubscriptionFound           WebhookEventType = "SUBSCRIPTION_FOUND"
	WebhookEventTypeSubscriptionSynced          WebhookEventType = "SUBSCRIPTION_SYNCED"
	WebhookEventTypePaymentStateUpdated         WebhookEventType = "PAYMENT_STATE_UPDATED"
	WebhookEventTypeTransactionAttachmentsFound WebhookEventType = "TRANSACTION_ATTACHMENTS_FOUND"
)

type CreateWebhookAuthRequest struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type CreateWebhookAuthResponse struct {
	ID     string `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Config struct {
		SecretKey string `json:"secret_key"`
	} `json:"config"`
}

func (c *client) CreateWebhookAuth(ctx context.Context, connectorID string) (string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_webhook_auth")

	body, err := json.Marshal(&CreateWebhookAuthRequest{
		Type: "hmac_signature",
		Name: connectorID,
	})
	if err != nil {
		return "", err
	}

	endpoint := fmt.Sprintf("%s/webhooks/auth", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewBuffer(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.configurationToken))

	var resp CreateWebhookAuthResponse
	_, err = c.httpClient.Do(ctx, req, &resp, nil)
	if err != nil {
		return "", err
	}

	return resp.Config.SecretKey, nil
}
