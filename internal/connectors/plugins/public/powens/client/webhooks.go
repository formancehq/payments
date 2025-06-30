package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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
	ID     int    `json:"id"`
	Type   string `json:"type"`
	Name   string `json:"name"`
	Config struct {
		SecretKey string `json:"secret_key"`
	} `json:"config"`
}

func (c *client) CreateWebhookAuth(ctx context.Context, name string) (string, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_webhook_auth")

	endpoint := fmt.Sprintf("%s/webhooks/auth", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, http.NoBody)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.configurationToken))

	query := req.URL.Query()
	query.Add("type", "hmac_signature")

	// Connector ID is not accepted by the API, it returns a 500....
	// fortunately name is unique also
	query.Add("name", name)
	req.URL.RawQuery = query.Encode()

	var resp CreateWebhookAuthResponse
	_, err = c.httpClient.Do(ctx, req, &resp, nil)
	if err != nil {
		return "", err
	}

	return resp.Config.SecretKey, nil
}

type webhookAuthResponse struct {
	AuthProviders []WebhookAuth `json:"authproviders"`
}

type WebhookAuth struct {
	ID   int    `json:"id"`
	Type string `json:"type"`
	Name string `json:"name"`
}

// There is no api documentation for these webhook auth endpoints, and I didn't
// found anything to filter them by name. So for now, we have no choice but to
// list them all and filter them by name by hand after that.
// A ticket has been created on powens to add the missing api documentation and
// to add a filter by name.
func (c *client) ListWebhookAuths(ctx context.Context) ([]WebhookAuth, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_webhook_auths")

	endpoint := fmt.Sprintf("%s/webhooks/auth", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, http.NoBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.configurationToken))

	query := req.URL.Query()
	query.Add("limit", "1000")
	req.URL.RawQuery = query.Encode()

	var resp webhookAuthResponse
	_, err = c.httpClient.Do(ctx, req, &resp, nil)
	if err != nil {
		return nil, err
	}

	return resp.AuthProviders, nil
}

func (c *client) DeleteWebhookAuth(ctx context.Context, id int) error {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "delete_webhook_auth")

	endpoint := fmt.Sprintf("%s/webhooks/auth/%d", c.endpoint, id)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, http.NoBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.configurationToken))

	_, err = c.httpClient.Do(ctx, req, nil, nil)
	if err != nil {
		return err
	}

	return nil
}

type AccountSyncedWebhook struct {
	BankAccountID int       `json:"id"`
	ConnectionID  int       `json:"id_connection"`
	UserID        int       `json:"id_user"`
	Name          string    `json:"name"`
	LastUpdate    time.Time `json:"last_update"`

	// We do not want to unmarshal the other fields as the payload can be too
	// big if we have lots of transactions. That can be the case if this is
	// the first connection synced, we're gonna have a full payload with all
	// transactions.

	// We only keep the bank account ID and the connection ID, so that we can
	// fetch manually the transactions.
}

func (w AccountSyncedWebhook) MarshalJSON() ([]byte, error) {
	lastUpdate := w.LastUpdate.Format(time.DateTime)

	return json.Marshal(struct {
		BankAccountID int    `json:"id"`
		ConnectionID  int    `json:"id_connection"`
		UserID        int    `json:"id_user"`
		Name          string `json:"name"`
		LastUpdate    string `json:"last_update"`
	}{
		BankAccountID: w.BankAccountID,
		ConnectionID:  w.ConnectionID,
		UserID:        w.UserID,
		Name:          w.Name,
		LastUpdate:    lastUpdate,
	})
}

func (w *AccountSyncedWebhook) UnmarshalJSON(data []byte) error {
	type accountSyncedWebhook struct {
		BankAccountID int    `json:"id"`
		ConnectionID  int    `json:"id_connection"`
		UserID        int    `json:"id_user"`
		Name          string `json:"name"`
		LastUpdate    string `json:"last_update"`
	}

	var aux accountSyncedWebhook
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}

	lastUpdate, err := time.Parse(time.DateTime, aux.LastUpdate)
	if err != nil {
		return err
	}

	*w = AccountSyncedWebhook{
		BankAccountID: aux.BankAccountID,
		ConnectionID:  aux.ConnectionID,
		UserID:        aux.UserID,
		Name:          aux.Name,
		LastUpdate:    lastUpdate,
	}
	return nil
}
