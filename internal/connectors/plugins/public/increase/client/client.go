package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, pageSize int, cursor string) ([]*Account, string, error)
	GetAccount(ctx context.Context, accountID string) (*Account, error)
	GetAccountBalance(ctx context.Context, accountID string) (*Balance, error)
	GetExternalAccounts(ctx context.Context, pageSize int, cursor string) ([]*ExternalAccount, string, error)
	GetTransactions(ctx context.Context, pageSize int, cursor string) ([]*Transaction, string, error)
	GetPendingTransactions(ctx context.Context, pageSize int, cursor string) ([]*Transaction, string, error)
	GetDeclinedTransactions(ctx context.Context, pageSize int, cursor string) ([]*Transaction, string, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiateACHTransferPayout(ctx context.Context, pr *ACHPayoutRequest) (*PayoutResponse, error)
	InitiateRTPTransferPayout(ctx context.Context, pr *RTPPayoutRequest) (*PayoutResponse, error)
	InitiateCheckTransferPayout(ctx context.Context, pr *CheckPayoutRequest) (*PayoutResponse, error)
	InitiateWireTransferPayout(ctx context.Context, pr *WireTransferPayoutRequest) (*PayoutResponse, error)
	CreateBankAccount(ctx context.Context, pr *BankAccountRequest) (*BankAccountResponse, error)
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
	req.Header.Add("Authorization", t.client.apiKey)
	return t.underlying.RoundTrip(req)
}

type responseWrapper[t any] struct {
	Data             t      `json:"data"`
	NextCursor       string `json:"next_cursor"`
	ResponseMetadata struct {
		NextCursor string `json:"next_cursor"`
	} `json:"response_metadata"`
}

const SandboxAPIEndpoint = "https://sandbox.increase.com"

func New(connectorName, apiKey, endpoint string) *client {
	if endpoint == "" {
		endpoint = SandboxAPIEndpoint
	}

	client := &client{
		apiKey:   apiKey,
		endpoint: endpoint,
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
