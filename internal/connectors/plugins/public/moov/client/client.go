package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetWallets(ctx context.Context, accountID string) ([]moov.Wallet, error)
	GetUsers(ctx context.Context, page int, pageSize int) ([]moov.Account, error)
	GetWallet(ctx context.Context, accountID string, walletID string) (*moov.Wallet, error)
	GetExternalAccounts(ctx context.Context, accountID string) ([]moov.BankAccount, error)
	GetPayments(ctx context.Context, accountID string, status moov.TransferStatus, skip int, count int, timeline Timeline) ([]moov.Transfer, Timeline, bool, int, error)
	NewWithClient(service MoovClient)
	InitiatePayout(ctx context.Context,
		sourceAccountId string,
		destinationAccountID string,
		pr moov.CreateTransfer) (*moov.Transfer, error)
}

type MoovClient interface {
	GetMoovAccounts(ctx context.Context, skip int, count int) ([]moov.Account, error)
	GetMoovWallets(ctx context.Context, accountID string) ([]moov.Wallet, error)
	GetMoovWallet(ctx context.Context, accountID string, walletID string) (*moov.Wallet, error)
	GetMoovBankAccounts(ctx context.Context, accountID string) ([]moov.BankAccount, error)
	GetMoovTransfers(ctx context.Context, accountID string, filters ...moov.ListTransferFilter) ([]moov.Transfer, error)
	CreateMoovTransfer(ctx context.Context, partnerAccountID string, transfer moov.CreateTransfer) (*moov.Transfer, *moov.TransferStarted, error)
	GetMoovTransferOptions(ctx context.Context, request PaymentOptionsRequest) (*moov.TransferOptions, error)
}

type PaymentOptionsRequest struct {
	PartnerAccountID     string
	SourceAccountID      string
	DestinationAccountID string
	Amount               int64
	Currency             string
}

type serviceWrapper struct {
	*moov.Client
}

type client struct {
	service MoovClient

	accountID string
}

func (c *serviceWrapper) GetMoovAccounts(ctx context.Context, skip int, count int) ([]moov.Account, error) {
	return c.ListAccounts(ctx, moov.Skip(skip), moov.Count(count))
}

func (c *serviceWrapper) GetMoovWallets(ctx context.Context, accountID string) ([]moov.Wallet, error) {
	return c.ListWallets(ctx, accountID)
}

func (c *serviceWrapper) GetMoovWallet(ctx context.Context, accountID string, walletID string) (*moov.Wallet, error) {
	return c.GetWallet(ctx, accountID, walletID)
}

func (c *serviceWrapper) GetMoovBankAccounts(ctx context.Context, accountID string) ([]moov.BankAccount, error) {
	return c.ListBankAccounts(ctx, accountID)
}

func (c *serviceWrapper) GetMoovTransfers(ctx context.Context, accountID string, filters ...moov.ListTransferFilter) ([]moov.Transfer, error) {
	return c.ListTransfers(ctx, accountID, filters...)
}

func (c *serviceWrapper) CreateMoovTransfer(ctx context.Context, partnerAccountID string, transfer moov.CreateTransfer) (*moov.Transfer, *moov.TransferStarted, error) {
	return c.CreateTransfer(ctx, partnerAccountID, transfer).WaitForRailResponse()
}

func (c *serviceWrapper) GetMoovTransferOptions(ctx context.Context, request PaymentOptionsRequest) (*moov.TransferOptions, error) {
	return c.TransferOptions(ctx, request.PartnerAccountID, moov.CreateTransferOptions{
		Source: moov.CreateTransferOptionsTarget{
			AccountID: request.SourceAccountID,
		},
		Destination: moov.CreateTransferOptionsTarget{
			AccountID: request.DestinationAccountID,
		},
		Amount: moov.Amount{
			Currency: request.Currency,
			Value:    request.Amount,
		},
	})
}

func New(connectorName string, endpoint string, publicKey string, secretKey string, accountID string) (*client, error) {

	moovClient, err := moov.NewClient(
		moov.WithHttpClient(metrics.NewHTTPClient(connectorName, models.DefaultConnectorClientTimeout)),
		moov.WithCredentials(moov.Credentials{
			PublicKey: publicKey,
			SecretKey: secretKey,
			Host:      endpoint,
		}),
	)

	if err != nil {
		return nil, err
	}

	client := &client{
		service:   &serviceWrapper{moovClient},
		accountID: accountID,
	}

	return client, nil
}

func (c *client) NewWithClient(service MoovClient) {
	c.service = service
}
