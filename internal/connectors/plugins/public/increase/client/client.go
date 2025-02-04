package client

import (
	"context"

	"github.com/Increase/increase-go"
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
	increaseClient *increase.Client
}

func New(apiKey string, environment string) *client {
	opts := []increase.Option{increase.WithAPIKey(apiKey)}
	if environment == "sandbox" {
		opts = append(opts, increase.WithBaseURL("https://sandbox.increase.com"))
	}
	return &client{
		increaseClient: increase.NewClient(opts...),
	}
}
