package client

import (
	"context"

	"github.com/coinbase-samples/prime-sdk-go/portfolios"
	"github.com/coinbase-samples/prime-sdk-go/wallets"
	"github.com/formancehq/payments/internal/connectors/metrics"
)

type Account struct {
	ID       string
	Name     string
	Metadata map[string]string
}

func (c *client) GetAccounts(ctx context.Context, page int, pageSize int) ([]*Account, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_accounts")

	// Use Coinbase Prime SDK portfolios service
	pfSvc := portfolios.NewPortfoliosService(c.sdk)
	res, err := pfSvc.ListPortfolios(ctx, &portfolios.ListPortfoliosRequest{})
	if err != nil {
		return nil, err
	}

	all := make([]*Account, 0, len(res.Portfolios))
	// Prepare wallets service once
	wlSvc := wallets.NewWalletsService(c.sdk)

	for _, p := range res.Portfolios {
		// Portfolio account
		all = append(all, &Account{
			ID:   p.Id,
			Name: p.Name,
			Metadata: map[string]string{
				"spec.coinbase.com/type":         "portfolio",
				"spec.coinbase.com/portfolio_id": p.Id,
			},
		})

		// Wallet accounts in this portfolio
		wres, err := wlSvc.ListWallets(ctx, &wallets.ListWalletsRequest{PortfolioId: p.Id})
		if err == nil {
			for _, w := range wres.Wallets {
				all = append(all, &Account{
					ID:   w.Id,
					Name: w.Name,
					Metadata: map[string]string{
						"spec.coinbase.com/type":         "wallet",
						"spec.coinbase.com/portfolio_id": p.Id,
						"spec.coinbase.com/wallet_type":  string(w.Type),
					},
				})
			}
		}
	}

	// Basic pagination slicing if caller requests pages
	if pageSize <= 0 {
		return all, nil
	}
	start := page * pageSize
	if start >= len(all) {
		return []*Account{}, nil
	}
	end := start + pageSize
	if end > len(all) {
		end = len(all)
	}
	return all[start:end], nil
}
