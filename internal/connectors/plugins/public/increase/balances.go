package increase

import (
	"context"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) FetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var fromPayload struct {
		AccountID string `json:"accountID"`
	}
	if err := json.Unmarshal(req.FromPayload, &fromPayload); err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to unmarshal from payload: %w", err)
	}

	balances, err := p.client.GetAccountBalances(ctx, fromPayload.AccountID)
	if err != nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to get account balances: %w", err)
	}

	pspBalances := make([]models.PSPBalance, len(balances))
	for i, balance := range balances {
		raw, err := json.Marshal(balance)
		if err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to marshal balance: %w", err)
		}

		pspBalances[i] = models.PSPBalance{
			AccountID: fromPayload.AccountID,
			Currency:  balance.Currency,
			Amount:    balance.Available,
			Raw:       raw,
		}
	}

	return models.FetchNextBalancesResponse{
		Balances: pspBalances,
		HasMore:  false,
	}, nil
}
