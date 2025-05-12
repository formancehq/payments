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
	CreateTemporaryCode(ctx context.Context, request CreateTemporaryCodeRequest) (CreateTemporaryCodeResponse, error)
}

type client struct {
	httpClient httpwrapper.Client

	clientID     string
	clientSecret string
	endpoint     string
}

func New(connectorName, clientID, clientSecret, endpoint string) Client {
	endpoint = strings.TrimSuffix(endpoint, "/")

	config := &httpwrapper.Config{
		Transport: metrics.NewTransport(connectorName, metrics.TransportOpts{}),
		OAuthConfig: &clientcredentials.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			TokenURL:     fmt.Sprintf("%s/api/v1/oauth/token", endpoint),
		},
	}

	return &client{
		httpClient: httpwrapper.NewClient(config),

		clientID:     clientID,
		clientSecret: clientSecret,
		endpoint:     endpoint,
	}
}
