package qonto

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/connectors/plugins/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/qonto/client"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	LastPage      int       `json:"lastPage"`
	LastUpdatedAt time.Time `json:"lastUpdatedAt"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	if req.PageSize == 0 {
		return models.FetchNextExternalAccountsResponse{}, models.ErrMissingPageSize
	}
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
		LastUpdatedAt: oldState.LastUpdatedAt,
	}

	needMore := false
	hasMore := false
	accounts := make([]models.PSPAccount, 0, req.PageSize)
	for page := oldState.LastPage; ; page++ {
		newState.LastPage = page
		pagedBeneficiaries, err := p.client.GetBeneficiaries(ctx, page, req.PageSize)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		accounts, err = p.beneficiaryToPSPAccounts(oldState.LastUpdatedAt, accounts, pagedBeneficiaries)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(accounts, pagedBeneficiaries, req.PageSize)
		if !needMore || !hasMore {
			break
		}
	}

	if !needMore {
		accounts = accounts[:req.PageSize]
	}

	if len(accounts) > 0 {
		var err error
		newState.LastUpdatedAt, err = time.ParseInLocation(client.QONTO_TIMEFORMAT, accounts[len(accounts)-1].Metadata["updated_at"], time.UTC)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
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

func (p *Plugin) beneficiaryToPSPAccounts(
	oldUpdatedAt time.Time,
	accounts []models.PSPAccount,
	pagedBeneficiaries []client.Beneficiary,
) ([]models.PSPAccount, error) {
	for _, beneficiary := range pagedBeneficiaries {
		updatedAt, err := time.ParseInLocation(client.QONTO_TIMEFORMAT, beneficiary.UpdatedAt, time.UTC)
		if err != nil {
			return accounts, err
		}
		createdAt, err := time.ParseInLocation(client.QONTO_TIMEFORMAT, beneficiary.CreatedAt, time.UTC)
		if err != nil {
			return accounts, err
		}
		raw, err := json.Marshal(beneficiary)
		if err != nil {
			return accounts, err
		}
		if updatedAt.Before(oldUpdatedAt) || updatedAt.Equal(oldUpdatedAt) {
			continue
		}

		accountReference, err := generateAccountReference(
			beneficiary.BankAccount.AccountNUmber,
			beneficiary.BankAccount.Iban,
			beneficiary.BankAccount.Bic,
			beneficiary.BankAccount.SwiftSortCode,
			beneficiary.BankAccount.RoutingNumber,
			beneficiary.ID,
		)
		if err != nil {
			p.logger.Info("mapping beneficiary to external account error: ", err)
			continue
		}
		accounts = append(accounts, models.PSPAccount{
			Reference:    accountReference,
			CreatedAt:    createdAt,
			Name:         &beneficiary.Name,
			DefaultAsset: pointer.For(currency.FormatAsset(supportedCurrenciesForExternalAccounts, beneficiary.BankAccount.Currency)),
			Metadata: map[string]string{
				"beneficiary_id":                     beneficiary.ID,
				"bank_account_number":                beneficiary.BankAccount.AccountNUmber,
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
