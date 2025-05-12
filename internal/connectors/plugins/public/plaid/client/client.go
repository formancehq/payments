package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/plaid/plaid-go/v34/plaid"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateLinkToken(ctx context.Context, req CreateLinkTokenRequest) (CreateLinkTokenResponse, error)
	GetWebhookVerificationKey(ctx context.Context, kid string) (*plaid.JWKPublicKey, error)
	BaseWebhookTranslation(body []byte) (BaseWebhooks, error)
}

type client struct {
	client *plaid.APIClient

	webhookKeysCache *lru.Cache[string, *plaid.JWKPublicKey]
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

	webhookKeysCache, _ := lru.New[string, *plaid.JWKPublicKey](2048)
	configuration.HTTPClient = metrics.NewHTTPClient(name, models.DefaultConnectorClientTimeout)

	return &client{
		client:           plaid.NewAPIClient(configuration),
		webhookKeysCache: webhookKeysCache,
	}
}
