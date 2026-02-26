package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"golang.org/x/oauth2/clientcredentials"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, cursor string, pageSize int) ([]Account, bool, string, error)
	GetAccountBalances(ctx context.Context, cursor string, pageSize int) ([]Balance, bool, string, error)
	GetTransactions(ctx context.Context, cursor string, pageSize int) ([]Transaction, bool, string, error)
}

type client struct {
	endpoint   string
	httpClient httpwrapper.Client
}

func New(connectorName, clientID, clientSecret, authEndpoint, endpoint string) Client {
	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
		OAuthConfig: &clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     authEndpoint + "/oauth/token",
		},
	}

	return &client{
		endpoint:   endpoint,
		httpClient: httpwrapper.NewClient(config),
	}
}
