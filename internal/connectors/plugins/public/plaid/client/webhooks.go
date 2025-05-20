package client

import (
	"encoding/json"

	"github.com/plaid/plaid-go/v34/plaid"
)

type BaseWebhooks struct {
	WebhookType plaid.WebhookType `json:"webhook_type"`
	WebhookCode string            `json:"webhook_code"`
	ItemID      string            `json:"item_id"`
}

func (c *client) BaseWebhookTranslation(body []byte) (BaseWebhooks, error) {
	var webhook BaseWebhooks
	if err := json.Unmarshal(body, &webhook); err != nil {
		return BaseWebhooks{}, err
	}
	return webhook, nil
}
