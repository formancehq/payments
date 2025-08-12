package client

import (
	"context"
	"strings"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateUser(context.Context) (CreateUserResponse, error)
	CreateTemporaryCode(context.Context, CreateTemporaryLinkRequest) (CreateTemporaryLinkResponse, error)

	DeleteUserConnection(ctx context.Context, req DeleteUserConnectionRequest) error
	DeleteUser(ctx context.Context, req DeleteUserRequest) error
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
}

func New(connectorName, clientID, clientSecret, configurationToken, endpoint string) (Client, error) {
	endpoint = strings.TrimSuffix(endpoint, "/")

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
	}

	return &client{
		httpClient: httpwrapper.NewClient(config),

		clientID:           clientID,
		clientSecret:       clientSecret,
		configurationToken: configurationToken,
		endpoint:           endpoint,
	}, nil
}
