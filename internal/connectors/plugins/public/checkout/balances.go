package checkout

import (
	"context"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/go-libs/v3/currency"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	accountBalances := make([]models.PSPBalance, 0, len(balances))

	for _, b := range balances {
		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, b.Currency)

		accountBalances = append(accountBalances, models.PSPBalance{
			AccountReference: b.CurrencyAccountID,
			Asset:     		  asset,
			Amount:    		  big.NewInt(b.Available),
			CreatedAt:        time.Now().UTC(),
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: accountBalances,
		HasMore:  false,
	}, nil
}
