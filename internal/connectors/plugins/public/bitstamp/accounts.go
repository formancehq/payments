package bitstamp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	// Bitstamp doesn't have a separate accounts/wallets concept.
	// Instead, we create a single "main" account to represent the Bitstamp account.
	// Balances will be fetched separately for each asset.

	accounts := []models.PSPAccount{
		{
			Reference: "main",
			Name:      strPtr("Bitstamp Main Account"),
			CreatedAt: time.Now(),
			Metadata: map[string]string{
				"provider": "bitstamp",
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
