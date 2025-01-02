package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	ListAccounts(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Account, error)
	GetBalances(ctx context.Context, accountID string) (*genericclient.Balances, error)
	ListBeneficiaries(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Beneficiary, error)
	ListTransactions(ctx context.Context, page, pageSize int64, updatedAtFrom time.Time) ([]genericclient.Transaction, error)
}

type apiTransport struct {
	APIKey     string
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.APIKey))

	return t.underlying.RoundTrip(req)
}

type client struct {
	apiClient *genericclient.APIClient
}

func New(connectorName string, apiKey, baseURL string) Client {
	transport := metrics.NewTransport(connectorName, metrics.TransportOpts{
		Transport: &apiTransport{
			APIKey:     apiKey,
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
	})

	configuration := genericclient.NewConfiguration()
	configuration.HTTPClient = &http.Client{Timeout: 5 * time.Second, Transport: transport}
	configuration.Servers[0].URL = baseURL

	genericClient := genericclient.NewAPIClient(configuration)

	return &client{
		apiClient: genericClient,
	}
}
