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
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context) ([]*Balance, error)
	GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error)
	GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error)
	GetTransactionsByAccount(ctx context.Context, page, pageSize int, accountID string) ([]*Transaction, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)

	CreateCompany(ctx context.Context, reqBody *CreateCompanyRequest) (*Company, error)
	CreateContact(ctx context.Context, companyID string, reqBody *CreateContactRequest) (*Contact, error)
	CreateBankPaymentMethod(ctx context.Context, companyID string, reqBody *CreateBankPaymentMethodRequest) (*PaymentMethod, error)
}

type client struct {
	httpClient httpwrapper.Client

	apiToken string
	endpoint string
}

type apiTransport struct {
	client     *client
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.client.apiToken)
	req.Header.Set("Content-Type", "application/json")
	return t.underlying.RoundTrip(req)
}

func New(connectorName, apiToken, endpoint string) *client {
	c := &client{
		apiToken: apiToken,
		endpoint: endpoint,
	}

	apiTransport := &apiTransport{
		client:     c,
		underlying: otelhttp.NewTransport(http.DefaultTransport),
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{
			Transport: apiTransport,
		}),
	}

	c.httpClient = httpwrapper.NewClient(config)

	return c
}

func (c *client) newRequest(ctx context.Context, method, path string, body io.Reader) (*http.Request, error) {
	url := fmt.Sprintf("%s/%s", strings.TrimSuffix(c.endpoint, "/"), strings.TrimPrefix(path, "/"))
	return http.NewRequestWithContext(ctx, method, url, body)
}
