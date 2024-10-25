package client

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/adyen/adyen-go-api-library/v7/src/adyen"
	"github.com/adyen/adyen-go-api-library/v7/src/common"
	"github.com/adyen/adyen-go-api-library/v7/src/management"
	"github.com/adyen/adyen-go-api-library/v7/src/webhook"
	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
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
	client                 *adyen.APIClient
	commonMetricAttributes []attribute.KeyValue

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
		client:                 c,
		commonMetricAttributes: CommonMetricsAttributes(),
		webhookUsername:        username,
		webhookPassword:        password,
		companyID:              companyID,
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

// recordMetrics is meant to be called in a defer
func (c *client) recordMetrics(ctx context.Context, start time.Time, operation string) {
	registry := metrics.GetMetricsRegistry()

	attrs := c.commonMetricAttributes
	attrs = append(attrs, attribute.String("operation", operation))
	opts := metric.WithAttributes(attrs...)

	registry.ConnectorPSPCalls().Add(ctx, 1, opts)
	registry.ConnectorPSPCallLatencies().Record(ctx, time.Since(start).Milliseconds(), opts)
}

func CommonMetricsAttributes() []attribute.KeyValue {
	metricsAttributes := []attribute.KeyValue{
		attribute.String("connector", "adyen"),
	}
	stack := os.Getenv("STACK")
	if stack != "" {
		metricsAttributes = append(metricsAttributes, attribute.String("stack", stack))
	}
	return metricsAttributes
}
