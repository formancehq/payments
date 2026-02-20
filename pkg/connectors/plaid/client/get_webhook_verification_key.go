package client

import (
	"context"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/plaid/plaid-go/v34/plaid"
)

func (c *client) GetWebhookVerificationKey(ctx context.Context, kid string) (*plaid.JWKPublicKey, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_webhook_verification_key")

	// Check if the key is already cached
	if key, ok := c.webhookKeysCache.Get(kid); ok {
		return key, nil
	}

	request := plaid.NewWebhookVerificationKeyGetRequest(kid)
	resp, _, err := c.client.PlaidApi.WebhookVerificationKeyGet(ctx).WebhookVerificationKeyGetRequest(*request).Execute()
	if err != nil {
		return nil, wrapSDKError(err)
	}

	// Cache the key for future use
	c.webhookKeysCache.Add(kid, &resp.Key)

	return &resp.Key, nil
}
