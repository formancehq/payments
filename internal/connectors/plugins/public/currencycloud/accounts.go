package currencycloud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"lastCreatedAt"`
}

func (p Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	if oldState.LastPage == 0 {
		oldState.LastPage = 1
	}

	newState := accountsState{
		LastPage:      oldState.LastPage,
		LastCreatedAt: oldState.LastCreatedAt,
	}

	var accounts []models.PSPAccount
	hasMore := false
	page := oldState.LastPage

OUTER:
	for {
		pagedAccounts, nextPage, err := p.client.GetAccounts(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		if len(pagedAccounts) == 0 {
			break
		}

		accounts, err = fillAccounts(accounts, pagedAccounts, oldState)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		switch {
		case len(accounts) > req.PageSize:
			// We have more accounts than requested, return the first `req.PageSize`
			// and set `hasMore` to true
			hasMore = true
			accounts = accounts[:req.PageSize]
			break OUTER

		case len(accounts) == req.PageSize:
			// We have exactly the number of accounts requested, return them and
			// set `hasMore` to true if it's not the last page
			if nextPage != -1 {
				// Not the last page
				hasMore = true
			}
			break OUTER

		case nextPage == -1:
			// No more accounts to fetch, and the number of accounts is less than
			// requested, return all accounts and set `hasMore` to false
			hasMore = false
			break OUTER

		}

		page = nextPage
	}

	newState.LastPage = page
	newState.LastCreatedAt = accounts[len(accounts)-1].CreatedAt

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

func fillAccounts(
	accounts []models.PSPAccount,
	pagedAccounts []*client.Account,
	oldState accountsState,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		switch account.CreatedAt.Compare(oldState.LastCreatedAt) {
		case -1, 0:
			// Account already ingested, skip
			continue
		default:
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: account.ID,
			CreatedAt: account.CreatedAt,
			Name:      &account.AccountName,
			Raw:       raw,
		})
	}

	return accounts, nil
}
