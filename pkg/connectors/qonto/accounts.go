package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/qonto/client"
	"github.com/formancehq/payments/pkg/connector"
	"sort"
	"strconv"
	"time"

)

type accountsState struct {
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

/*
*
Few things to be aware of on this method specifically for Qonto:
* There's no way to fetch only the internal accounts, so we fetch the Organization. This returns all accounts,
without connector.
* The same endpoint returns the balances as well (which are handled in fetchNextBalance, but relying on the output of this)
*/
func (p *Plugin) fetchNextAccounts(ctx context.Context, req connector.FetchNextAccountsRequest) (connector.FetchNextAccountsResponse, error) {

	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			err := connector.NewWrappedError(
				fmt.Errorf("failed to unmarshall state"),
				err,
			)
			return connector.FetchNextAccountsResponse{}, err
		}
	} else {
		oldState = accountsState{}
	}

	newState := accountsState{
		LastUpdatedAt: oldState.LastUpdatedAt,
	}

	organization, err := p.client.GetOrganization(ctx)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	sortOrgBankAccountsByUpdatedAndIdAtAsc(organization)
	accounts := make([]connector.PSPAccount, 0, req.PageSize)

	accounts, err = fillAccounts(organization.BankAccounts, accounts, oldState)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	if len(accounts) > 0 {
		newState.LastUpdatedAt = accounts[len(accounts)-1].CreatedAt
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextAccountsResponse{}, err
	}

	return connector.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  false,
	}, nil
}

func sortOrgBankAccountsByUpdatedAndIdAtAsc(organization *client.Organization) {
	sort.Slice(organization.BankAccounts, func(i, j int) bool {
		sortKeyI := fmt.Sprintf("%s-%s", organization.BankAccounts[i].UpdatedAt, organization.BankAccounts[i].Id)
		sortKeyJ := fmt.Sprintf("%s-%s", organization.BankAccounts[j].UpdatedAt, organization.BankAccounts[j].Id)

		return sortKeyI < sortKeyJ
	})
}

func fillAccounts(
	bankAccounts []client.OrganizationBankAccount,
	accounts []connector.PSPAccount,
	oldState accountsState,
) ([]connector.PSPAccount, error) {
	for _, bankAccount := range bankAccounts {
		updatedAt, err := time.ParseInLocation(client.QontoTimeformat, bankAccount.UpdatedAt, time.UTC)
		if err != nil {
			err := connector.NewWrappedError(
				fmt.Errorf("invalid time format for bank account"),
				err,
			)
			return nil, err
		}

		// Ignore accounts that have already been processed
		if updatedAt.Before(oldState.LastUpdatedAt) {
			continue
		}

		raw, err := json.Marshal(bankAccount)
		if err != nil {
			return nil, err
		}

		meta := map[string]string{
			"bank_account_iban":   bankAccount.Iban,
			"bank_account_bic":    bankAccount.Bic,
			"bank_account_number": bankAccount.AccountNumber,
			"status":              bankAccount.Status,
			"is_external_account": strconv.FormatBool(bankAccount.IsExternalAccount),
			"main":                strconv.FormatBool(bankAccount.Main),
		}

		accounts = append(accounts, connector.PSPAccount{
			Reference:    bankAccount.Id,
			CreatedAt:    updatedAt, // Qonto does not give us the createdAt info.
			Name:         &bankAccount.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesForInternalAccounts, bankAccount.Currency)),
			Metadata:     meta,
			Raw:          raw,
		})
	}

	return accounts, nil
}
