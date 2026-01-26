package routable

import (
	"context"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var accountBalances []models.PSPBalance
	for _, b := range balances {
		precision, err := currency.GetPrecision(supportedCurrenciesWithDecimal, b.Currency)
		if err != nil {
			return models.FetchNextBalancesResponse{}, err
		}

		amount, err := currency.GetAmountWithPrecisionFromString(b.AvailableAmount, precision)
		if err != nil {
			return models.FetchNextBalancesResponse{}, err
		}

		createdAt := time.Now().UTC()
		accountBalances = append(accountBalances, models.PSPBalance{
			AccountReference: "routable-balance",
			CreatedAt:        createdAt,
			Amount:           amount,
			Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, b.Currency),
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
