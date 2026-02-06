package client

import (
	"context"
	"net/url"

	"github.com/formancehq/payments/pkg/connector/metrics"
	"github.com/formancehq/payments/pkg/connector"
	"github.com/stripe/stripe-go/v80"
)

const StripeConnectUrlPrefix = "connect_"

type endpointConfig struct {
	eventTypes []*string
	isConnect  bool
}

var endpoints = []endpointConfig{
	{
		eventTypes: []*string{
			stripe.String(string(stripe.EventTypeBalanceAvailable)),
		},
		isConnect: false,
	},
	{
		eventTypes: []*string{stripe.String(string(stripe.EventTypeBalanceAvailable))},
		isConnect:  true,
	},
}

func urlForEndpoint(webhookBaseURL string, conf endpointConfig) (string, error) {
	path := "root"
	if conf.isConnect {
		path = "connect"
	}
	return url.JoinPath(webhookBaseURL, path)
}

func (c *client) CreateWebhookEndpoints(ctx context.Context, webhookBaseURL string) ([]*stripe.WebhookEndpoint, error) {
	results := make([]*stripe.WebhookEndpoint, 0, len(endpoints))

	// in case this is being run at app startup the webhooks might already exist
	// remaining is a map of url -> endpointConfig
	updated, remaining, err := c.updateExistingWebhookEndpoints(ctx, webhookBaseURL)
	if err != nil {
		return results, wrapSDKErr(err)
	}
	results = append(results, updated...)

	for u, conf := range remaining {
		params := &stripe.WebhookEndpointParams{
			EnabledEvents: conf.eventTypes,
			URL:           stripe.String(u),
			APIVersion:    stripe.String(stripe.APIVersion),
			Connect:       stripe.Bool(conf.isConnect),
		}
		result, err := c.webhookEndpointClient.New(params)
		if err != nil {
			return results, wrapSDKErr(err)
		}
		results = append(results, result)
	}
	return results, nil
}

func (c *client) updateExistingWebhookEndpoints(ctx context.Context, webhookBaseURL string) (
	updated []*stripe.WebhookEndpoint,
	remaining map[string]endpointConfig,
	err error,
) {
	filters := stripe.ListParams{
		Context: metrics.OperationContext(ctx, "list_webhook_endpoints"),
	}
	itr := c.webhookEndpointClient.List(&stripe.WebhookEndpointListParams{
		ListParams: filters,
	})

	if err := itr.Err(); err != nil {
		return nil, remaining, err
	}

	// prepopulate a list of endpoints we want to ensure are created/up-to-date
	remaining = make(map[string]endpointConfig)
	for _, conf := range endpoints {
		u, err := urlForEndpoint(webhookBaseURL, conf)
		if err != nil {
			return nil, remaining, err
		}
		remaining[u] = conf
	}

	updated = make([]*stripe.WebhookEndpoint, 0)
	for _, data := range itr.WebhookEndpointList().Data {
		conf, ok := remaining[data.URL]
		if !ok {
			continue
		}

		result, err := c.webhookEndpointClient.Update(
			data.ID,
			&stripe.WebhookEndpointParams{
				EnabledEvents: conf.eventTypes,
				URL:           stripe.String(data.URL),
			},
		)
		if err != nil {
			return nil, remaining, err
		}
		updated = append(updated, result)

		// delete this record from remaining so we don't create a duplicate endpoint
		delete(remaining, data.URL)
	}
	return updated, remaining, nil
}

func (c *client) DeleteWebhookEndpoints(configs []connector.PSPWebhookConfig) error {
	for _, conf := range configs {
		_, err := c.webhookEndpointClient.Del(conf.Name, &stripe.WebhookEndpointParams{})
		if err != nil {
			return wrapSDKErr(err)
		}
	}
	return nil
}
