package bankingbridge

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState workflowState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	newState := workflowState{
		Cursor: oldState.Cursor,
	}

	accounts := make([]models.PSPAccount, 0, req.PageSize)
	pagedAccounts, hasMore, cursor, err := p.client.GetAccounts(ctx, newState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	for _, acc := range pagedAccounts {
		accounts = append(accounts, models.PSPAccount{
			Reference: acc.Reference,
		})
	}

	newState.Cursor = cursor
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
