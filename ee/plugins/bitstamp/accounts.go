package bitstamp

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextAccounts emits one PSPAccount per Bitstamp currency with
// any non-zero balance, folding in per-currency enrichment metadata
// from the install-time caches. See MAPPINGS §4.1.
func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	// Enrichment is best-effort: failures log and continue. Accounts
	// ship without the missing metadata rather than failing the cycle.
	if err := p.ensureEnrichment(ctx); err != nil {
		p.logger.WithField("error", err.Error()).
			Errorf("accounts cycle: enrichment refresh incomplete")
	}

	currencyIndex, err := p.currenciesIndex(ctx)
	if err != nil {
		p.logger.WithField("error", err.Error()).
			Errorf("currencies index refresh failed; accounts ship without networks metadata")
		currencyIndex = map[string]client.Currency{}
	}

	accounts := make([]models.PSPAccount, 0, len(balances))
	for _, bal := range balances {
		if isEmptyBalance(bal) {
			continue
		}
		symbol := mappers.NormalizeCurrency(bal.Currency)
		enrich := p.buildEnrichmentForCurrency(currencies, currencyIndex[symbol], symbol)
		account, err := mappers.AccountBalanceToPSPAccountEnriched(currencies, bal, enrich)
		if err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to map account %s: %w", bal.Currency, err)
		}
		if account == nil {
			continue
		}
		accounts = append(accounts, *account)
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		HasMore:  false,
	}, nil
}

func isEmptyBalance(b client.AccountBalance) bool {
	return mappers.IsZeroAmount(b.Available) &&
		mappers.IsZeroAmount(b.Total) &&
		mappers.IsZeroAmount(b.Reserved)
}
