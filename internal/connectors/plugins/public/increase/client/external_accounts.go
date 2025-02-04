package client

import (
	"context"
	"fmt"

	"github.com/Increase/increase-go"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) GetExternalAccounts(ctx context.Context, page int, pageSize int) ([]*ExternalAccount, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_external_accounts")

	params := increase.ExternalAccountListParams{
		Limit: increase.F(int64(pageSize)),
	}
	if page > 0 {
		params.Cursor = increase.F(fmt.Sprintf("%d", page*pageSize))
	}

	resp, err := c.increaseClient.ExternalAccounts.List(ctx, params)
	if err != nil {
		return nil, err
	}

	accounts := make([]*ExternalAccount, len(resp.Data))
	for i, acc := range resp.Data {
		accounts[i] = &ExternalAccount{
			ID:            string(acc.ID),
			Type:          string(acc.Type),
			Status:        string(acc.Status),
			AccountNumber: acc.AccountNumber,
			RoutingNumber: acc.RoutingNumber,
		}
	}

	return accounts, nil
}
