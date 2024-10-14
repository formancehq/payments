package client

import (
	"context"
	"fmt"
	"net/http"

	"github.com/adyen/adyen-go-api-library/v7/src/adyen"
	"github.com/adyen/adyen-go-api-library/v7/src/common"
	"github.com/adyen/adyen-go-api-library/v7/src/management"
	"github.com/adyen/adyen-go-api-library/v7/src/webhook"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/models"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetMerchantAccounts(ctx context.Context, pageNumber, pageSize int32) ([]management.Merchant, error)
	CreateWebhook(ctx context.Context, url string, connectorID string) error
	VerifyWebhookBasicAuth(basicAuth *models.BasicAuth) bool
	VerifyWebhookHMAC(item webhook.NotificationItem) bool
	DeleteWebhook(ctx context.Context, connectorID string) error
	TranslateWebhook(req string) (*webhook.Webhook, error)
}

type client struct {
	client *adyen.APIClient

	webhookUsername string
	webhookPassword string

	companyID string

	standardWebhook *management.Webhook
	hmacKey         string
}

func New(apiKey, username, password, companyID string, liveEndpointPrefix string) (Client, error) {
	adyenConfig := &common.Config{
		ApiKey:      apiKey,
		Environment: common.TestEnv,
		Debug:       true,
	}

	if liveEndpointPrefix != "" {
		adyenConfig.Environment = common.LiveEnv
		adyenConfig.LiveEndpointURLPrefix = liveEndpointPrefix
		adyenConfig.Debug = false
	}

	c := adyen.NewClient(adyenConfig)

	return &client{
		client:          c,
		webhookUsername: username,
		webhookPassword: password,
		companyID:       companyID,
	}, nil
}

// wrap a public error for cases that we don't want to retry
// so that activities can classify this error for temporal
func (c *client) wrapSDKError(err error, statusCode int) error {
	if statusCode >= http.StatusBadRequest && statusCode < http.StatusInternalServerError {
		return fmt.Errorf("%w: %w", httpwrapper.ErrStatusCodeClientError, err)
	}

	// Adyen SDK doesn't appear to catch anything above 500
	// let's return an error here too even if it was nil
	if statusCode >= http.StatusInternalServerError {
		return fmt.Errorf("unexpected status code %d: %w", statusCode, err)
	}
	return err
}
