package qonto

import (
	"context"
	"encoding/json"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"github.com/formancehq/payments/internal/utils/pagination"
	"sort"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	if req.PageSize == 0 {
		return models.FetchNextAccountsResponse{}, models.ErrMissingPageSize
	}

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

	organization, err := p.client.GetOrganization(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	sortOrgBankAccountsByUpdatedAtAsc(organization)
	accounts := make([]models.PSPAccount, 0, req.PageSize)
	_, hasMore := pagination.ShouldFetchMore(accounts, organization.BankAccounts, req.PageSize)
	if hasMore {
		organization.BankAccounts = organization.BankAccounts[:req.PageSize]
	}

	accounts, newState.LastUpdatedAt, err = fillAccounts(organization.BankAccounts, accounts, oldState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
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

func sortOrgBankAccountsByUpdatedAtAsc(organization *client.Organization) {
	sort.Slice(organization.BankAccounts, func(i, j int) bool {
		updatedAtI, _ := time.ParseInLocation(client.QONTO_TIMEFORMAT, organization.BankAccounts[i].UpdatedAt, time.UTC)
		updatedAtJ, _ := time.ParseInLocation(client.QONTO_TIMEFORMAT, organization.BankAccounts[j].UpdatedAt, time.UTC)
		return updatedAtI.Before(updatedAtJ)
	})
}

func fillAccounts(
	bankAccounts []client.OrganizationBankAccount,
	accounts []models.PSPAccount,
	oldState accountsState,
) ([]models.PSPAccount, time.Time, error) {
	newestUpdatedAt := time.Time{}
	for _, bankAccount := range bankAccounts {
		updatedAt, err := time.ParseInLocation(client.QONTO_TIMEFORMAT, bankAccount.UpdatedAt, time.UTC)
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

		meta := map[string]string{
			"iban":                bankAccount.Iban,
			"bic":                 bankAccount.Bic,
			"account_number":      bankAccount.AccountNumber,
			"status":              bankAccount.Status,
			"is_external_account": strconv.FormatBool(bankAccount.IsExternalAccount),
			"main":                strconv.FormatBool(bankAccount.Main),
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    bankAccount.Id,
			CreatedAt:    updatedAt, // Qonto does not give us the createdAt info.
			Name:         &bankAccount.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesWithDecimal, bankAccount.Currency)),
			Metadata:     meta,
			Raw:          raw,
		})
	}

	return accounts, newestUpdatedAt, nil
}
