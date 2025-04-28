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
	GetOrganization(ctx context.Context) (*Organization, error)
	GetAccountBalances(ctx context.Context) ([]*Balance, error)
	GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error)
	GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)
}

type client struct {
	httpClient httpwrapper.Client

	clientID     string
	apiKey       string
	endpoint     string
	stagingToken string
}

type apiTransport struct {
	client     *client
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	auth := fmt.Sprintf("%s:%s", t.client.clientID, t.client.apiKey)

	req.Header.Set("Authorization", auth)
	req.Header.Set("Content-Type", "application/json")
	if t.client.stagingToken != "" {
		req.Header.Set("X-Qonto-Staging-Token", t.client.stagingToken)
	}
	return t.underlying.RoundTrip(req)
}

func New(connectorName, clientID, apiKey, endpoint, stagingToken string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")

	client := &client{
		clientID:     clientID,
		apiKey:       apiKey,
		endpoint:     endpoint,
		stagingToken: stagingToken,
	}

	apiTransport := &apiTransport{
		client:     client,
		underlying: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
	}
	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: apiTransport,
		}),
	}
	client.httpClient = httpwrapper.NewClient(config)

	return client
}

func (c *client) buildEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf("%s/%s", c.endpoint, fmt.Sprintf(path, args...))
}
