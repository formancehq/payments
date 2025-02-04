package increase

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	// TODO: externalAccountsState will be used to know at what point we're at
	// when fetching the PSP external accounts. We highly recommend to use this
	// state to not poll data already polled.
	// This struct will be stored as a raw json, you're free to put whatever
	// you want.
	// Example:
	// LastPage int `json:"lastPage"`
	// LastIDCreated int64 `json:"lastIDCreated"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	// TODO: if needed, uncomment the following lines to get the related account in request
	// var from models.PSPAccount
	// if req.FromPayload == nil {
	// 	return models.FetchNextExternalAccountsResponse{}, models.ErrMissingFromPayloadInRequest
	// }
	// if err := json.Unmarshal(req.FromPayload, &from); err != nil {
	// 	return models.FetchNextExternalAccountsResponse{}, err
	// }

	newState := externalAccountsState{
		// TODO: fill new state with old state values
	}

	needMore := false
	hasMore := false
	accounts := make([]models.PSPAccount, 0, req.PageSize)
	for /* TODO: range over pages or others */ page := 0; ; page++ {
		pagedRecipients, err := p.client.GetExternalAccounts(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		// TODO: transfer PSP object into formance object
		accounts = append(accounts, models.PSPAccount{})

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedRecipients, req.PageSize)
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
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}