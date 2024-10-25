package client

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/formancehq/payments/internal/connectors/httpwrapper"
	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/stripe/stripe-go/v79"
	"github.com/stripe/stripe-go/v79/account"
	"github.com/stripe/stripe-go/v79/balance"
	"github.com/stripe/stripe-go/v79/balancetransaction"
	"github.com/stripe/stripe-go/v79/bankaccount"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

//go:generate mockgen -source client.go -destination client_generated.go -package client . Client
type Client interface {
	GetAccounts(ctx context.Context, timeline Timeline, pageSize int64) ([]*stripe.Account, Timeline, bool, error)
	GetAccountBalances(ctx context.Context, accountID string) (*stripe.Balance, error)
	GetExternalAccounts(ctx context.Context, accountID string, timeline Timeline, pageSize int64) ([]*stripe.BankAccount, Timeline, bool, error)
	GetPayments(ctx context.Context, accountID string, timeline Timeline, pageSize int64) ([]*stripe.BalanceTransaction, Timeline, bool, error)
}

type client struct {
	accountClient            account.Client
	balanceClient            balance.Client
	bankAccountClient        bankaccount.Client
	balanceTransactionClient balancetransaction.Client

	commonMetricAttributes []attribute.KeyValue
}

func New(backend stripe.Backend, apiKey string) Client {
	if backend == nil {
		backend = stripe.GetBackend(stripe.APIBackend)
	}

	return &client{
		accountClient:            account.Client{B: backend, Key: apiKey},
		balanceClient:            balance.Client{B: backend, Key: apiKey},
		bankAccountClient:        bankaccount.Client{B: backend, Key: apiKey},
		balanceTransactionClient: balancetransaction.Client{B: backend, Key: apiKey},
		commonMetricAttributes:   CommonMetricsAttributes(),
	}
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

func limit(wanted int64, have int) *int64 {
	needed := wanted - int64(have)
	return &needed
}

// wrap a public error for cases that we don't want to retry
// so that activities can classify this error for temporal
func wrapSDKErr(err error) error {
	if err == nil {
		return nil
	}

	stripeErr, ok := err.(*stripe.Error)
	if !ok {
		return err
	}

	switch stripeErr.Type {
	case stripe.ErrorTypeInvalidRequest, stripe.ErrorTypeIdempotency:
		return fmt.Errorf("%w: %w", httpwrapper.ErrStatusCodeClientError, err)

	}
	return err
}

func CommonMetricsAttributes() []attribute.KeyValue {
	metricsAttributes := []attribute.KeyValue{
		attribute.String("connector", "stripe"),
	}
	stack := os.Getenv("STACK")
	if stack != "" {
		metricsAttributes = append(metricsAttributes, attribute.String("stack", stack))
	}
	return metricsAttributes
}
