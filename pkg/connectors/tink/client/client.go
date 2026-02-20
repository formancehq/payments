package client

import (
	"context"
	"fmt"
	"strings"

	"github.com/formancehq/payments/pkg/connector/httpwrapper"
	"github.com/formancehq/payments/pkg/connector/metrics"
	"golang.org/x/oauth2/clientcredentials"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateUser(ctx context.Context, userID string, market string, locale string) (CreateUserResponse, error)
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
			Scopes: func() []string {
				out := make([]string, len(allScopes))
				for i, s := range allScopes {
					out[i] = string(s)
				}
				return out
			}(),
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
