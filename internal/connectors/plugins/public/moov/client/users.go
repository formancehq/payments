package client

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (c *client) GetUsers(ctx context.Context, page int, pageSize int) ([]moov.Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_users")

	users, err := c.service.GetMoovAccounts(ctx, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to get moov accounts: %w", err)
	}

	return users, nil
}
