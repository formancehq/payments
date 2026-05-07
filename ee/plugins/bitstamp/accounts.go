package bitstamp

import (
	"context"
	"encoding/json"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

// bitstampLaunchDate is used as a fixed CreatedAt for accounts since Bitstamp
// does not provide account creation dates.
var bitstampLaunchDate = time.Date(2011, 8, 2, 0, 0, 0, 0, time.UTC)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(balances))
	for _, bal := range balances {
		symbol := normalizeCurrency(bal.Currency)
		if symbol == "" {
			continue
		}

		// Skip zero-balance currencies.
		if isZeroAmount(bal.Available) && isZeroAmount(bal.Total) && isZeroAmount(bal.Reserved) {
			continue
		}

		raw, err := json.Marshal(bal)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		var defaultAsset *string
		if _, ok := currencies[symbol]; ok {
			asset := currency.FormatAsset(currencies, symbol)
			defaultAsset = &asset
		}

		accounts = append(accounts, models.PSPAccount{
			Reference:    symbol,
			CreatedAt:    bitstampLaunchDate,
			DefaultAsset: defaultAsset,
			Raw:          raw,
		})
	}

	// Bitstamp returns all balances in a single call — no pagination needed.
	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: nil,
		HasMore:  false,
	}, nil
}
