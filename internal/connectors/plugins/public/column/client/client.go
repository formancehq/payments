package client

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client interface {
	GetAccounts(ctx context.Context, cursor string, pageSize int) ([]*Account, bool, error)
	GetAccountBalances(ctx context.Context, accountID string) (*Balance, error)
	GetCounterparties(ctx context.Context, cursor string, pageSize int) ([]*Counterparties, bool, error)
	GetTransactions(ctx context.Context, cursor string, pageSize int) ([]*Transaction, bool, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)
	CreateCounterPartyBankAccount(ctx context.Context, data CounterPartyBankAccountRequest) (CounterPartyBankAccountResponse, error)
	ReversePayout(ctx context.Context, pr *ReversePayoutRequest) (*ReversePayoutResponse, error)
	CreateEventSubscription(ctx context.Context, es *CreateEventSubscriptionRequest) (*EventSubscription, error)
	DeleteEventSubscription(ctx context.Context, eventID string) (*EventSubscription, error)
	ListEventSubscriptions(ctx context.Context) ([]*EventSubscription, error)
	SetHttpClient(httpClient HTTPClient)
}

type ColumnAddress struct {
	Line1       string `json:"line_1,omitempty"`
	Line2       string `json:"line_2,omitempty"`
	City        string `json:"city,omitempty"`
	State       string `json:"state,omitempty"`
	PostalCode  string `json:"postal_code,omitempty"`
	CountryCode string `json:"country_code,omitempty"`
}

//go:generate mockgen -destination=http_generated.go -package=client . HTTPClient
type HTTPClient interface {
	Do(context.Context, *http.Request, any, any) (int, error)
}

type client struct {
	httpClient httpwrapper.Client

	apiKey   string
	endpoint string
}

type apiTransport struct {
	client     *client
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	auth := fmt.Sprintf(":%s", t.client.apiKey)
	basicAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte(auth))

	req.Header.Set("Authorization", basicAuth)
	req.Header.Set("Content-Type", "application/json")

	return t.underlying.RoundTrip(req)
}

func New(connectorName, apiKey, endpoint string) *client {
	client := &client{
		apiKey:   apiKey,
		endpoint: endpoint,
	}

	apiTransport := &apiTransport{
		client:     client,
		underlying: otelhttp.NewTransport(http.DefaultTransport),
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: apiTransport,
		}),
	}

	client.httpClient = httpwrapper.NewClient(config)

	return client
}

func (c *client) SetHttpClient(httpClient HTTPClient) {
	c.httpClient = httpClient
}

func (c *client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	endpoint := fmt.Sprintf("%s/%s", strings.TrimSuffix(c.endpoint, "/"), path)
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return req, nil
}
