package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/pkg/connector/httpwrapper"
	"github.com/formancehq/payments/pkg/connector/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

type Client interface {
	GetAccounts(ctx context.Context, pageSize int, cursor string, createdAtAfter time.Time) ([]*Account, string, error)
	GetAccountBalance(ctx context.Context, accountID string) (*Balance, time.Time, error)
	GetExternalAccounts(ctx context.Context, pageSize int, cursor string) ([]*ExternalAccount, string, error)
	GetTransactions(ctx context.Context, pageSize int, timeline Timeline) ([]*Transaction, Timeline, bool, error)
	GetTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	GetPendingTransactions(ctx context.Context, pageSize int, timeline Timeline) ([]*Transaction, Timeline, bool, error)
	GetPendingTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	GetDeclinedTransactions(ctx context.Context, pageSize int, timeline Timeline) ([]*Transaction, Timeline, bool, error)
	GetDeclinedTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest, idempotencyKey string) (*TransferResponse, error)
	InitiateACHTransferPayout(ctx context.Context, pr *ACHPayoutRequest, idempotencyKey string) (*PayoutResponse, error)
	InitiateRTPTransferPayout(ctx context.Context, pr *RTPPayoutRequest, idempotencyKey string) (*PayoutResponse, error)
	InitiateCheckTransferPayout(ctx context.Context, pr *CheckPayoutRequest, idempotencyKey string) (*PayoutResponse, error)
	InitiateWireTransferPayout(ctx context.Context, pr *WireTransferPayoutRequest, idempotencyKey string) (*PayoutResponse, error)
	CreateBankAccount(ctx context.Context, pr *BankAccountRequest, idempotencyKey string) (*BankAccountResponse, error)
	CreateEventSubscription(ctx context.Context, req *CreateEventSubscriptionRequest, idempotencyKey string) (*EventSubscription, error)
	ListEventSubscriptions(ctx context.Context) ([]*EventSubscription, error)
	UpdateEventSubscription(ctx context.Context, req *UpdateEventSubscriptionRequest, webhookID string) (*EventSubscription, error)
	SetHttpClient(httpClient HTTPClient)
}

//go:generate mockgen -destination=http_generated.go -package=client . HTTPClient
type HTTPClient interface {
	Do(context.Context, *http.Request, any, any) (int, error)
}

type client struct {
	httpClient httpwrapper.Client

	apiKey              string
	endpoint            string
	webhookSharedSecret string
}

type apiTransport struct {
	client     *client
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.client.apiKey))
	req.Header.Add("Content-Type", "application/json")
	return t.underlying.RoundTrip(req)
}

type ResponseWrapper[t any] struct {
	Data             t      `json:"data"`
	NextCursor       string `json:"next_cursor"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

func New(connectorName, apiKey, endpoint, webhookSharedSecret string) *client {
	client := &client{
		apiKey:              apiKey,
		endpoint:            endpoint,
		webhookSharedSecret: webhookSharedSecret,
	}

	apiTransport := &apiTransport{
		client:     client,
		underlying: otelhttp.NewTransport(http.DefaultTransport),
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport("increase", metrics.TransportOpts{
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
