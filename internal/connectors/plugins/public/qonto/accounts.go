package qonto

import (
	"context"
	"encoding/json"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	} else {
		oldState = accountsState{}
	}

	newState := accountsState{
		LastUpdatedAt: oldState.LastUpdatedAt,
	}

	var accounts []models.PSPAccount

	organization, err := p.client.GetOrganization(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}
	accounts, newState.LastUpdatedAt, err = fillAccounts(organization.BankAccounts, accounts, oldState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	/*
		TODO
		Need to check the behaviour of the framework:
		* We are always keeping the "lastUpdatedAt", I believe this gets stored to ensure we don't save/query multiple time
		the same account -- is that true? Note that the API always return all accounts, so we can't use that for filtering
		on the API side.
		* The PSPAccount only has a notion of CreatedAt, and the API has only "updatedAt". It might not be a big deal
		if the framework can support upsert by reference?
	*/
	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
	}, nil
}

func fillAccounts(
	bankAccounts []client.QontoBankAccount,
	accounts []models.PSPAccount,
	oldState accountsState,
) ([]models.PSPAccount, time.Time, error) {
	newestUpdatedAt := time.Time{}
	for _, bankAccount := range bankAccounts {

		// TODO check date format (particularly timezone -- the dates passed in are in UTC, we should save that info)
		updatedAt, err := time.Parse("2006-01-02T15:04:05.999Z", bankAccount.UpdatedAt)
		if err != nil {
			return nil, newestUpdatedAt, err
		}

		// Ignore accounts that have already been processed
		switch updatedAt.Compare(oldState.LastUpdatedAt) {
		case -1, 0:
			// Account already ingested, skip
			continue
		default:
		}
		// and update future runs with the newest processed account
		if updatedAt.Compare(newestUpdatedAt) == 1 {
			newestUpdatedAt = updatedAt
		}

		raw, err := json.Marshal(bankAccount)
		if err != nil {
			return nil, newestUpdatedAt, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    bankAccount.Id,
			CreatedAt:    updatedAt, // Qonto does not give us the createdAt info.
			Name:         &bankAccount.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, bankAccount.Currency)),
			Raw:          raw,
		})
	}

	return accounts, newestUpdatedAt, nil
}
