package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"github.com/plaid/plaid-go/v34/plaid"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateLinkToken(ctx context.Context, req CreateLinkTokenRequest) (CreateLinkTokenResponse, error)
}

type client struct {
	client *plaid.APIClient
}

func New(name, clientID, clientSecret string, isSandbox bool) Client {
	configuration := plaid.NewConfiguration()

	configuration.AddDefaultHeader("PLAID-CLIENT-ID", clientID)
	configuration.AddDefaultHeader("PLAID-SECRET", clientSecret)

	env := plaid.Production
	if isSandbox {
		env = plaid.Sandbox
	}
	configuration.UseEnvironment(env)

	configuration.HTTPClient = metrics.NewHTTPClient(name, models.DefaultConnectorClientTimeout)

	return &client{
		client: plaid.NewAPIClient(configuration),
	}
}
