package coinbaseprime

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastPage int `json:"lastPage"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	// Fetch a single SDK page based on state
	page := oldState.LastPage
	pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(pagedAccounts))
	for _, a := range pagedAccounts {
		raw, _ := json.Marshal(a)
		var namePtr *string
		if a.Name != "" {
			n := a.Name
			namePtr = &n
		}
		accounts = append(accounts, models.PSPAccount{
			Reference: a.ID,
			CreatedAt: time.Now(),
			Name:      namePtr,
			Metadata:  a.Metadata,
			Raw:       raw,
		})
	}

	hasMore := len(pagedAccounts) == req.PageSize
	newState := accountsState{LastPage: page}
	if hasMore {
		newState.LastPage = page + 1
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
