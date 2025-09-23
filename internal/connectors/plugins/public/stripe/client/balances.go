package client

import (
	"context"
	"fmt"

	pluginsdkmetrics "github.com/formancehq/payments/pkg/pluginsdk/metrics"
	"github.com/stripe/stripe-go/v79"
)

func (c *client) GetAccountBalances(ctx context.Context, accountID string) (*stripe.Balance, error) {
	filters := stripe.Params{Context: pluginsdkmetrics.OperationContext(ctx, "list_balances")}
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
