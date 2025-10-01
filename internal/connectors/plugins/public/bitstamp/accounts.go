package bitstamp

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type accountsState struct {
	// TODO: accountsState will be used to know at what point we're at when
	// fetching the PSP accounts.
	// This struct will be stored as a raw json, you're free to put whatever
	// you want.
	// Example:
	// LastPage int `json:"lastPage"`
	// LastIDCreated int64 `json:"lastIDCreated"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		// TODO: fill new state with old state value
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false
	for /* TODO: range over pages or others */ page := 0; ; page++ {
		pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		// TODO: transfer PSP object into formance object
		accounts = append(accounts, models.PSPAccount{})

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedAccounts, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	// TODO: don't forget to update your state accordingly
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
