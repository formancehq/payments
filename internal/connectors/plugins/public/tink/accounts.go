package tink

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/tink/client"
	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	NextPageToken string `json:"nextPageToken"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := accountsState{
		NextPageToken: oldState.NextPageToken,
	}

	var from models.BankBridgeFromPayload
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	var webhook client.AccountTransactionsModifiedWebhook
	if err := json.Unmarshal(from.FromPayload, &webhook); err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	hasMore := false
	for {
		pagedAccounts, err := p.client.ListAccounts(ctx, webhook.ExternalUserID, newState.NextPageToken)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		accounts, err = toPSPAccounts(accounts, pagedAccounts.Accounts)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		newState.NextPageToken = pagedAccounts.NextPageToken
		if pagedAccounts.NextPageToken != "" {
			break
		}

		needMore := len(accounts) < req.PageSize
		hasMore = pagedAccounts.NextPageToken != ""

		if !needMore || !hasMore {
			break
		}
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

func toPSPAccounts(
	accounts []models.PSPAccount,
	pagedAccounts []client.Account,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return accounts, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: account.ID,
			CreatedAt: time.Now().UTC(),
			Name:      &account.Name,
			Raw:       raw,
		})
	}

	return accounts, nil
}
