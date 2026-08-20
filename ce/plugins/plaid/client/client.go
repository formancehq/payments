package client

import (
	"context"

	"github.com/formancehq/payments/pkg/domain/httpwrapper"
	"github.com/formancehq/payments/pkg/domain/metrics"
	"github.com/formancehq/payments/pkg/domain/models"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/plaid/plaid-go/v34/plaid"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	CreateUser(ctx context.Context, userID string) (string, error)
	CreateLinkToken(ctx context.Context, req CreateLinkTokenRequest) (CreateLinkTokenResponse, error)
	UpdateLinkToken(ctx context.Context, req UpdateLinkTokenRequest) (UpdateLinkTokenResponse, error)
	ExchangePublicToken(ctx context.Context, req ExchangePublicTokenRequest) (ExchangePublicTokenResponse, error)
	GetWebhookVerificationKey(ctx context.Context, kid string) (*plaid.JWKPublicKey, error)
	BaseWebhookTranslation(body []byte) (BaseWebhooks, error)
	DeleteItem(ctx context.Context, req DeleteItemRequest) error
	DeleteUser(ctx context.Context, userToken string) error
	FormanceOpenBankingRedirect(ctx context.Context, req FormanceOpenBankingRedirectRequest) error
	ListAccounts(ctx context.Context, accessToken string) (plaid.AccountsGetResponse, error)
	ListTransactions(ctx context.Context, accessToken string, cursor string, pageSize int) (plaid.TransactionsSyncResponse, error)
	TranslateItemAddResultWebhook(body []byte) (plaid.ItemAddResultWebhook, error)
	TranslateSessionFinishedWebhook(body []byte) (plaid.LinkSessionFinishedWebhook, error)
	TranslateUserPendingDisconnectWebhook(body []byte) (plaid.PendingDisconnectWebhook, error)
	TranslateUserPendingExpirationWebhook(body []byte) (plaid.PendingExpirationWebhook, error)
	TranslateItemErrorWebhook(body []byte) (plaid.ItemErrorWebhook, error)
}

type client struct {
	client *plaid.APIClient

	formanceHTTPClient httpwrapper.Client

	webhookKeysCache *lru.Cache[string, *plaid.JWKPublicKey]
}

// TODO(polo): enable compression ? We have to activate compression directly in the http client
func New(name, clientID, clientSecret string, isSandbox bool) (Client, error) {
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

	formanceHTTPClient := httpwrapper.NewClient(&httpwrapper.Config{
		Transport: metrics.NewTransport(name, metrics.TransportOpts{}),
	})

	return &client{
		client:             plaid.NewAPIClient(configuration),
		formanceHTTPClient: formanceHTTPClient,
		webhookKeysCache:   webhookKeysCache,
	}, nil
}
