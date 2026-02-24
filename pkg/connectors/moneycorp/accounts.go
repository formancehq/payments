package moneycorp

import (
	"context"
	"encoding/json"
	"strconv"
	"time"

	"github.com/formancehq/payments/pkg/connectors/moneycorp/client"
	"github.com/formancehq/payments/pkg/connector"
	
)

type accountsState struct {
	LastPage int `json:"lastPage"`
	// Moneycorp does not send the creation date for accounts, but we can still
	// sort by ID created (which is incremental when creating accounts).
	LastIDCreated int64 `json:"lastIDCreated"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}
	}

	if oldState.LastIDCreated == 0 {
		oldState.LastIDCreated = -1
	}

	newState := accountsState{
		LastPage:      oldState.LastPage,
		LastIDCreated: oldState.LastIDCreated,
	}

	accounts := make([]connector.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page

		pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		accounts, err = toPSPAccounts(oldState.LastIDCreated, accounts, pagedAccounts)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		needMore, hasMore = connector.ShouldFetchMore(accounts, pagedAccounts, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		id, err := strconv.ParseInt(accounts[len(accounts)-1].Reference, 10, 64)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		newState.LastIDCreated = id
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

func toPSPAccounts(
	lastIDSeen int64,
	accounts []connector.PSPAccount,
	pagedAccounts []*client.Account,
) ([]connector.PSPAccount, error) {
	for _, account := range pagedAccounts {
		id, err := strconv.ParseInt(account.ID, 10, 64)
		if err != nil {
			return accounts, err
		}

		if id <= lastIDSeen {
			continue
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return accounts, err
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference: account.ID,
			// Moneycorp does not send the opening date of the account
			CreatedAt: time.Now().UTC(),
			Name:      &account.Attributes.AccountName,
			Raw:       raw,
		})
	}
	return accounts, nil
}
