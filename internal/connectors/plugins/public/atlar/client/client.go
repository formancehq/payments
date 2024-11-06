package client

import (
	"context"
	"net/url"
	"time"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/formancehq/payments/internal/models"
	atlar_client "github.com/get-momo/atlar-v1-go-client/client"
	"github.com/get-momo/atlar-v1-go-client/client/accounts"
	"github.com/get-momo/atlar-v1-go-client/client/counterparties"
	"github.com/get-momo/atlar-v1-go-client/client/credit_transfers"
	"github.com/get-momo/atlar-v1-go-client/client/external_accounts"
	"github.com/get-momo/atlar-v1-go-client/client/third_parties"
	"github.com/get-momo/atlar-v1-go-client/client/transactions"
	atlar_models "github.com/get-momo/atlar-v1-go-client/models"
	httptransport "github.com/go-openapi/runtime/client"
	"github.com/go-openapi/strfmt"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type Client interface {
	GetV1Accounts(ctx context.Context, token string, pageSize int64) (*accounts.GetV1AccountsOK, error)
	GetV1AccountsID(ctx context.Context, id string) (*accounts.GetV1AccountsIDOK, error)

	PostV1CounterParties(ctx context.Context, newExternalBankAccount models.BankAccount) (*counterparties.PostV1CounterpartiesCreated, error)
	GetV1CounterpartiesID(ctx context.Context, counterPartyID string) (*counterparties.GetV1CounterpartiesIDOK, error)

	GetV1ExternalAccounts(ctx context.Context, token string, pageSize int64) (*external_accounts.GetV1ExternalAccountsOK, error)
	GetV1ExternalAccountsID(ctx context.Context, externalAccountID string) (*external_accounts.GetV1ExternalAccountsIDOK, error)

	GetV1BetaThirdPartiesID(ctx context.Context, id string) (*third_parties.GetV1betaThirdPartiesIDOK, error)

	GetV1Transactions(ctx context.Context, token string, pageSize int64) (*transactions.GetV1TransactionsOK, error)
	GetV1TransactionsID(ctx context.Context, id string) (*transactions.GetV1TransactionsIDOK, error)

	PostV1CreditTransfers(ctx context.Context, req *atlar_models.CreatePaymentRequest) (*credit_transfers.PostV1CreditTransfersCreated, error)
	GetV1CreditTransfersGetByExternalIDExternalID(ctx context.Context, externalID string) (*credit_transfers.GetV1CreditTransfersGetByExternalIDExternalIDOK, error)
}

type client struct {
	client                 *atlar_client.Rest
	commonMetricAttributes []attribute.KeyValue
}

func New(baseURL *url.URL, accessKey, secret string) Client {
	c := &client{
		client:                 createAtlarClient(baseURL, accessKey, secret),
		commonMetricAttributes: CommonMetricsAttributes(),
	}

	return c
}

func createAtlarClient(baseURL *url.URL, accessKey, secret string) *atlar_client.Rest {
	transport := httptransport.New(
		baseURL.Host,
		baseURL.Path,
		[]string{baseURL.Scheme},
	)
	basicAuth := httptransport.BasicAuth(accessKey, secret)
	transport.DefaultAuthentication = basicAuth
	client := atlar_client.New(transport, strfmt.Default)
	return client
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
		attribute.String("connector", "generic"),
	}
	return metricsAttributes
}
