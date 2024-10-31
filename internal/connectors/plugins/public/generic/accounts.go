package generic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type accountsState struct {
	LastCreatedAtFrom time.Time `json:"lastCreatedAtFrom"`
}

func (p Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		LastCreatedAtFrom: oldState.LastCreatedAtFrom,
	}

	accounts := make([]models.PSPAccount, 0)
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pagedAccounts, err := p.client.ListAccounts(ctx, int64(page), int64(req.PageSize), oldState.LastCreatedAtFrom)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		accounts, err = fillAccounts(pagedAccounts, accounts, oldState)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedAccounts, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		newState.LastCreatedAtFrom = accounts[len(accounts)-1].CreatedAt
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
	pagedAccounts []genericclient.Account,
	accounts []models.PSPAccount,
	oldState accountsState,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		switch account.CreatedAt.Compare(oldState.LastCreatedAtFrom) {
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
			Reference: account.Id,
			CreatedAt: account.CreatedAt,
			Name:      &account.AccountName,
			Metadata:  account.Metadata,
			Raw:       raw,
		})
	}

	return accounts, nil
}
