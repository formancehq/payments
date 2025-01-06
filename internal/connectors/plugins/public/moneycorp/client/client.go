package client

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context, accountID string) ([]*Balance, error)
	GetRecipients(ctx context.Context, accountID string, page int, pageSize int) ([]*Recipient, error)
	GetTransactions(ctx context.Context, accountID string, page, pageSize int, lastCreatedAt time.Time) ([]*Transaction, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	GetTransfer(ctx context.Context, accountID string, transferID string) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)
}

type client struct {
	httpClient httpwrapper.Client
	endpoint   string
}

func New(connectorName, clientID, apiKey, endpoint string) *client {
	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{Transport: &apiTransport{
			connectorName: connectorName,
			clientID:      clientID,
			apiKey:        apiKey,
			endpoint:      endpoint,
			underlying:    otelhttp.NewTransport(http.DefaultTransport),
		}}),
		HttpErrorCheckerFn: func(statusCode int) error {
			if statusCode == http.StatusTooManyRequests {
				return httpwrapper.ErrStatusCodeTooManyRequests
			}

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

	return &client{
		httpClient: httpwrapper.NewClient(config),
		endpoint:   endpoint,
	}
}
