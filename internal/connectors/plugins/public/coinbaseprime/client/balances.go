package client

import (
	"context"
	"strings"

	cbbalances "github.com/coinbase-samples/prime-sdk-go/balances"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Balance struct {
	Symbol string
	Amount string
}

func (c *client) GetAccountBalances(ctx context.Context, accountRef string) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_account_balances")

	// Call Coinbase SDK balances service for the given portfolio/accountRef
	svc := cbbalances.NewBalancesService(c.sdk)
	res, err := svc.ListPortfolioBalances(ctx, &cbbalances.ListPortfolioBalancesRequest{PortfolioId: accountRef})
	if err != nil {
		return nil, err
	}

	out := make([]*Balance, 0, len(res.Balances))
	for _, b := range res.Balances {
		out = append(out, &Balance{
			Symbol: strings.ToUpper(b.Symbol),
			Amount: b.Amount,
		})
	}
	return out, nil
}

func (c *client) GetWalletBalance(ctx context.Context, walletId string) ([]*Balance, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_wallet_balance")
	svc := cbbalances.NewBalancesService(c.sdk)
	res, err := svc.GetWalletBalance(ctx, &cbbalances.GetWalletBalanceRequest{Id: walletId})
	if err != nil {
		return nil, err
	}
	if res == nil || res.Balance == nil {
		return []*Balance{}, nil
	}
	return []*Balance{{
		Symbol: strings.ToUpper(res.Balance.Symbol),
		Amount: res.Balance.Amount,
	}}, nil
}
