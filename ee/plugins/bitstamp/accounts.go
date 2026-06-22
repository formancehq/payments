package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/formancehq/payments/ee/plugins/bitstamp/client"
	"github.com/formancehq/payments/ee/plugins/bitstamp/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// fetchNextAccounts emits one PSPAccount per Bitstamp currency with
// any non-zero balance, folding in per-currency enrichment metadata
// from the in-process caches. See MAPPINGS §4.1.
func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var state accountsState
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to unmarshal accounts state: %w", err)
		}
	}
	if state.AccountCurrenciesImportedAt == nil {
		state.AccountCurrenciesImportedAt = map[string]string{}
	}

	balances, err := p.client.GetAccountBalances(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch accounts: %w", err)
	}

	unhandledBalances := make([]client.AccountBalance, 0, len(balances))
	for _, bal := range balances {
		symbol := mappers.NormalizeCurrency(bal.Currency)
		if _, ok := state.AccountCurrenciesImportedAt[symbol]; ok {
			// we've already imported this account in a previous run
			continue
		}
		if isEmptyBalance(bal) {
			continue
		}
		unhandledBalances = append(unhandledBalances, bal)
	}

	if len(unhandledBalances) == 0 {
		return models.FetchNextAccountsResponse{
			Accounts: []models.PSPAccount{},
			NewState: req.State,
			HasMore:  false,
		}, nil
	}

	currencyIndex, err := p.currenciesIndex(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	// Enrichment is best-effort: failures log and continue. Accounts
	// ship without the missing metadata rather than failing the cycle.
	enrich, err := p.fetchAccountEnrichmentData(ctx)
	if err != nil {
		p.logger.WithField("error", err.Error()).
			Errorf("accounts cycle: enrichment refresh incomplete")
	}

	accounts := make([]models.PSPAccount, 0, len(unhandledBalances))
	for _, bal := range unhandledBalances {
		symbol := mappers.NormalizeCurrency(bal.Currency)
		accountEnrich := buildEnrichmentForCurrency(enrich, currencyIndex, symbol)
		account, err := mappers.AccountBalanceToPSPAccountEnriched(currencyIndex, bal, accountEnrich)
		if err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to map account %s: %w", bal.Currency, err)
		}
		if account == nil {
			continue
		}
		state.AccountCurrenciesImportedAt[symbol] = time.Now().UTC().Format(time.RFC3339)
		accounts = append(accounts, *account)
	}

	newState, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to marshal accounts state: %w", err)
	}

	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: newState,
		HasMore:  false,
	}, nil
}

func isEmptyBalance(b client.AccountBalance) bool {
	return mappers.IsZeroAmount(b.Available) &&
		mappers.IsZeroAmount(b.Total) &&
		mappers.IsZeroAmount(b.Reserved)
}
