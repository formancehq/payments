package chainbridge

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/utils/assets"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	tokenBalances, err := p.client.GetBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	balances := make([]models.PSPBalance, 0, len(tokenBalances))
	for _, b := range tokenBalances {
		if !assets.IsValid(b.Asset) {
			continue
		}

		balances = append(balances, models.PSPBalance{
			AccountReference: b.MonitorID,
			CreatedAt:        b.FetchedAt,
			Amount:           b.Amount,
			Asset:            b.Asset,
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
