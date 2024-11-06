package atlar

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
	"github.com/get-momo/atlar-v1-go-client/client/external_accounts"
)

type externalAccountsState struct {
	NextToken string `json:"nextToken"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	var externalAccounts []models.PSPAccount
	nextToken := oldState.NextToken
	for {
		resp, err := p.client.GetV1ExternalAccounts(ctx, nextToken, int64(req.PageSize))
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		externalAccounts, err = p.fillExternalAccounts(ctx, resp, externalAccounts)
		if err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}

		nextToken = resp.Payload.NextToken
		if resp.Payload.NextToken == "" || len(externalAccounts) >= req.PageSize {
			break
		}
	}

	// If token is empty, this is perfect as the next polling task will refetch
	// everything ! And that's what we want since Atlar doesn't provide any
	// filters or sorting options.
	newState := externalAccountsState{
		NextToken: nextToken,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	return models.FetchNextExternalAccountsResponse{
		ExternalAccounts: externalAccounts,
		NewState:         payload,
		HasMore:          nextToken != "",
	}, nil
}

func (p *Plugin) fillExternalAccounts(
	ctx context.Context,
	pagedExternalAccounts *external_accounts.GetV1ExternalAccountsOK,
	accounts []models.PSPAccount,
) ([]models.PSPAccount, error) {
	for _, externalAccount := range pagedExternalAccounts.Payload.Items {
		resp, err := p.client.GetV1CounterpartiesID(ctx, externalAccount.CounterpartyID)
		if err != nil {
			return nil, err
		}
		counterparty := resp.Payload

		newAccount, err := ExternalAccountFromAtlarData(externalAccount, counterparty)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, newAccount)
	}

	return accounts, nil
}
