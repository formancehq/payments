package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/sync/singleflight"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, int, error)
	GetBalances(ctx context.Context, page int, pageSize int) ([]*Balance, int, error)
	GetBeneficiaries(ctx context.Context, page int, pageSize int) ([]*Beneficiary, int, error)
	GetContactID(ctx context.Context, accountID string) (*Contact, error)
	GetTransactions(ctx context.Context, page int, pageSize int, updatedAtFrom time.Time) ([]Transaction, int, error)
	InitiateTransfer(ctx context.Context, transferRequest *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, payoutRequest *PayoutRequest) (*PayoutResponse, error)
}

type apiTransport struct {
	c          *client
	underlying *otelhttp.Transport
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	if t.c.authToken != "" {
		req.Header.Add("X-Auth-Token", t.c.authToken)
	}

	return t.underlying.RoundTrip(req)
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
	loginID    string
	apiKey     string

	singleFlight singleflight.Group
	authToken    string
}

func (c *client) buildEndpoint(path string, args ...interface{}) string {
	return fmt.Sprintf("%s/%s", c.endpoint, fmt.Sprintf(path, args...))
}

const DevAPIEndpoint = "https://devapi.currencycloud.com"

// New creates a new client for the CurrencyCloud API.
func New(connectorName string, loginID, apiKey, endpoint string) Client {
	if endpoint == "" {
		endpoint = DevAPIEndpoint
	}

	c := &client{
		endpoint: endpoint,
		loginID:  loginID,
		apiKey:   apiKey,
	}
	baseTransport := &apiTransport{
		c:          c,
		underlying: otelhttp.NewTransport(http.DefaultTransport),
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{Transport: baseTransport}),
	}
	c.httpClient = httpwrapper.NewClient(config)
	return c
}
