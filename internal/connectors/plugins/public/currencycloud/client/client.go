package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, int, error)
	GetBalances(ctx context.Context, page int, pageSize int) ([]*Balance, int, error)
	GetBeneficiaries(ctx context.Context, page int, pageSize int) ([]*Beneficiary, int, error)
	GetContactID(ctx context.Context, accountID string) (*Contact, error)
	GetTransactions(ctx context.Context, page int, pageSize int, updatedAtFrom time.Time) ([]Transaction, int, error)
}

type apiTransport struct {
	client     *client
	authToken  string
	underlying *otelhttp.Transport
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("X-Auth-Token", t.authToken)

	if t.authToken == "" {
		// Tokens expire after 30 minutes of inactivity which should not be the case
		// for us since we're polling the API frequently.
		authToken, err := t.client.authenticate(req.Context(), newHTTPClient())
		if err != nil {
			return nil, err
		}
		t.authToken = authToken
	}

	return t.underlying.RoundTrip(req)
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
	loginID    string
	apiKey     string
}

func (c *client) buildEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf("%s/%s", c.endpoint, fmt.Sprintf(path, args...))
}

const DevAPIEndpoint = "https://devapi.currencycloud.com"

func newHTTPClient() *http.Client {
	return &http.Client{
		Transport: otelhttp.NewTransport(http.DefaultTransport),
	}
}

// New creates a new client for the CurrencyCloud API.
func New(ctx context.Context, loginID, apiKey, endpoint string) (Client, error) {
	if endpoint == "" {
		endpoint = DevAPIEndpoint
	}

	c := &client{
		endpoint: endpoint,
		loginID:  loginID,
		apiKey:   apiKey,
	}

	config := &httpwrapper.Config{
		Transport: &apiTransport{
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
	httpClient, err := httpwrapper.NewClient(config)
	if err != nil {
		return nil, err
	}
	c.httpClient = httpClient
	return c, nil
}
