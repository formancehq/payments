package client

import (
	"context"

	"github.com/formancehq/payments/internal/connectors/metrics"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

func (c *client) GetWallets(ctx context.Context, accountID string) ([]moov.Wallet, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "list_wallets")

	wallets, err := c.service.GetMoovWallets(ctx, accountID)
	if err != nil {
		return nil, err
	}

	return wallets, nil
}

func (c *client) GetWallet(ctx context.Context, accountID string, walletID string) (*moov.Wallet, error) {
	ctx = context.WithValue(ctx, metrics.MetricOperationContextKey, "get_wallet")

	wallet, err := c.service.GetMoovWallet(ctx, accountID, walletID)
	if err != nil {
		return nil, err
	}

	return wallet, nil
}
