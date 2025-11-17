package client

import (
	"context"
	"net/url"
	"strings"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/stripe/stripe-go/v80"
)

const StripeConnectUrlPrefix = "connect_"

type endpointConfig struct {
	eventType string
	isConnect bool
}

var endpoints = []endpointConfig{
	{
		eventType: string(stripe.EventTypeBalanceAvailable),
		isConnect: false,
	},
	{
		eventType: string(stripe.EventTypeBalanceAvailable),
		isConnect: true,
	},
}

func (c *client) CreateWebhookEndpoints(ctx context.Context, webhookBaseURL string) ([]*stripe.WebhookEndpoint, error) {
	results := make([]*stripe.WebhookEndpoint, 0, 2)

	for _, conf := range endpoints {
		path := strings.ReplaceAll(string(conf.eventType), ".", "_")
		if conf.isConnect {
			path = StripeConnectUrlPrefix + path
		}

		u, err := url.JoinPath(webhookBaseURL, path)
		if err != nil {
			return results, err
		}

		params := &stripe.WebhookEndpointParams{
			EnabledEvents: []*string{
				stripe.String(conf.eventType),
			},
			URL:        stripe.String(u),
			APIVersion: stripe.String(stripe.APIVersion),
			Connect:    stripe.Bool(conf.isConnect),
		}
		result, err := c.webhookEndpointClient.New(params)
		if err != nil {
			return results, wrapSDKErr(err)
		}
		results = append(results, result)
	}
	return results, nil
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
