package bitstamp

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v5/pkg/types/currency"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	accountBalances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	if len(accountBalances) == 0 {
		return models.FetchNextBalancesResponse{
			Balances: []models.PSPBalance{},
			HasMore:  false,
		}, nil
	}

	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	now := time.Now().UTC()
	balances := make([]models.PSPBalance, 0, len(accountBalances))
	for _, bal := range accountBalances {
		symbol := mappers.NormalizeCurrency(bal.Currency)
		precision, ok := currencies[symbol]
		if !ok {
			p.logger.Infof("skipping balance %s: unsupported currency", symbol)
			continue
		}

		amount, err := mappers.ParseDecimalAmount(bal.Available, precision)
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
