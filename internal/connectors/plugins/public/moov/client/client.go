package client

import (
	"context"
	"net/http"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, skip int, count int) ([]*moov.Account, bool, error)
	GetWallets(ctx context.Context, accountID string, skip int, count int) ([]*moov.Wallet, bool, error)
	GetBankAccounts(ctx context.Context, accountID string, skip int, count int) ([]*moov.BankAccount, bool, error)
	GetTransfers(ctx context.Context, startTime time.Time, skip int, count int) ([]*moov.Transfer, bool, error)
	CreateTransfer(ctx context.Context, req *TransferRequest) (*moov.Transfer, error)
	CreatePayout(ctx context.Context, req *PayoutRequest) (*moov.Transfer, error)
}

type client struct {
	moovClient *moov.Client
	metricsClient httpwrapper.Client
}

// TransferRequest represents a request to create a transfer between Moov wallets
type TransferRequest struct {
	SourceWalletID      string
	DestinationWalletID string
	Amount              int64
	Currency            string
	Description         string
	Metadata            map[string]string
}

// PayoutRequest represents a request to create a payout from a Moov wallet to a bank account
type PayoutRequest struct {
	WalletID      string
	BankAccountID string
	Amount        int64
	Currency      string
	Description   string
	Metadata      map[string]string
}

// New creates a new Moov client with metrics collection
func New(connectorName string, publicKey, secretKey, environment string) (Client, error) {
	// Set the base URL based on the environment
	baseURL := "https://api.moov.io"
	if environment == "sandbox" {
		baseURL = "https://api.sandbox.moov.io"
	}

	// Initialize Moov client with credentials
	mc, err := moov.NewClient(
		moov.WithCredentials(
			moov.NewCredentials(publicKey, secretKey),
		),
		moov.WithBaseURL(baseURL),
	)
	if err != nil {
		return nil, err
	}

	// Create metrics client
	metricsClient := httpwrapper.NewClient(&httpwrapper.Config{
		Timeout: models.DefaultConnectorClientTimeout,
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
	})

	return &client{
		moovClient: mc,
		metricsClient: metricsClient,
	}, nil
}

// GetAccounts fetches accounts from the Moov API
func (c *client) GetAccounts(ctx context.Context, skip int, count int) ([]*moov.Account, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	// Set up pagination parameters
	params := &moov.ListAccountParams{
		Skip:  skip,
		Count: count,
	}

	// Call the Moov API
	accounts, resp, err := c.moovClient.ListAccounts(ctx, params)
	if err != nil {
		return nil, false, err
	}

	// Check if there are more accounts to fetch
	hasMore := len(accounts) == count

	return accounts, hasMore, nil
}

// GetWallets fetches wallets for an account from the Moov API
func (c *client) GetWallets(ctx context.Context, accountID string, skip int, count int) ([]*moov.Wallet, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_wallets")

	// Set up pagination parameters
	params := &moov.ListWalletParams{
		Skip:  skip,
		Count: count,
	}

	// Call the Moov API
	wallets, resp, err := c.moovClient.ListWallets(ctx, accountID, params)
	if err != nil {
		return nil, false, err
	}

	// Check if there are more wallets to fetch
	hasMore := len(wallets) == count

	return wallets, hasMore, nil
}

// GetBankAccounts fetches bank accounts for an account from the Moov API
func (c *client) GetBankAccounts(ctx context.Context, accountID string, skip int, count int) ([]*moov.BankAccount, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_bank_accounts")

	// Set up pagination parameters
	params := &moov.ListBankAccountParams{
		Skip:  skip,
		Count: count,
	}

	// Call the Moov API
	bankAccounts, resp, err := c.moovClient.ListBankAccounts(ctx, accountID, params)
	if err != nil {
		return nil, false, err
	}

	// Check if there are more bank accounts to fetch
	hasMore := len(bankAccounts) == count

	return bankAccounts, hasMore, nil
}

// GetTransfers fetches transfers from the Moov API
func (c *client) GetTransfers(ctx context.Context, startTime time.Time, skip int, count int) ([]*moov.Transfer, bool, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_transfers")

	// Set up pagination parameters
	params := &moov.ListTransferParams{
		Skip:      skip,
		Count:     count,
		StartTime: startTime,
	}

	// Call the Moov API
	transfers, resp, err := c.moovClient.ListTransfers(ctx, params)
	if err != nil {
		return nil, false, err
	}

	// Check if there are more transfers to fetch
	hasMore := len(transfers) == count

	return transfers, hasMore, nil
}

// CreateTransfer creates a transfer between Moov wallets
func (c *client) CreateTransfer(ctx context.Context, req *TransferRequest) (*moov.Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_transfer")

	// Create the transfer request
	transferReq := moov.CreateTransferRequest{
		Source: moov.Source{
			WalletID: req.SourceWalletID,
		},
		Destination: moov.Destination{
			WalletID: req.DestinationWalletID,
		},
		Amount: moov.Amount{
			Value:    req.Amount,
			Currency: req.Currency,
		},
		Description: req.Description,
		Metadata:    req.Metadata,
	}

	// Call the Moov API
	transfer, resp, err := c.moovClient.CreateTransfer(ctx, transferReq)
	if err != nil {
		return nil, err
	}

	return transfer, nil
}

// CreatePayout creates a payout from a Moov wallet to a bank account
func (c *client) CreatePayout(ctx context.Context, req *PayoutRequest) (*moov.Transfer, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "create_payout")

	// Create the transfer request for a payout
	transferReq := moov.CreateTransferRequest{
		Source: moov.Source{
			WalletID: req.WalletID,
		},
		Destination: moov.Destination{
			BankAccountID: req.BankAccountID,
		},
		Amount: moov.Amount{
			Value:    req.Amount,
			Currency: req.Currency,
		},
		Description: req.Description,
		Metadata:    req.Metadata,
	}

	// Call the Moov API
	transfer, resp, err := c.moovClient.CreateTransfer(ctx, transferReq)
	if err != nil {
		return nil, err
	}

	return transfer, nil
}