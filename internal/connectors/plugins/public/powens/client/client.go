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
	CreateTemporaryLink(context.Context, CreateTemporaryLinkRequest) (CreateTemporaryLinkResponse, error)
	CreateWebhookAuth(ctx context.Context, connectorID string) (string, error)
}

type client struct {
	httpClient httpwrapper.Client

	clientID           string
	clientSecret       string
	configurationToken string
	endpoint           string
}

func New(connectorName, clientID, clientSecret, configurationToken, endpoint string) Client {
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
	}
}
