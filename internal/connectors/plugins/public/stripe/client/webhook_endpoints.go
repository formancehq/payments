package client

import (
	"context"

	"github.com/stripe/stripe-go/v79"
)

func (c *client) CreateWebhookEndpoint(ctx context.Context, webhookBaseURL string) (*stripe.WebhookEndpoint, error) {
	// TODO: let's allow the update of enabled events if the code changes

	params := &stripe.WebhookEndpointParams{
		EnabledEvents: []*string{
			stripe.String(string(stripe.EventTypeBalanceAvailable)),
		},
		URL: stripe.String(webhookBaseURL),
	}
	result, err := c.webhookEndpointClient.New(params)
	if err != nil {
		return nil, wrapSDKErr(err)
	}
	return result, nil
}
