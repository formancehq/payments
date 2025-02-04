package client

import (
	"context"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	increase "github.com/increase/increase-go"
)

type Client interface {
	GetAccounts(ctx context.Context, lastID string, pageSize int64) ([]*Account, string, bool, error)
	GetAccountBalances(ctx context.Context, accountID string) ([]*Balance, error)
	GetTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error)
	GetPendingTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error)
	GetDeclinedTransactions(ctx context.Context, lastID string, pageSize int64) ([]*Transaction, string, bool, error)
	GetExternalAccounts(ctx context.Context, lastID string, pageSize int64) ([]*ExternalAccount, string, bool, error)
	CreateExternalAccount(ctx context.Context, req *CreateExternalAccountRequest) (*ExternalAccount, error)
	CreateTransfer(ctx context.Context, req *CreateTransferRequest) (*Transfer, error)
	CreateACHTransfer(ctx context.Context, req *CreateACHTransferRequest) (*Transfer, error)
	CreateWireTransfer(ctx context.Context, req *CreateWireTransferRequest) (*Transfer, error)
	CreateCheckTransfer(ctx context.Context, req *CreateCheckTransferRequest) (*Transfer, error)
	CreateRTPTransfer(ctx context.Context, req *CreateRTPTransferRequest) (*Transfer, error)
	CreateEventSubscription(ctx context.Context, req *CreateEventSubscriptionRequest) (*EventSubscription, error)
	VerifyWebhookSignature(payload []byte, signature string) error
}

type client struct {
	httpClient *http.Client
	sdk        *increase.Client
}

func NewClient(apiKey string) Client {
	httpClient := &http.Client{
		Transport: api.NewTransport("increase", api.TransportOpts{}),
	}

	sdk := increase.NewClient(
		apiKey,
		increase.WithHTTPClient(httpClient),
	)

	return &client{
		httpClient: httpClient,
		sdk:        sdk,
	}
}
