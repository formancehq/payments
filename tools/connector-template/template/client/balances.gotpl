package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Balance struct {}

func (c *client) GetAccountBalances(ctx context.Context) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	// TODO: call PSP to have balances
	return nil, nil
}
