package chainbridge

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastCreatedAt time.Time `json:"lastCreatedAt"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	monitors, err := p.client.GetMonitors(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(monitors))
	for _, m := range monitors {
		if !oldState.LastCreatedAt.IsZero() && !m.CreatedAt.After(oldState.LastCreatedAt) {
			continue
		}

		raw, err := json.Marshal(m)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: m.ID,
			CreatedAt: m.CreatedAt,
			Name:      &m.Address,
			Metadata: map[string]string{
				"chain":   m.Chain,
				"address": m.Address,
				"status":  m.Status,
			},
			Raw: raw,
		})
	}

	newState := accountsState{
		LastCreatedAt: oldState.LastCreatedAt,
	}
	if len(accounts) > 0 {
		newState.LastCreatedAt = accounts[len(accounts)-1].CreatedAt
	}

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
