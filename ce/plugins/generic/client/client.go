package client

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/payments/ce/plugins/generic/client/generated"
	"github.com/formancehq/payments/pkg/domain/httpwrapper"
	"github.com/formancehq/payments/pkg/domain/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	ListAccounts(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Account, error)
	GetBalances(ctx context.Context, accountID string) (*genericclient.Balances, error)
	ListBeneficiaries(ctx context.Context, page, pageSize int64, createdAtFrom time.Time) ([]genericclient.Beneficiary, error)
	ListTransactions(ctx context.Context, page, pageSize int64, updatedAtFrom time.Time) ([]genericclient.Transaction, error)
	CreatePayout(ctx context.Context, request *PayoutRequest) (*PayoutResponse, error)
	CreateTransfer(ctx context.Context, request *TransferRequest) (*TransferResponse, error)
}

type apiTransport struct {
	APIKey     string
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.APIKey))

	return t.underlying.RoundTrip(req)
}

type genericAPIError struct {
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

type client struct {
	httpClient httpwrapper.Client
	baseURL    string
}

func New(connectorName string, apiKey, baseURL string) Client {
	transport := metrics.NewTransport(connectorName, metrics.TransportOpts{
		Transport: &apiTransport{
			APIKey:     apiKey,
			underlying: http.DefaultTransport,
		},
	})

	return &client{
		httpClient: httpwrapper.NewClient(&httpwrapper.Config{
			Transport: transport,
		}),
		baseURL: baseURL,
	}
}
