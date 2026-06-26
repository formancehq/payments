package currencycloud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/ce/plugins/currencycloud/client"
	"github.com/formancehq/payments/pkg/domain/models"
)

type externalAccountsState struct {
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"lastCreatedAt"`
	// LastProcessedID is the reference of the last beneficiary emitted at exactly
	// LastCreatedAt, so the inclusive (>=) watermark filter excludes only that
	// already-processed row while keeping distinct same-timestamp beneficiaries.
	LastProcessedID string `json:"lastProcessedID"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	if oldState.LastPage == 0 {
		oldState.LastPage = 1
	}

	newState := externalAccountsState{
		LastPage:        oldState.LastPage,
		LastCreatedAt:   oldState.LastCreatedAt,
		LastProcessedID: oldState.LastProcessedID,
	}

	var accounts []models.PSPAccount
	hasMore := false
	page := oldState.LastPage
	for {
		pagedBeneficiaries, nextPage, err := p.client.GetBeneficiaries(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		if len(pagedBeneficiaries) == 0 {
			break
		}

		accounts, err = fillBeneficiaries(accounts, pagedBeneficiaries, oldState)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
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
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func fillBeneficiaries(
	accounts []models.PSPAccount,
	pagedBeneficiaries []*client.Beneficiary,
	oldState externalAccountsState,
) ([]models.PSPAccount, error) {
	for _, beneficiary := range pagedBeneficiaries {
		// Inclusive watermark: skip beneficiaries strictly before it, and the single
		// already-processed beneficiary at exactly the watermark. Distinct
		// beneficiaries sharing that timestamp are kept (M-CON2).
		cmp := beneficiary.CreatedAt.Compare(oldState.LastCreatedAt)
		if cmp < 0 || (cmp == 0 && beneficiary.ID == oldState.LastProcessedID) {
			continue
		}

		raw, err := json.Marshal(beneficiary)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    beneficiary.ID,
			CreatedAt:    beneficiary.CreatedAt,
			Name:         &beneficiary.Name,
			DefaultAsset: &beneficiary.Currency,
			Raw:          raw,
		})
	}

	return accounts, nil
}
