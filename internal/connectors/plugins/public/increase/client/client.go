package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, pageSize int, cursor string, createdAtAfter time.Time) ([]*Account, string, error)
	GetAccount(ctx context.Context, accountID string) (*Account, error)
	GetAccountBalance(ctx context.Context, accountID string) (*Balance, time.Time, error)
	GetExternalAccounts(ctx context.Context, pageSize int, cursor string) ([]*ExternalAccount, string, error)
	GetExternalAccount(ctx context.Context, accountID string) (*ExternalAccount, error)
	GetTransactions(ctx context.Context, pageSize int, createdAtAfter time.Time) ([]*Transaction, string, error)
	GetTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	GetPendingTransactions(ctx context.Context, pageSize int, createdAtAfter time.Time) ([]*Transaction, string, error)
	GetPendingTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	GetDeclinedTransactions(ctx context.Context, pageSize int, createdAtAfter time.Time) ([]*Transaction, string, error)
	GetDeclinedTransaction(ctx context.Context, transactionID string) (*Transaction, error)
	GetTransfer(ctx context.Context, transferID string) (*TransferResponse, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	GetACHTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error)
	GetRTPTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error)
	GetWireTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error)
	GetCheckTransferPayout(ctx context.Context, transferID string) (*PayoutResponse, error)
	InitiateACHTransferPayout(ctx context.Context, pr *ACHPayoutRequest) (*PayoutResponse, error)
	InitiateRTPTransferPayout(ctx context.Context, pr *RTPPayoutRequest) (*PayoutResponse, error)
	InitiateCheckTransferPayout(ctx context.Context, pr *CheckPayoutRequest) (*PayoutResponse, error)
	InitiateWireTransferPayout(ctx context.Context, pr *WireTransferPayoutRequest) (*PayoutResponse, error)
	CreateBankAccount(ctx context.Context, pr *BankAccountRequest) (*BankAccountResponse, error)
	CreateEventSubscription(ctx context.Context, req *CreateEventSubscriptionRequest) (*EventSubscription, error)
	ListEventSubscriptions(ctx context.Context) ([]*EventSubscription, error)
	UpdateEventSubscription(ctx context.Context, req *UpdateEventSubscriptionRequest, webhookID string) (*EventSubscription, error)
	VerifyWebhookSignature(payload []byte, signature string) error
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
	return t.underlying.RoundTrip(req)
}

type responseWrapper[t any] struct {
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

func (c *client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	endpoint := fmt.Sprintf("%s/%s", strings.TrimSuffix(c.endpoint, "/"), path)
	req, err := http.NewRequestWithContext(ctx, method, endpoint, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	return req, nil
}
