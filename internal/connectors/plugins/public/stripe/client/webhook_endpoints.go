package client

import (
	"context"
	"net/url"
	"strings"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/stripe/stripe-go/v79"
)

func (c *client) CreateWebhookEndpoint(ctx context.Context, webhookBaseURL string) (*stripe.WebhookEndpoint, error) {
	// TODO: let's allow the update of enabled events if the code changes

	u, err := url.JoinPath(webhookBaseURL, strings.ReplaceAll(string(stripe.EventTypeBalanceAvailable), ".", "_"))
	if err != nil {
		return nil, err
	}
	c.logger.Infof("webhook url: %s", u)
	params := &stripe.WebhookEndpointParams{
		EnabledEvents: []*string{
			stripe.String(string(stripe.EventTypeBalanceAvailable)),
		},
		URL: stripe.String(u),
	}
	result, err := c.webhookEndpointClient.New(params)
	if err != nil {
		return nil, wrapSDKErr(err)
	}
	return result, nil
}

func (c *client) DeleteWebhookEndpoints(ctx context.Context) error {
	filters := stripe.ListParams{
		Context: metrics.OperationContext(ctx, "list_webhook_endpoints"),
	}
	itr := c.webhookEndpointClient.List(&stripe.WebhookEndpointListParams{
		ListParams: filters,
	})

	for _, data := range itr.WebhookEndpointList().Data {
		_, err := c.webhookEndpointClient.Del(data.ID, &stripe.WebhookEndpointParams{})
		if err != nil {
			return wrapSDKErr(err)
		}
	}
	return nil
}
