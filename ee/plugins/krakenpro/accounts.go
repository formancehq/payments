package krakenpro

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	Fetched bool `json:"fetched"`
}

// fetchNextAccounts returns a single PSPAccount representing the Kraken Pro account.
// Phase 1: one account per configured API key.
// Phase 2 (future): enumerate sub-accounts via master key endpoint.
func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	if oldState.Fetched {
		return models.FetchNextAccountsResponse{
			Accounts: nil,
			HasMore:  false,
		}, nil
	}

	now := time.Now().UTC()
	name := "Kraken Pro"

	accounts := []models.PSPAccount{
		{
			Reference: p.accountRef,
			CreatedAt: now,
			Name:      &name,
			Metadata: map[string]string{
				"provider": ProviderName,
			},
		},
	}

	newState := accountsState{Fetched: true}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  false,
	}, nil
}
