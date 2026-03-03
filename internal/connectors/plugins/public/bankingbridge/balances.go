package bankingbridge

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bankingbridge/client"
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
		Cursor:             oldState.Cursor,
		LastSeenImportedAt: oldState.LastSeenImportedAt,
	}

	balances := make([]models.PSPBalance, 0, req.PageSize)
	pagedBalances, hasMore, cursor, err := p.client.GetAccountBalances(ctx, newState.Cursor, newState.LastSeenImportedAt, req.PageSize)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	for _, balance := range pagedBalances {
		balances = append(balances, ToPSPBalance(balance))
		newState.LastSeenImportedAt = balance.ImportedAt.Format(ImportedAtLayout)
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

func ToPSPBalance(in client.Balance) models.PSPBalance {
	amount := big.NewInt(in.AmountInMinors)
	return models.PSPBalance{
		AccountReference: in.AccountReference,
		Amount:           amount,
		Asset:            in.Asset,
		CreatedAt:        in.ReportedAt,
	}
}
