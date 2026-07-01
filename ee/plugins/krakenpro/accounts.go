package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/pkg/domain/models"
)

// fetchNextAccounts emits one PSPAccount per Kraken asset class in
// BalanceEx (coinbaseprime wallet-per-asset model): a spot account plus
// one per staking/earn variant, keyed by raw code with wallet_type set.
// Accounts derive purely from BalanceEx via the same inclusion predicate
// as fetchNextBalances, so a balance can never reference an account that
// wasn't emitted. AccountAssetsImportedAt (keyed by reference) de-dups
// across cycles since Kraken has no per-asset creation timestamp. See
// MAPPINGS §5.
func (p *Plugin) fetchNextAccounts(ctx context.Context, req models.FetchNextAccountsRequest) (models.FetchNextAccountsResponse, error) {
	var state accountsState
	if len(req.State) > 0 {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to unmarshal accounts state: %w", err)
		}
	}
	if state.AccountAssetsImportedAt == nil {
		state.AccountAssetsImportedAt = map[string]string{}
	}

	currencies, _, err := p.ensureAssets(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, err
	}

	entries, err := p.client.GetBalanceEx(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch balance ex: %w", err)
	}

	now := time.Now().UTC()
	accounts := make([]models.PSPAccount, 0, len(entries))

	// One account per BalanceEx variant (raw Kraken code as Reference).
	// Zero balances are NOT filtered: Kraken only returns a row for an
	// asset the account holds (or has held), and emitting every variant
	// via the same predicate as fetchNextBalances guarantees no balance
	// ever references an account that wasn't emitted (see MAPPINGS §5/§6).
	for rawCode, entry := range entries {
		if _, ok := mappers.IncludeBalanceEntry(currencies, rawCode); !ok {
			continue
		}
		ref := strings.ToUpper(strings.TrimSpace(rawCode))
		if _, already := state.AccountAssetsImportedAt[ref]; already {
			continue
		}
		account, mapErr := mappers.RawBalanceToPSPAccount(currencies, ref, entry)
		if mapErr != nil {
			p.logger.WithField("rawCode", ref).Errorf("map account: %v", mapErr)
			continue
		}
		if account == nil {
			continue
		}
		state.AccountAssetsImportedAt[ref] = now.Format(time.RFC3339)
		accounts = append(accounts, *account)
	}

	newState, err := json.Marshal(state)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to marshal accounts state: %w", err)
	}

	p.logger.WithField("emitted", len(accounts)).Infof("krakenpro fetch_accounts cycle done")
	return models.FetchNextAccountsResponse{
		Accounts: accounts,
		NewState: newState,
		HasMore:  false,
	}, nil
}
