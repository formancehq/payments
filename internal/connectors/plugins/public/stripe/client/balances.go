package client

import (
	"context"
	"fmt"
	"time"

	"github.com/stripe/stripe-go/v79"
)

func (c *client) GetAccountBalances(ctx context.Context, accountID string) (*stripe.Balance, error) {
	start := time.Now()
	defer c.recordMetrics(ctx, start, "list_balances")

	var filters stripe.Params
	if accountID != "" {
		filters.StripeAccount = &accountID
	}

	balance, err := c.balanceClient.Get(&stripe.BalanceParams{Params: filters})
	err = wrapSDKErr(err)
	if err != nil {
		return nil, fmt.Errorf("failed to get stripe balance: %w", err)
	}
	return balance, nil
}
