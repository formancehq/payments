package currencycloud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/pkg/connectors/currencycloud/client"
	"github.com/formancehq/payments/pkg/connector"
)

type accountsState struct {
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"lastCreatedAt"`
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
		LastPage:      oldState.LastPage,
		LastCreatedAt: oldState.LastCreatedAt,
	}

	var accounts []connector.PSPAccount
	hasMore := false
	page := oldState.LastPage
	for {
		pagedAccounts, nextPage, err := p.client.GetAccounts(ctx, page, req.PageSize)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		if len(pagedAccounts) == 0 {
			break
		}

		accounts, err = fillAccounts(accounts, pagedAccounts, oldState)
		if err != nil {
			return connector.FetchNextAccountsResponse{}, err
		}

		var needMore bool
		needMore, hasMore, accounts = shouldFetchMore(accounts, nextPage, req.PageSize)
		if !needMore {
			break
		}

		page = nextPage
	}

	newState.LastPage = page
	if len(accounts) > 0 {
		newState.LastCreatedAt = accounts[len(accounts)-1].CreatedAt
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
	pagedAccounts []*client.Account,
	oldState accountsState,
) ([]connector.PSPAccount, error) {
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

		accounts = append(accounts, connector.PSPAccount{
			Reference: account.ID,
			CreatedAt: account.CreatedAt,
			Name:      &account.AccountName,
			Raw:       raw,
		})
	}

	return accounts, nil
}
