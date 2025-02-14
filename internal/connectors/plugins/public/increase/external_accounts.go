package increase

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/increase/client"
	"github.com/formancehq/payments/internal/models"
)

type externalAccountsState struct {
	NextCursor string `json:"next_cursor"`
}

func (p *Plugin) fetchNextExternalAccounts(ctx context.Context, req models.FetchNextExternalAccountsRequest) (models.FetchNextExternalAccountsResponse, error) {
	var oldState externalAccountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextExternalAccountsResponse{}, err
		}
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	hasMore := false
	pagedRecipients, nextCursor, err := p.client.GetExternalAccounts(ctx, req.PageSize, oldState.NextCursor)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	accounts, err = fillExternalAccounts(pagedRecipients, accounts, req.PageSize)
	if err != nil {
		return models.FetchNextExternalAccountsResponse{}, err
	}

	hasMore = nextCursor != ""

	newState := externalAccountsState{
		NextCursor: nextCursor,
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

func fillExternalAccounts(
	pagedAccounts []*client.ExternalAccount,
	accounts []models.PSPAccount,
	pageSize int,
) ([]models.PSPAccount, error) {
	for _, account := range pagedAccounts {
		if len(accounts) >= pageSize {
			break
		}

		createdTime, err := time.Parse(time.RFC3339, account.CreatedAt)
		if err != nil {
			return nil, err
		}

		raw, err := json.Marshal(account)
		if err != nil {
			return nil, err
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: account.ID,
			CreatedAt: createdTime,
			Raw:       raw,
			Metadata: map[string]string{
				"type":          account.Type,
				"accountHolder": account.AccountHolder,
				"accountNumber": account.AccountNumber,
				"status":        account.Status,
				"description":   account.Description,
				"routingNumber": account.RoutingNumber,
			},
		})
	}

	return accounts, nil
}

func (p *Plugin) mapExternalAccount(
	account *client.ExternalAccount,
) (*models.PSPAccount, error) {
	createdTime, err := time.Parse(time.RFC3339, account.CreatedAt)
	if err != nil {
		return nil, err
	}

	raw, err := json.Marshal(account)
	if err != nil {
		return nil, err
	}

	pspAccount := models.PSPAccount{
		Reference: account.ID,
		CreatedAt: createdTime,
		Raw:       raw,
		Metadata: map[string]string{
			"type":          account.Type,
			"accountHolder": account.AccountHolder,
			"accountNumber": account.AccountNumber,
			"status":        account.Status,
			"description":   account.Description,
			"routingNumber": account.RoutingNumber,
		},
	}

	return &pspAccount, nil
}
