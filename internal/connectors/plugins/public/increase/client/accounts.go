package client

import (
	"context"

	"github.com/Increase/increase-go"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	params := &increase.AccountListParams{
		Limit: increase.F(int64(pageSize)),
	}
	if page > 0 {
		params.Cursor = increase.F(string(rune(page * pageSize)))
	}

	resp, err := c.increaseClient.Accounts.List(ctx, params)
	if err != nil {
		return nil, err
	}

	accounts := make([]*Account, len(resp.Data))
	for i, acc := range resp.Data {
		accounts[i] = &Account{
			ID:        string(acc.ID),
			Name:      acc.Name,
			Type:      string(acc.Type),
			Status:    string(acc.Status),
			Currency:  string(acc.Currency),
			Balance:   acc.Balance,
			CreatedAt: acc.CreatedAt.String(),
		}
	}

	return accounts, nil
}
