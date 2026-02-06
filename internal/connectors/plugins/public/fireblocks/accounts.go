package fireblocks

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	NextCursor string `json:"nextCursor"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	resp, err := p.client.GetVaultAccountsPaged(ctx, oldState.NextCursor, int(req.PageSize))
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(resp.Accounts))
	for _, account := range resp.Accounts {
		raw, err := json.Marshal(account)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		createdAt := time.Now()
		if account.CreationDate > 0 {
			createdAt = time.UnixMilli(account.CreationDate)
		}

		accounts = append(accounts, models.PSPAccount{
			Reference: account.ID,
			CreatedAt: createdAt,
			Name:      &account.Name,
			Raw:       raw,
		})
	}

	newState := accountsState{
		NextCursor: resp.Paging.After,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	hasMore := resp.Paging.After != ""

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
