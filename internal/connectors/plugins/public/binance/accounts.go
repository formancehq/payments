package binance

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	// Binance doesn't have a separate accounts/wallets concept.
	// Instead, we create a single "spot" account to represent the Binance spot wallet.
	// Balances will be fetched separately for each asset.

	accounts := []models.PSPAccount{
		{
			Reference: "spot",
			Name:      strPtr("Binance Spot Account"),
			CreatedAt: time.Now(),
			Metadata: map[string]string{
				"provider": "binance",
				"type":     "spot",
			},
			Raw: json.RawMessage(`{"type": "spot"}`),
		},
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: nil,
		HasMore:  false,
	}, nil
}

func strPtr(s string) *string {
	return &s
}
