package bitstamp

import (
	"context"
	"fmt"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextAccounts emits one PSPAccount per Bitstamp currency with
// any non-zero balance. The full AccountBalance JSON is preserved on
// PSPAccount.Raw so the nested fetch_balances task can derive the
// per-account balance without a second API call (see balances.go).
func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	currencies, err := p.getCurrencies(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("fetch accounts: %w", err)
	}

	accounts := make([]models.PSPAccount, 0, len(balances))
	for _, bal := range balances {
		if isEmptyBalance(bal) {
			continue
		}
		account, err := mappers.AccountBalanceToPSPAccount(currencies, bal)
		if err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("map account %s: %w", bal.Currency, err)
		}
		if account == nil {
			continue
		}
		accounts = append(accounts, *account)
	}

	// Bitstamp returns all balances in a single call; no pagination
	// cursor is needed and HasMore is always false. The accounts task
	// is therefore stateless, which is exactly what the nested
	// fetch_balances Qonto pattern requires.
	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		HasMore:  false,
	}, nil
}

// isEmptyBalance reports whether a Bitstamp balance row carries no
// funds at all. We skip these here rather than in the mapper so the
// mapper stays a pure value transform (testable in isolation).
func isEmptyBalance(b client.AccountBalance) bool {
	return mappers.IsZeroAmount(b.Available) &&
		mappers.IsZeroAmount(b.Total) &&
		mappers.IsZeroAmount(b.Reserved)
}
