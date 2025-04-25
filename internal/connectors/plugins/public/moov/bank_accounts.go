package moov

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/moov/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
	"github.com/moovfinancial/moov-go/pkg/moov"
)

type bankAccountsState struct {
	AccountIndex int      `json:"account_index"`
	AccountID    string   `json:"account_id"`
	Skip         int      `json:"skip"`
	AccountIDs   []string `json:"account_ids"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState bankAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	// If we don't have account IDs yet, we need to get them from the request
	if len(oldState.AccountIDs) == 0 && req.FromPayload != nil {
		var from models.PSPOther
		if err := json.Unmarshal(req.FromPayload, &from); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		var account moov.Account
		if err := json.Unmarshal(from.Other, &account); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		oldState.AccountIDs = []string{account.ID}
	}

	// If we still don't have account IDs, return an error
	if len(oldState.AccountIDs) == 0 {
		return models.FetchNextExternalAccountsResponse{}, models.ErrMissingFromPayloadInRequest
	}

	newState := bankAccountsState{
		AccountIndex: oldState.AccountIndex,
		AccountID:    oldState.AccountID,
		Skip:         oldState.Skip,
		AccountIDs:   oldState.AccountIDs,
	}

	// If we've processed all accounts, we're done
	if newState.AccountIndex >= len(newState.AccountIDs) {
		payload, err := json.Marshal(newState)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		return models.FetchNextExternalAccountsResponse{
			ExternalAccounts: []models.PSPAccount{},
			NewState:         payload,
			HasMore:          false,
		}, nil
	}

	// Get the current account ID
	accountID := newState.AccountIDs[newState.AccountIndex]
	newState.AccountID = accountID

	externalAccounts := make([]models.PSPAccount, 0, req.PageSize)
	needMore := false
	hasMore := false

	bankAccounts, hasMoreBankAccounts, err := p.client.GetBankAccounts(ctx, accountID, newState.Skip, req.PageSize)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	for _, bankAccount := range bankAccounts {
		raw, err := json.Marshal(bankAccount)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		metadata := map[string]string{
			client.MoovAccountIDMetadataKey:                accountID,
			client.MoovBankAccountIDMetadataKey:            bankAccount.ID,
			client.MoovRoutingNumberMetadataKey:            bankAccount.RoutingNumber,
			client.MoovAccountNumberLastFourMetadataKey:    bankAccount.LastFourAccountNumber,
			client.MoovBankAccountTypeMetadataKey:          string(bankAccount.BankAccountType),
			client.MoovBankAccountHolderNameMetadataKey:    bankAccount.HolderName,
			client.MoovBankAccountHolderTypeMetadataKey:    string(bankAccount.HolderType),
		}

		externalAccounts = append(externalAccounts, models.PSPAccount{
			Reference: bankAccount.ID,
			CreatedAt: time.Now(), // Moov API doesn't provide creation time for bank accounts
			Type:      models.ACCOUNT_TYPE_EXTERNAL,
			Name:      &bankAccount.HolderName,
			Raw:       raw,
			Metadata:  metadata,
		})
	}

	needMore, hasMore = pagination.ShouldFetchMore(externalAccounts, bankAccounts, req.PageSize)
	if !needMore {
		externalAccounts = externalAccounts[:req.PageSize]
	}

	// Update state for next fetch
	if len(bankAccounts) < req.PageSize {
		// Move to the next account
		newState.AccountIndex++
		newState.Skip = 0
	} else {
		// Continue with the current account
		newState.Skip += len(bankAccounts)
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: externalAccounts,
		NewState:         payload,
		HasMore:          hasMore || hasMoreBankAccounts || newState.AccountIndex < len(newState.AccountIDs),
	}, nil
}