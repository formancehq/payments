package bankingbridge

import (
	"context"
	"encoding/json"

	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState workflowState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	newState := workflowState{
		Cursor: oldState.Cursor,
	}

	payments := make([]models.PSPPayment, 0, req.PageSize)
	pagedTrxs, hasMore, cursor, err := p.client.GetTransactions(ctx, newState.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	for _, trx := range pagedTrxs {
		payments = append(payments, models.PSPPayment{
			SourceAccountReference: pointer.For(trx.AccountReference),
		})
	}

	newState.Cursor = cursor
	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  hasMore,
	}, nil
}
