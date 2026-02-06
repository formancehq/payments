package currencycloud

import (
	"context"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/pkg/connector"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req connector.FetchNextBalancesRequest) (connector.FetchNextBalancesResponse, error) {
	page := 1
	balances := make([]connector.PSPBalance, 0)
	for {
		if page < 0 {
			break
		}

		pagedBalances, nextPage, err := p.client.GetBalances(ctx, page, req.PageSize)
		if err != nil {
			return connector.FetchNextBalancesResponse{}, err
		}

		page = nextPage

		for _, balance := range pagedBalances {
			precision, ok := supportedCurrenciesWithDecimal[balance.Currency]
			if !ok {
				return connector.FetchNextBalancesResponse{}, nil
			}

			amount, err := currency.GetAmountWithPrecisionFromString(balance.Amount.String(), precision)
			if err != nil {
				return connector.FetchNextBalancesResponse{}, err
			}

			balances = append(balances, connector.PSPBalance{
				AccountReference: balance.AccountID,
				CreatedAt:        balance.UpdatedAt,
				Amount:           amount,
				Asset:            currency.FormatAsset(supportedCurrenciesWithDecimal, balance.Currency),
			})
		}
	}

	return connector.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
