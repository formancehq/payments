package generic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/ce/plugins/generic/client/generated"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type externalAccountsState struct {
	LastCreatedAtFrom time.Time `json:"lastCreatedAtFrom"`
	// LastProcessedID is the reference of the last external account emitted at
	// exactly LastCreatedAtFrom, so the inclusive (>=) watermark filter excludes
	// only that already-processed row while keeping distinct same-timestamp ones.
	LastProcessedID string `json:"lastProcessedID"`
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
		LastProcessedID:   oldState.LastProcessedID,
	}

	accounts := make([]models.PSPAccount, 0)
	needMore := false
	hasMore := false
	for page := 1; ; page++ {
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
		newState.LastProcessedID = accounts[len(accounts)-1].Reference
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
		// Inclusive watermark: skip accounts strictly before it, and the single
		// already-processed account at exactly the watermark. Distinct accounts
		// sharing that timestamp are kept (M-CON2).
		cmp := account.CreatedAt.Compare(oldState.LastCreatedAtFrom)
		if cmp < 0 || (cmp == 0 && account.Id == oldState.LastProcessedID) {
			continue
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
