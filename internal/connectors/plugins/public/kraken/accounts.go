package kraken

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	// Kraken doesn't have a separate accounts/wallets concept like Coinbase Prime.
	// Instead, we create a single "main" account to represent the Kraken account.
	// Balances will be fetched separately for each asset.

	accounts := []models.PSPAccount{
		{
			Reference: "main",
			Name:      strPtr("Kraken Main Account"),
			CreatedAt: time.Now(),
			Metadata: map[string]string{
				"provider": "kraken",
			},
			Raw: json.RawMessage(`{"type": "main"}`),
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
