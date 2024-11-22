package client

import (
	"context"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"golang.org/x/oauth2/clientcredentials"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateIBANBankAccount(ctx context.Context, userID string, req *CreateIBANBankAccountRequest) (*BankAccount, error)
	CreateUSBankAccount(ctx context.Context, userID string, req *CreateUSBankAccountRequest) (*BankAccount, error)
	CreateCABankAccount(ctx context.Context, userID string, req *CreateCABankAccountRequest) (*BankAccount, error)
	CreateGBBankAccount(ctx context.Context, userID string, req *CreateGBBankAccountRequest) (*BankAccount, error)
	CreateOtherBankAccount(ctx context.Context, userID string, req *CreateOtherBankAccountRequest) (*BankAccount, error)
	GetBankAccounts(ctx context.Context, userID string, page, pageSize int) ([]BankAccount, error)
	GetPayin(ctx context.Context, payinID string) (*PayinResponse, error)
	InitiatePayout(ctx context.Context, payoutRequest *PayoutRequest) (*PayoutResponse, error)
	GetPayout(ctx context.Context, payoutID string) (*PayoutResponse, error)
	GetRefund(ctx context.Context, refundID string) (*Refund, error)
	GetTransactions(ctx context.Context, walletsID string, page, pageSize int, afterCreatedAt time.Time) ([]Payment, error)
	InitiateWalletTransfer(ctx context.Context, transferRequest *TransferRequest) (*TransferResponse, error)
	GetWalletTransfer(ctx context.Context, transferID string) (TransferResponse, error)
	GetUsers(ctx context.Context, page int, pageSize int) ([]User, error)
	GetWallets(ctx context.Context, userID string, page, pageSize int) ([]Wallet, error)
	GetWallet(ctx context.Context, walletID string) (*Wallet, error)
	ListAllHooks(ctx context.Context) ([]*Hook, error)
	CreateHook(ctx context.Context, eventType EventType, URL string) error
	UpdateHook(ctx context.Context, hookID string, URL string) error
}

// TODO(polo): Fetch Client wallets (FEES, ...) in the future
type client struct {
	httpClient httpwrapper.Client

	clientID string
	endpoint string
}

func New(clientID, apiKey, endpoint string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")

	config := &httpwrapper.Config{
		CommonMetricsAttributes: httpwrapper.CommonMetricsAttributesFor("mangopay"),
		OAuthConfig: &clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: apiKey,
			TokenURL:     endpoint + "/v2.01/oauth/token",
		},
	}

	return &client{
		httpClient: httpwrapper.NewClient(config),

		clientID: clientID,
		endpoint: endpoint,
	}
}
