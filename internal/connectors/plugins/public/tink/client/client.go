package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"golang.org/x/oauth2/clientcredentials"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateUser(ctx context.Context, userID string, market string) (CreateUserResponse, error)
	CreateTemporaryAuthorizationCode(ctx context.Context, request CreateTemporaryCodeRequest) (CreateTemporaryCodeResponse, error)
	CreateWebhook(ctx context.Context, eventType WebhookEventType, connectorID string, url string) (CreateWebhookResponse, error)
	DeleteWebhook(ctx context.Context, webhookID string) error
	GetAccountTransactionsModifiedWebhook(ctx context.Context, payload []byte) (AccountTransactionsModifiedWebhook, error)
	GetAccountTransactionsDeletedWebhook(ctx context.Context, payload []byte) (AccountTransactionsDeletedWebhook, error)
	GetRefreshFinishedWebhook(ctx context.Context, payload []byte) (RefreshFinishedWebhook, error)
	GetAccountCreatedWebhook(ctx context.Context, payload []byte) (AccountCreatedWebhook, error)
	DeleteUserConnection(ctx context.Context, req DeleteUserConnectionRequest) error
	DeleteUser(ctx context.Context, req DeleteUserRequest) error
	ListTransactions(ctx context.Context, req ListTransactionRequest) (ListTransactionResponse, error)
	ListAccounts(ctx context.Context, userID string, nextPageToken string) (ListAccountsResponse, error)
	GetAccount(ctx context.Context, userID string, accountID string) (Account, error)
}

type client struct {
	httpClient httpwrapper.Client
	userClient httpwrapper.Client

	connectorName string
	clientID      string
	clientSecret  string
	endpoint      string
}

func New(connectorName, clientID, clientSecret, endpoint string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
		OAuthConfig: &clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     fmt.Sprintf("%s/api/v1/oauth/token", endpoint),
			Scopes: []string{
				// Authorization
				string(SCOPES_AUTHORIZATION_READ),
				string(SCOPES_AUTHORIZATION_GRANT),

				// Users
				string(SCOPES_USER_CREATE),
				string(SCOPES_USER_READ),
				string(SCOPES_USER_DELETE),

				// Consents
				string(SCOPES_CONSENTS_READONLY),

				// Providers
				string(SCOPES_PROVIDERS_READ),

				// Credentials
				string(SCOPES_CREDENTIALS_READ),
				string(SCOPES_CREDENTIALS_WRITE),
				string(SCOPES_CREDENTIALS_REFRESH),

				// Accounts
				string(SCOPES_ACCOUNTS_READ),

				// Balances
				string(SCOPES_BALANCES_READ),

				// Transactions
				string(SCOPES_TRANSACTIONS_READ),

				// Webhooks
				string(SCOPES_WEBHOOKS),
			},
		},
	}

	c := &client{
		httpClient: httpwrapper.NewClient(config),

		connectorName: connectorName,
		clientID:      clientID,
		clientSecret:  clientSecret,
		endpoint:      endpoint,
	}

	c.userClient = c.createUserHTTPClient()

	return c
}

func (c *client) createUserHTTPClient() httpwrapper.Client {
	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(c.connectorName, metrics.TransportOpts{}),
	}

	return httpwrapper.NewClient(config)
}
