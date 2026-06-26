package currencycloud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/ce/plugins/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

type accountsState struct {
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedID is the reference of the last account emitted at exactly
	// LastCreatedAt, so the inclusive (>=) watermark filter excludes only that
	// already-processed row while keeping distinct same-timestamp accounts.
	LastProcessedID string `json:"lastProcessedID"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
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
		LastPage:        oldState.LastPage,
		LastCreatedAt:   oldState.LastCreatedAt,
		LastProcessedID: oldState.LastProcessedID,
	}

	var accounts []models.PSPAccount
	hasMore := false
	page := oldState.LastPage
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
		newState.LastProcessedID = accounts[len(accounts)-1].Reference
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

func fillAccounts(
	accounts []models.PSPAccount,
	pagedAccounts []*client.Account,
	oldState accountsState,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		// Inclusive watermark: skip accounts strictly before it, and the single
		// already-processed account at exactly the watermark. Distinct accounts
		// sharing that timestamp are kept (M-CON2).
		cmp := account.CreatedAt.Compare(oldState.LastCreatedAt)
		if cmp < 0 || (cmp == 0 && account.ID == oldState.LastProcessedID) {
			continue
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
