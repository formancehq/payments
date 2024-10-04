package moneycorp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moneycorp/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type accountsState struct {
	LastPage int `json:"lastPage"`
	// Moneycorp does not send the creation date for accounts, but we can still
	// sort by ID created (which is incremental when creating accounts).
	LastIDCreated string `json:"lastIDCreated"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		LastPage:      oldState.LastPage,
		LastIDCreated: oldState.LastIDCreated,
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	hasMore := false
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page

		pagedAccounts, err := p.client.GetAccounts(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
		if len(pagedAccounts) == 0 {
			break
		}

		accounts, err = toPSPAccounts(oldState.LastIDCreated, accounts, pagedAccounts)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
		if len(accounts) == 0 {
			continue
		}
		newState.LastIDCreated = accounts[len(accounts)-1].Reference

		needMore := true
		needMore, hasMore, accounts = pagination.ShouldFetchMore(accounts, page, req.PageSize)
		if !needMore {
			break
		}
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

func toPSPAccounts(
	lastIDSeen string,
	accounts []models.PSPAccount,
	pagedAccounts []*client.Account,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		if account.ID <= lastIDSeen {
			continue
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return accounts, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: account.ID,
			// Moneycorp does not send the opening date of the account
			CreatedAt: time.Now().UTC(),
			Name:      &account.Attributes.AccountName,
			Raw:       raw,
		})
	}
	return accounts, nil
}
