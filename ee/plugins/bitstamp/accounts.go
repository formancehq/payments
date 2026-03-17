package bitstamp

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/models"
)

// bitstampLaunchDate is used as a fixed CreatedAt for accounts since Bitstamp
// does not provide account creation dates.
var bitstampLaunchDate = time.Date(2011, 8, 2, 0, 0, 0, 0, time.UTC)

func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	accounts := make([]models.PSPAccount, 0, len(balances))
	for _, bal := range balances {
		symbol := strings.ToUpper(strings.TrimSpace(bal.Currency))
		if symbol == "" {
			continue
		}

		if _, ok := p.currencies[symbol]; !ok {
			p.logger.Infof("skipping account %s: unsupported currency", symbol)
			continue
		}

		// Skip zero-balance currencies.
		if bal.Available == "0" && bal.Total == "0" && bal.Reserved == "0" {
			continue
		}

		raw, err := json.Marshal(bal)
		if err != nil {
			return models.FetchNextAccountsResponse{}, err
		}

		defaultAsset := currency.FormatAsset(p.currencies, symbol)
		name := symbol

		accounts = append(accounts, models.PSPAccount{
			Reference:    symbol,
			CreatedAt:    bitstampLaunchDate,
			Name:         &name,
			DefaultAsset: &defaultAsset,
			Metadata: map[string]string{
				"available": bal.Available,
				"reserved":  bal.Reserved,
			},
			Raw: raw,
		})
	}

	// Bitstamp returns all balances in a single call — no pagination needed.
	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: nil,
		HasMore:  false,
	}, nil
}
