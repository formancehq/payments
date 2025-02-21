package gocardless

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/gocardless/client"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/pagination"
)

type externalAccountsState struct {
	After            string    `url:"after,omitempty" json:"after,omitempty"`
	Before           string    `url:"before,omitempty" json:"before,omitempty"`
	LastCreationDate time.Time `json:"LastCreationDate"`
}

type OwnerType struct {
	Reference string `json:"reference"`
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

	if from.Reference == "" {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf("reference field is required")
	}

	if len(from.Reference) < 2 || (from.Reference[:2] != "CR" && from.Reference[:2] != "CU") {
		return models.FetchNextExternalAccountsResponse{}, fmt.Errorf(
			"ownerId field must start with 'CR' for creditor account or 'CU' customer account",
		)
	}

	newState := externalAccountsState{
		After:            oldState.After,
		Before:           oldState.Before,
		LastCreationDate: oldState.LastCreationDate,
	}

	var externalBankAccounts []models.PSPAccount
	hasMore := false
	needMore := false

	for {
		pagedExternalBankAccounts, nextCursor, err := p.client.GetExternalAccounts(ctx, from.Reference, req.PageSize, newState.After, newState.Before)

		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		newState.After = nextCursor.After
		newState.Before = nextCursor.Before

		externalBankAccounts, err = fillExternalAccounts(oldState.LastCreationDate, pagedExternalBankAccounts, externalBankAccounts)

		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		needMore, hasMore = pagination.ShouldFetchMore(externalBankAccounts, pagedExternalBankAccounts, req.PageSize)

		if !needMore || !hasMore {
			break
		}

	}

	if !needMore {
		externalBankAccounts = externalBankAccounts[:req.PageSize]
	}

	if len(externalBankAccounts) > 0 {
		newState.LastCreationDate = externalBankAccounts[len(externalBankAccounts)-1].CreatedAt
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
	lastCreatedAt time.Time,
	pagedBankAccounts []client.GocardlessGenericAccount,
	bankAccounts []models.PSPAccount,
) ([]models.PSPAccount, error) {
	for _, bankAccount := range pagedBankAccounts {
		createdAt := time.Unix(bankAccount.CreatedAt, 0)

		switch createdAt.Compare(lastCreatedAt) {
		case -1, 0:
			continue
		default:
		}

		bankAccount, err := externalAccountFromGocardlessData(bankAccount)

		if err != nil {
			return nil, err
		}

		bankAccounts = append(bankAccounts, bankAccount)
	}

	return bankAccounts, nil

}
