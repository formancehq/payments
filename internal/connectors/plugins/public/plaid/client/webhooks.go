package client

import (
	"encoding/json"

	"github.com/plaid/plaid-go/v34/plaid"
)

type BaseWebhooks struct {
	WebhookType plaid.WebhookType `json:"webhook_type"`
	WebhookCode string            `json:"webhook_code"`
	ItemID      string            `json:"item_id"`
	Environment string            `json:"environment"`
}

func (c *client) BaseWebhookTranslation(body []byte) (BaseWebhooks, error) {
	var webhook BaseWebhooks
	if err := json.Unmarshal(body, &webhook); err != nil {
		return BaseWebhooks{}, err
	}
	return webhook, nil
}

func (c *client) TranslateItemAddResultWebhook(body []byte) (plaid.ItemAddResultWebhook, error) {
	var webhook plaid.ItemAddResultWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return plaid.ItemAddResultWebhook{}, err
	}
	return webhook, nil
}

func (c *client) TranslateSessionFinishedWebhook(body []byte) (plaid.LinkSessionFinishedWebhook, error) {
	var webhook plaid.LinkSessionFinishedWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return plaid.LinkSessionFinishedWebhook{}, err
	}
	return webhook, nil
}

func (c *client) TranslateUserPendingDisconnectWebhook(body []byte) (plaid.PendingDisconnectWebhook, error) {
	var webhook plaid.PendingDisconnectWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return plaid.PendingDisconnectWebhook{}, err
	}
	return webhook, nil
}

func (c *client) TranslateUserPendingExpirationWebhook(body []byte) (plaid.PendingExpirationWebhook, error) {
	var webhook plaid.PendingExpirationWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return plaid.PendingExpirationWebhook{}, err
	}
	return webhook, nil
}

func (c *client) TranslateItemErrorWebhook(body []byte) (plaid.ItemErrorWebhook, error) {
	var webhook plaid.ItemErrorWebhook
	if err := json.Unmarshal(body, &webhook); err != nil {
		return plaid.ItemErrorWebhook{}, err
	}
	return webhook, nil
}

type ErrorWebhook struct {
	ErrorType       string `json:"error_type"`
	ErrorCodeReason string `json:"error_code_reason"`
	ErrorMessage    string `json:"error_message"`
	DisplayMessage  string `json:"display_message"`
}

type SyncUpdatesAvailableWebhook struct {
	BaseWebhooks
	InitialUpdateComplete    bool `json:"initial_update_complete"`
	HistoricalUpdateComplete bool `json:"historical_update_complete"`
}

type HistoricalUpdateWebhook struct {
	BaseWebhooks
	NewTransactionsCount int          `json:"new_transactions"`
	ErrorWebhook         ErrorWebhook `json:"error"`
}

type UserPermissionRevokedWebhook struct {
	BaseWebhooks
	ErrorWebhook ErrorWebhook `json:"error"`
}
