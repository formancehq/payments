package bitstamp

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	accountBalances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	now := time.Now().UTC()
	balances := make([]models.PSPBalance, 0, len(accountBalances))
	for _, bal := range accountBalances {
		symbol := normalizeCurrency(bal.Currency)
		precision, ok := currencies[symbol]
		if !ok {
			p.logger.Infof("skipping balance %s: unsupported currency", symbol)
			continue
		}

		amount, err := currency.GetAmountWithPrecisionFromString(bal.Available, precision)
		if err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse balance for %s: %w", symbol, err)
		}

		asset := currency.FormatAsset(currencies, symbol)
		balances = append(balances, models.PSPBalance{
			AccountReference: symbol,
			Asset:            asset,
			Amount:           amount,
			CreatedAt:        now,
		})
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
