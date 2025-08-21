package client

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/checkout/checkout-sdk-go"
	"github.com/checkout/checkout-sdk-go/configuration"
	"github.com/checkout/checkout-sdk-go/nas"
	
	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context) ([]*Balance, error)
	GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error)
	GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)
}

type client struct {
	sdk              *nas.Api
	entityID 		 string
}

type acceptHeaderTransport struct {
    base http.RoundTripper
}

func (t *acceptHeaderTransport) RoundTrip(req *http.Request) (*http.Response, error) {
    r := req.Clone(req.Context())
    r.Header.Set("Accept", "application/json; schema_version=3.0")
    if r.Header.Get("Content-Type") == "" {
        r.Header.Set("Content-Type", "application/json")
    }
    return t.base.RoundTrip(r)
}

func New(
	env string,
	oauthClientID string,
	oauthClientSecret string,
	entityID string,
) *client {
	var environment configuration.Environment
	switch strings.ToLower(strings.TrimSpace(env)) {
	case "sandbox":
		environment = configuration.Sandbox()
	default:
		environment = configuration.Production()
	}

	httpClient := &http.Client{
		Transport: &acceptHeaderTransport{
			base: metrics.NewTransport("checkout", metrics.TransportOpts{}),
		},
		Timeout: 30 * time.Second,
	}

	sdk, err := checkout.Builder().
		OAuth().
		WithClientCredentials(strings.TrimSpace(oauthClientID), strings.TrimSpace(oauthClientSecret)).
		WithEnvironment(environment).
		WithHttpClient(httpClient).
		WithScopes(getOAuthScopes()).
		Build()
	if err != nil {
		panic(err)
	}

	return &client{
		sdk:      sdk,
		entityID: entityID,
	}
}

func getOAuthScopes() []string {
	return []string{"accounts", "balances"}
}