package bankingbridge

import (
	"context"
	"encoding/json"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var oldState workflowState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextBalancesResponse{}, err
		}
	}

	newState := workflowState{
		Cursor: oldState.Cursor,
	}

	balances := make([]models.PSPBalance, 0, req.PageSize)
	pagedBalances, hasMore, cursor, err := p.client.GetAccountBalances(ctx, newState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	for _, balance := range pagedBalances {
		balances = append(balances, models.PSPBalance{
			AccountReference: balance.AccountReference,
		})
	}

	newState.Cursor = cursor
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
