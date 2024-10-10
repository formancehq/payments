package client

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context, accountID string) ([]*Balance, error)
	GetRecipients(ctx context.Context, accountID string, page int, pageSize int) ([]*Recipient, error)
	GetTransactions(ctx context.Context, accountID string, page, pageSize int, lastCreatedAt time.Time) ([]*Transaction, error)
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
}

func New(clientID, apiKey, endpoint string) (*client, error) {
	config := &httpwrapper.Config{
		Transport: &apiTransport{
			clientID:   clientID,
			apiKey:     apiKey,
			endpoint:   endpoint,
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
		HttpErrorCheckerFn: func(statusCode int) error {
			if statusCode == http.StatusNotFound {
				return nil
			} else if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
				return httpwrapper.ErrStatusCodeClientError
			} else if statusCode >= http.StatusInternalServerError {
				return httpwrapper.ErrStatusCodeServerError
			}
			return nil

		},
	}
	endpoint = strings.TrimSuffix(endpoint, "/")

	httpClient, err := httpwrapper.NewClient(config)
	c := &client{
		httpClient: httpClient,
		endpoint:   endpoint,
	}
	return c, err
}
