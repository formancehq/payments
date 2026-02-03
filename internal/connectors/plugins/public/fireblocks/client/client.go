package client

import (
	"context"
	"crypto/rsa"
	"net/http"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	ListAssets(ctx context.Context) ([]Asset, error)
	GetVaultAccountsPaged(ctx context.Context, cursor string, limit int) (*VaultAccountsPagedResponse, error)
	GetVaultAccount(ctx context.Context, vaultAccountID string) (*VaultAccount, error)
	GetVaultAccountAsset(ctx context.Context, vaultAccountID, assetID string) (*VaultAsset, error)
	ListTransactions(ctx context.Context, createdAfter int64, limit int) ([]Transaction, error)
}

type client struct {
	httpClient httpwrapper.Client
	baseURL    string
}

func New(connectorName string, apiKey string, privateKey *rsa.PrivateKey, baseURL string) Client {
	transport := metrics.NewTransport(connectorName, metrics.TransportOpts{
		Transport: &fireblocksTransport{
			apiKey:     apiKey,
			privateKey: privateKey,
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
	})

	return &client{
		httpClient: httpwrapper.NewClient(&httpwrapper.Config{
			Transport: transport,
		}),
		baseURL: baseURL,
	}
}
