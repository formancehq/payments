package client

import (
	"context"

	"github.com/Increase/increase-go"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) GetAccountBalances(ctx context.Context) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	resp, err := c.increaseClient.Accounts.List(ctx, increase.AccountListParams{})
	if err != nil {
		return nil, err
	}

	balances := make([]*Balance, len(resp.Data))
	for i, acc := range resp.Data {
		bal, err := c.increaseClient.Accounts.Balance(ctx, string(acc.ID), increase.AccountBalanceParams{})
		if err != nil {
			return nil, err
		}
		balances[i] = &Balance{
			AccountID:     string(acc.ID),
			Currency:      string(acc.Currency),
			Amount:        bal.AvailableBalance,
			LastUpdatedAt: acc.CreatedAt.String(),
		}
	}

	return balances, nil
}
