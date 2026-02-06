package coinbase

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchNextBalances(ctx context.Context, req models.FetchNextBalancesRequest) (models.FetchNextBalancesResponse, error) {
	var from models.PSPAccount
	if req.FromPayload == nil {
		return models.FetchNextBalancesResponse{}, fmt.Errorf("missing from payload when fetching balances")
	}
	if err := json.Unmarshal(req.FromPayload, &from); err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	// Coinbase Exchange accounts already contain balance info
	// We need to fetch the account again to get the latest balance
	rawAccounts, err := p.client.GetAccounts(ctx)
	if err != nil {
		return models.FetchNextBalancesResponse{}, err
	}

	var balances []models.PSPBalance
	for _, acc := range rawAccounts {
		if acc.ID != from.Reference {
			continue
		}

		precision, ok := supportedCurrenciesWithDecimal[acc.Currency]
		if !ok {
			// Skip unsupported currencies
			break
		}

		balance, err := currency.GetAmountWithPrecisionFromString(acc.Balance, precision)
		if err != nil {
			return models.FetchNextBalancesResponse{}, fmt.Errorf("failed to parse balance for account %s: %w", acc.ID, err)
		}

		asset := currency.FormatAsset(supportedCurrenciesWithDecimal, acc.Currency)

		balances = append(balances, models.PSPBalance{
			AccountReference: acc.ID,
			Asset:            asset,
			Amount:           balance,
			CreatedAt:        time.Now().UTC(),
		})

		break
	}

	return models.FetchNextBalancesResponse{
		Balances: balances,
		HasMore:  false,
	}, nil
}
