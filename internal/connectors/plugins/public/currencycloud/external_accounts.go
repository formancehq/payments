package currencycloud

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/currencycloud/client"
	"github.com/formancehq/payments/internal/models"
)

type externalAccountsState struct {
	LastPage      int       `json:"lastPage"`
	LastCreatedAt time.Time `json:"lastCreatedAt"`
}

func (p Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
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
		LastPage:      oldState.LastPage,
		LastCreatedAt: oldState.LastCreatedAt,
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

		needMore := true
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
		switch beneficiary.CreatedAt.Compare(oldState.LastCreatedAt) {
		case -1, 0:
			// Account already ingested, skip
			continue
		default:
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
