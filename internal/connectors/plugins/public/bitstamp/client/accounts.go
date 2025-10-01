package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Account struct{}

func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	// TODO: call PSP to get accounts
	return nil, nil
}
