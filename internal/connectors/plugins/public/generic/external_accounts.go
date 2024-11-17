package generic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/genericclient"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	LastCreatedAtFrom time.Time `json:"lastCreatedAtFrom"`
}

func (p *Plugin) fetchExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	newState := externalAccountsState{
		LastCreatedAtFrom: oldState.LastCreatedAtFrom,
	}

	accounts := make([]models.PSPAccount, 0)
	needMore := false
	hasMore := false
	for page := 0; ; page++ {
		pagedExternalAccounts, err := p.client.ListBeneficiaries(ctx, int64(page), int64(req.PageSize), oldState.LastCreatedAtFrom)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = fillExternalAccounts(pagedExternalAccounts, accounts, oldState)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedExternalAccounts, req.PageSize)
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
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func fillExternalAccounts(
	pagedExternalAccounts []genericclient.Beneficiary,
	accounts []models.PSPAccount,
	oldState externalAccountsState,
) ([]models.PSPAccount, error) {
	for _, account := range pagedExternalAccounts {
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
			Name:      &account.OwnerName,
			Metadata:  account.Metadata,
			Raw:       raw,
		})
	}

	return accounts, nil
}
