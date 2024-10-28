package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	lru "github.com/hashicorp/golang-lru/v2"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

const apiEndpoint = "https://api.wise.com"

type apiTransport struct {
	APIKey     string
	underlying http.RoundTripper
}

func (t *apiTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", t.APIKey))

	return t.underlying.RoundTrip(req)
}

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetBalance(ctx context.Context, profileID uint64, balanceID uint64) (*Balance, error)
	GetBalances(ctx context.Context, profileID uint64) ([]Balance, error)
	GetPayout(ctx context.Context, payoutID string) (*Payout, error)
	CreatePayout(ctx context.Context, quote Quote, targetAccount uint64, transactionID string) (*Payout, error)
	GetProfiles(ctx context.Context) ([]Profile, error)
	CreateQuote(ctx context.Context, profileID, currency string, amount json.Number) (Quote, error)
	GetRecipientAccounts(ctx context.Context, profileID uint64, pageSize int, seekPositionForNext uint64) (*RecipientAccountsResponse, error)
	GetRecipientAccount(ctx context.Context, accountID uint64) (*RecipientAccount, error)
	GetTransfers(ctx context.Context, profileID uint64, offset int, limit int) ([]Transfer, error)
	GetTransfer(ctx context.Context, transferID string) (*Transfer, error)
	CreateTransfer(ctx context.Context, quote Quote, targetAccount uint64, transactionID string) (*Transfer, error)
	CreateWebhook(ctx context.Context, profileID uint64, name, triggerOn, url, version string) (*WebhookSubscriptionResponse, error)
	ListWebhooksSubscription(ctx context.Context, profileID uint64) ([]WebhookSubscriptionResponse, error)
	DeleteWebhooks(ctx context.Context, profileID uint64, subscriptionID string) error
	TranslateTransferStateChangedWebhook(ctx context.Context, payload []byte) (Transfer, error)
	TranslateBalanceUpdateWebhook(ctx context.Context, payload []byte) (BalanceUpdateWebhookPayload, error)
}

type client struct {
	httpClient httpwrapper.Client

	mux                    *sync.Mutex
	recipientAccountsCache *lru.Cache[uint64, *RecipientAccount]
}

func (c *client) endpoint(path string) string {
	return fmt.Sprintf("%s/%s", apiEndpoint, path)
}

func New(apiKey string) (Client, error) {
	recipientsCache, _ := lru.New[uint64, *RecipientAccount](2048)
	config := &httpwrapper.Config{
		CommonMetricsAttributes: httpwrapper.CommonMetricsAttributesFor("wise"),
		Transport: &apiTransport{
			APIKey:     apiKey,
			underlying: otelhttp.NewTransport(http.DefaultTransport),
		},
	}

	httpClient, err := httpwrapper.NewClient(config)
	return &client{
		httpClient:             httpClient,
		mux:                    &sync.Mutex{},
		recipientAccountsCache: recipientsCache,
	}, err
}
