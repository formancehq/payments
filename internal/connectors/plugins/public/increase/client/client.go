package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error)
	GetAccountBalances(ctx context.Context) ([]*Balance, error)
	GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error)
	GetTransactions(ctx context.Context, page, pageSize int) ([]*Transaction, error)
	InitiateTransfer(ctx context.Context, tr *TransferRequest) (*TransferResponse, error)
	InitiatePayout(ctx context.Context, pr *PayoutRequest) (*PayoutResponse, error)
}

type client struct {
	httpClient httpwrapper.Client

	// TODO: fill config parameters
	// You may need fields here for authentication purpose
}

func New( /* TODO: fill config parameters */ ) *client {
	config := &httpwrapper.Config{
		// TODO: you can set an underlying http transport in metrics.TransportOpts for authentication for example
		Transport: metrics.NewTransport("increase", metrics.TransportOpts{}),
		// TODO: if the PSP requires special http status code handling, you can override the default handling by setting a
		// custom HttpErrorCheckerFn like below
		// HttpErrorCheckerFn: func(statusCode int) error {
		// 	if statusCode == http.StatusNotFound {
		// 		return nil
		// 	} else if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		// 		return httpwrapper.ErrStatusCodeClientError
		// 	} else if statusCode >= http.StatusInternalServerError {
		// 		return httpwrapper.ErrStatusCodeServerError
		// 	}
		// 	return nil
		// },
	}

	return &client{
		httpClient: httpwrapper.NewClient(config),
	}
}
