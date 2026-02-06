package adyen

import (
	"context"
	"encoding/json"
	"time"

	"github.com/adyen/adyen-go-api-library/v7/src/management"
	"github.com/formancehq/payments/pkg/connector"
)

type accountsState struct {
	LastPage int `json:"lastPage"`

	// Adyen API sort the accounts by ID which is the same as the name
	// and we cannot sort by other things. It means that when we fetched
	// everything, we will need to return an empty state in order to
	// refetch everything at the next polling iteration...
	// It should not change anything in the database, but it will generate
	// duplicates in events, but with the same IdempotencyKey.
	LastID string `json:"lastId"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
	}

	if oldState.LastPage == 0 {
		oldState.LastPage = 1
	}

	newState := accountsState{
		LastPage: oldState.LastPage,
		LastID:   oldState.LastID,
	}

	var accounts []connector.PSPAccount
	hasMore := false
	page := oldState.LastPage
	for {
		pagedAccount, err := p.client.GetMerchantAccounts(ctx, int32(page), int32(req.PageSize))
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		if len(pagedAccount) == 0 {
			hasMore = false
			break
		}

		accounts, err = fillAccounts(accounts, pagedAccount, oldState)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		var needMore bool
		needMore, hasMore, accounts = shouldFetchMore(accounts, pagedAccount, req.PageSize)
		if !needMore || !hasMore {
			break
		}

		page++
	}

	newState.LastPage = page
	if len(accounts) > 0 {
		newState.LastID = accounts[len(accounts)-1].Reference
	}

	if !hasMore {
		// Since the merchant accounts sorting is done by ID, if a new one is
		// created with and ID lower than the last one we fetched, we will not
		// fetch it. So we need to reset the state to fetch everything again
		// when we have fetched eveything.
		// It will not create duplicates inside the database since we're based
		// on the ID of the account, but it will create duplicates in the events
		// but with the same IdempotencyKey, so should be fine.
		newState = accountsState{}
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}

func fillAccounts(
	accounts []connector.PSPAccount,
	pagedAccount []management.Merchant,
	oldState accountsState,
) ([]connector.PSPAccount, error) {
	for _, account := range pagedAccount {
		if *account.Id <= oldState.LastID {
			continue
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference: *account.Id,
			CreatedAt: time.Now().UTC(),
			Name:      account.Name,
			Raw:       raw,
		})
	}

	return accounts, nil
}
