package krakenpro

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	now := time.Now().UTC()
	name := "Kraken Pro"

	accounts := []models.PSPAccount{
		{
			Reference: p.accountRef,
			CreatedAt: now,
			Name:      &name,
			Metadata: map[string]string{
				MetadataPrefix + "provider": ProviderName,
			},
		},
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		HasMore:  false,
	}, nil
}
