package client

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetMonitors(ctx context.Context) ([]*Monitor, error)
	GetBalances(ctx context.Context) ([]*TokenBalance, error)
}

type apiTransport struct {
	apiKey     string
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.apiKey))
	return t.underlying.RoundTrip(req)
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
}

func New(connectorName, apiKey, endpoint string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")

	c := &client{
		endpoint: endpoint,
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: &apiTransport{
				apiKey:     apiKey,
				underlying: http.DefaultTransport,
			},
		}),
	}
	c.httpClient = httpwrapper.NewClient(config)

	return c
}

func (c *client) buildEndpoint(path string) string {
	return fmt.Sprintf("%s/%s", c.endpoint, path)
}
