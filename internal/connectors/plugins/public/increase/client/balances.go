package client

import (
	"context"

	"github.com/Increase/increase-go"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) GetAccountBalances(ctx context.Context) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	resp, err := c.increaseClient.Accounts.List(ctx, &increase.AccountListParams{})
	if err != nil {
		return nil, err
	}

	balances := make([]*Balance, len(resp.Data))
	for i, acc := range resp.Data {
		balances[i] = &Balance{
			AccountID:     string(acc.ID),
			Currency:      string(acc.Currency),
			Amount:       acc.Balance,
			LastUpdatedAt: acc.UpdatedAt.String(),
		}
	}

	return balances, nil
}
