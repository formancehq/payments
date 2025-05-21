package gocardless

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
)

type externalAccountsState struct {
	After string `url:"after,omitempty" json:"after,omitempty"`
}

type OwnerType struct {
	ID string `json:"id"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (
	models.FetchNextExternalAccountsResponse, error,
) {

	var oldState externalAccountsState

	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	var from OwnerType
	if req.FromPayload == nil {
		return models.FetchNextExternalAccountsResponse{}, models.ErrMissingFromPayloadInRequest
	}

	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	if from.ID == "" {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("id field is required")
	}

	if len(from.ID) < 2 || (from.ID[:2] != "CR" && from.ID[:2] != "CU") {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf(
			"ownerId field must start with 'CR' for creditor account or 'CU' customer account",
		)
	}

	newState := externalAccountsState{
		After: oldState.After,
	}

	var externalBankAccounts []models.PSPAccount
	hasMore := false

	pagedExternalBankAccounts, nextCursor, err := p.client.GetExternalAccounts(ctx, from.ID, req.PageSize, newState.After)

	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	newState.After = nextCursor.After

	externalBankAccounts, err = fillExternalAccounts(pagedExternalBankAccounts, externalBankAccounts)

	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	hasMore = nextCursor.After != ""

	if !hasMore && len(externalBankAccounts) > 0 {
		newState.After = externalBankAccounts[len(externalBankAccounts)-1].Reference
	}

	if len(externalBankAccounts) > req.PageSize {
		externalBankAccounts = externalBankAccounts[:req.PageSize]
	}

	payload, err := json.Marshal(newState)

	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: externalBankAccounts,
		NewState:         payload,
		HasMore:          hasMore,
	}, nil
}

func fillExternalAccounts(

	pagedBankAccounts []client.GocardlessGenericAccount,
	bankAccounts []models.PSPAccount,
) ([]models.PSPAccount, error) {
	for _, bankAccount := range pagedBankAccounts {

		bankAccount, err := externalAccountFromGocardlessData(bankAccount)

		if err != nil {
			return nil, err
		}

		bankAccounts = append(bankAccounts, bankAccount)
	}

	return bankAccounts, nil

}
