package mangopay

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/mangopay/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	LastPage         int       `json:"lastPage"`
	LastCreationDate time.Time `json:"lastCreationDate"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	} else {
		oldState = externalAccountsState{
			// Mangopay pages start at 1
			LastPage: 1,
		}
	}

	var from client.User
	if req.FromPayload == nil {
		return models.FetchNextExternalAccountsResponse{}, errors.New("missing from payload when fetching external accounts")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	newState := externalAccountsState{
		LastPage:         oldState.LastPage,
		LastCreationDate: oldState.LastCreationDate,
	}

	var accounts []models.PSPAccount
	needMore := false
	hasMore := false
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page

		pagedExternalAccounts, err := p.client.GetBankAccounts(ctx, from.ID, page, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = fillExternalAccounts(pagedExternalAccounts, accounts, from, oldState)
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
		newState.LastCreationDate = accounts[len(accounts)-1].CreatedAt
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
	pagedExternalAccounts []client.BankAccount,
	accounts []models.PSPAccount,
	from client.User,
	oldState externalAccountsState,
) ([]models.PSPAccount, error) {
	for _, bankAccount := range pagedExternalAccounts {
		creationDate := time.Unix(bankAccount.CreationDate, 0)
		if creationDate.Before(oldState.LastCreationDate) {
			continue
		}

		raw, err := json.Marshal(bankAccount)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: bankAccount.ID,
			CreatedAt: creationDate,
			Name:      &bankAccount.OwnerName,
			Metadata: map[string]string{
				"user_id": from.ID,
			},
			Raw: raw,
		})
	}

	return accounts, nil
}
