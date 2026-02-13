package chainbridge

import (
	"context"
	"fmt"

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

		amount, ok := b.ParseAmount()
		if !ok {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse amount %q for monitor %s", b.Amount, b.MonitorID)
		}

		balances = append(balances, models.PSPBalance{
			AccountReference: b.MonitorID,
			CreatedAt:        b.FetchedAt,
			Amount:           amount,
			Asset:            b.Asset,
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
