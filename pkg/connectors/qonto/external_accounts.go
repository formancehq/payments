package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connectors/qonto/client"
	"github.com/formancehq/payments/pkg/connector"
	"time"

)

type externalAccountsState struct {
	LastUpdatedAt   time.Time `json:"lastUpdatedAt"`
	Page            int       `json:"page"`
	LastProcessedId string    `json:"lastProcessedId"`
}

/*
*
This is a classic implementation, using primarily lastUpdatedAt for connector. However this has an edge case, if multiple
external accounts have the same updatedAt -- if the state.lastUpdatedAt doesn't change (as in all external accounts in a
page were updated at the same time), we have to use Qonto's pagination in addition of the lastUpdatedFrom parameter.
*/
func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req connector.FetchNextExternalAccountsRequest) (connector.FetchNextExternalAccountsResponse, error) {
	if req.PageSize == 0 {
		return connector.FetchNextExternalAccountsResponse{}, connector.ErrMissingPageSize
	}
	if req.PageSize > client.QontoMaxPageSize {
		return connector.FetchNextExternalAccountsResponse{}, connector.ErrExceededMaxPageSize
	}
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			err := connector.NewWrappedError(
				fmt.Errorf("failed to unmarshall state"),
				err,
			)
			return connector.FetchNextExternalAccountsResponse{}, err
		}
	}
	if oldState.Page == 0 {
		oldState.Page = 1
	}
	newState := externalAccountsState{
		LastUpdatedAt:   oldState.LastUpdatedAt,
		Page:            oldState.Page,
		LastProcessedId: oldState.LastProcessedId,
	}

	hasMore := false
	accounts := make([]connector.PSPAccount, 0, req.PageSize)

	beneficiaries, err := p.client.GetBeneficiaries(ctx, oldState.LastUpdatedAt, oldState.Page, req.PageSize)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	accounts, err = p.beneficiaryToPSPAccounts(oldState.LastUpdatedAt, oldState.LastProcessedId, accounts, beneficiaries)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	_, hasMore = connector.ShouldFetchMore(accounts, beneficiaries, req.PageSize)

	if len(accounts) > 0 {
		var err error
		newState.LastUpdatedAt, err = time.ParseInLocation(client.QontoTimeformat, accounts[len(accounts)-1].Metadata["updated_at"], time.UTC)
		if err != nil {
			return connector.FetchNextExternalAccountsResponse{}, err
		}
		newState.LastProcessedId = accounts[len(accounts)-1].Reference
	}
	if newState.LastUpdatedAt.Equal(oldState.LastUpdatedAt) {
		newState.Page++
	} else {
		newState.Page = 1
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return connector.FetchNextExternalAccountsResponse{}, err
	}

	return connector.FetchNextExternalAccountsResponse{
		ExternalAccounts: accounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func (p *Plugin) beneficiaryToPSPAccounts(
	oldUpdatedAt time.Time,
	lastProcessedId string,
	accounts []connector.PSPAccount,
	pagedBeneficiaries []client.Beneficiary,
) ([]connector.PSPAccount, error) {
	for _, beneficiary := range pagedBeneficiaries {
		updatedAt, err := time.ParseInLocation(client.QontoTimeformat, beneficiary.UpdatedAt, time.UTC)
		if err != nil {
			err := connector.NewWrappedError(
				fmt.Errorf("invalid time format for updatedAt beneficiary"),
				err,
			)
			return accounts, err
		}
		createdAt, err := time.ParseInLocation(client.QontoTimeformat, beneficiary.CreatedAt, time.UTC)
		if err != nil {
			err := connector.NewWrappedError(
				fmt.Errorf("invalid time format for createdAt beneficiary"),
				err,
			)
			return accounts, err
		}
		raw, err := json.Marshal(beneficiary)
		if err != nil {
			return accounts, err
		}
		accountReference, err := generateAccountReference(
			beneficiary.BankAccount.AccountNumber,
			beneficiary.BankAccount.Iban,
			beneficiary.BankAccount.Bic,
			beneficiary.BankAccount.SwiftSortCode,
			beneficiary.BankAccount.RoutingNumber,
			beneficiary.Id,
		)
		if updatedAt.Before(oldUpdatedAt) || updatedAt.Equal(oldUpdatedAt) && accountReference == lastProcessedId {
			continue
		}
		if err != nil {
			p.logger.Info("mapping beneficiary to external account error: ", err)
			continue
		}
		accounts = append(accounts, connector.PSPAccount{
			Reference:    accountReference,
			CreatedAt:    createdAt,
			Name:         &beneficiary.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesForExternalAccounts, beneficiary.BankAccount.Currency)),
			Metadata: map[string]string{
				"beneficiary_id":                     beneficiary.Id,
				"bank_account_number":                beneficiary.BankAccount.AccountNumber,
				"bank_account_iban":                  beneficiary.BankAccount.Iban,
				"bank_account_bic":                   beneficiary.BankAccount.Bic,
				"bank_account_swift_sort_code":       beneficiary.BankAccount.SwiftSortCode,
				"bank_account_routing_number":        beneficiary.BankAccount.RoutingNumber,
				"bank_account_intermediary_bank_bic": beneficiary.BankAccount.IntermediaryBankBic,
				"updated_at":                         beneficiary.UpdatedAt,
			},
			Raw: raw,
		})

	}
	return accounts, nil
}

/*
*
There's no unique ID for the beneficiary's bank account, but depending on the type of bank account different fields are
populated. (see https://api-doc.qonto.com/docs/business-api/d34477c258c06-list-beneficiaries)

	Swift BIC or SEPA: iban, currency and bic will be present.
	Swift code: account_number, swift_sort_code, intermediary_bank_bic and currency will be present.
	Swift routing number: account_number, routing_number, intermediary_bank_bic and currency will be present.

We are not using the intermediary bank bic as part of the reference, as it's not part of the identity of the account
(it's just an attribute for routing transfers)
*/
func generateAccountReference(accountNumber, iban, bic, swiftSortCode, routingNumber, beneficiaryId string) (string, error) {
	switch {
	case iban != "":
		return iban + "-" + bic, nil
	case swiftSortCode != "":
		return accountNumber + "-" + swiftSortCode, nil
	case routingNumber != "":
		return accountNumber + "-" + routingNumber, nil
	default:
		return "", fmt.Errorf("invalid account, unable to generate reference for beneficiary %v", beneficiaryId)
	}
}
