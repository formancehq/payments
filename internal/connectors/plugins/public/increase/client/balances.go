package client

import (
	"context"

	"github.com/Increase/increase-go"
)

type Balance struct {
	Available int64  `json:"available"`
	Currency  string `json:"currency"`
}

func mapBalance(b *increase.Balance) *Balance {
	return &Balance{
		Available: b.Available.MinorUnits,
		Currency:  string(b.Currency),
	}
}

func (c *client) GetAccountBalances(ctx context.Context, accountID string) ([]*Balance, error) {
	ctx = context.WithValue(ctx, "metric_operation", "get_account_balance")

	balance, err := c.sdk.Balances.Get(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return []*Balance{mapBalance(balance)}, nil
}
