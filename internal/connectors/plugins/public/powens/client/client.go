package client

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	lru "github.com/hashicorp/golang-lru/v2"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateUser(context.Context) (CreateUserResponse, error)
	CreateTemporaryLink(context.Context, CreateTemporaryLinkRequest) (CreateTemporaryLinkResponse, error)

	DeleteUserConnection(ctx context.Context, req DeleteUserConnectionRequest) error
	DeleteUser(ctx context.Context, req DeleteUserRequest) error
	GetBankAccount(ctx context.Context, accessToken string, bankAccountID int) (BankAccount, error)
	ListTransactions(ctx context.Context, accessToken string, lastUpdate time.Time, pageSize int) (TransactionResponse, error)
	DeleteWebhookAuth(ctx context.Context, id int) error

	CreateWebhookAuth(ctx context.Context, name string) (string, error)
	ListWebhookAuths(ctx context.Context) ([]WebhookAuth, error)
}

type client struct {
	httpClient httpwrapper.Client

	clientID           string
	clientSecret       string
	configurationToken string
	endpoint           string

	bankAccountsCache *lru.Cache[string, BankAccount]
}

func New(connectorName, clientID, clientSecret, configurationToken, endpoint string) (Client, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	bankAccountsCache, err := lru.New[string, BankAccount](1024)
	if err != nil {
		return nil, fmt.Errorf("failed to create bank accounts cache: %w", err)
	}

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
	}

	return &client{
		httpClient: httpwrapper.NewClient(config),

		clientID:           clientID,
		clientSecret:       clientSecret,
		configurationToken: configurationToken,
		endpoint:           endpoint,

		bankAccountsCache: bankAccountsCache,
	}, nil
}
