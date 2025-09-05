package dummypay

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

type accountsState struct {
	NextToken int `json:"nextToken"`
}

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var oldState accountsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextAccountsResponse{}, err
		}
	}

	accounts, next, err := p.client.FetchAccounts(ctx, oldState.NextToken, req.PageSize)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch accounts from client: %w", err)
	}

	newState := accountsState{
		NextToken: next,
	}
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: payload,
		HasMore:  next > 0,
	}, nil
}
