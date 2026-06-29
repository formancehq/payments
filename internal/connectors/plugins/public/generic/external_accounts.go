package generic

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/genericclient/v3"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/pkg/domain/pagination"
)

type externalAccountsState struct {
	LastCreatedAtFrom time.Time `json:"lastCreatedAtFrom"`
	// LastProcessedID is the reference of the last external account emitted at
	// exactly LastCreatedAtFrom, so the inclusive (>=) watermark filter excludes
	// only that already-processed row while keeping distinct same-timestamp ones.
	LastProcessedID string `json:"lastProcessedID"`
	// Page is the next page to fetch within the current LastCreatedAtFrom second.
	// It advances while the watermark second is unchanged (a same-second group
	// larger than one page) and resets to 1 once the watermark moves to a newer
	// second, so a same-second group spanning pages is walked without re-scanning
	// from page 1 each cycle (which a single LastProcessedID cannot do).
	Page int `json:"page"`
}

func (p *Plugin) fetchExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}
	if oldState.Page < 1 {
		oldState.Page = 1
	}

	newState := externalAccountsState{
		LastCreatedAtFrom: oldState.LastCreatedAtFrom,
		LastProcessedID:   oldState.LastProcessedID,
		Page:              oldState.Page,
	}

	accounts := make([]models.PSPAccount, 0)
	needMore := false
	hasMore := false
	// Resume at the persisted page and walk forward (no trim-and-restart, which
	// would skip the trimmed rows); the page cursor below records how far we got.
	page := oldState.Page
	for {
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
		page++
	}

	if len(accounts) > 0 {
		newState.LastCreatedAtFrom = accounts[len(accounts)-1].CreatedAt
		newState.LastProcessedID = accounts[len(accounts)-1].Reference
		// Same-second group still draining -> resume after consumed pages; else
		// the watermark moved to a newer second, so re-anchor at page 1.
		if newState.LastCreatedAtFrom.Equal(oldState.LastCreatedAtFrom) {
			newState.Page = page + 1
		} else {
			newState.Page = 1
		}
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
