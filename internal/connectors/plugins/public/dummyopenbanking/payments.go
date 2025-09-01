package dummyopenbanking

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/formancehq/payments/internal/models"
)

type paymentsState struct {
	NextToken int `json:"nextToken"`
}

func (p *Plugin) fetchNextPayments(ctx context.Context, req models.FetchNextPaymentsRequest) (models.FetchNextPaymentsResponse, error) {
	var oldState paymentsState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &oldState); err != nil {
			return models.FetchNextPaymentsResponse{}, err
		}
	}

	payments, next, err := p.client.FetchPayments(ctx, oldState.NextToken, req.PageSize)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, fmt.Errorf("failed to fetch payments from client: %w", err)
	}

	newState := paymentsState{
		NextToken: next,
	}

	payload, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextPaymentsResponse{}, err
	}

	return models.FetchNextPaymentsResponse{
		Payments: payments,
		NewState: payload,
		HasMore:  next > 0,
	}, nil
}
