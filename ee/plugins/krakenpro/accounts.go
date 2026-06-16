package krakenpro

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
	"github.com/formancehq/payments/internal/models"
)

// fetchNextAccounts emits one PSPAccount per Kraken asset class in
// BalanceEx (coinbaseprime wallet-per-asset model): a spot account plus
// one per staking/earn variant, keyed by raw code with wallet_type set.
// The spot account is always emitted for a held symbol — even at zero
// spot balance — so order/conversion resolution always finds a trading
// wallet. AccountAssetsImportedAt (keyed by reference) de-dups across
// cycles since Kraken has no per-asset creation timestamp. See MAPPINGS §5.
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
	assetCodes := p.snapshotAssetCodes()

	entries, err := p.client.GetBalanceEx(ctx)
	if err != nil {
		return models.FetchNextAccountsResponse{}, fmt.Errorf("failed to fetch balance ex: %w", err)
	}

	now := time.Now().UTC()
	accounts := make([]models.PSPAccount, 0, len(entries))

	// spotSeen tracks which symbols already have a spot account so we
	// can guarantee one exists for every held symbol after the loop.
	spotSeen := map[string]bool{}

	emit := func(rawCode string, entry client.BalanceExEntry) {
		ref := strings.ToUpper(strings.TrimSpace(rawCode))
		if _, already := state.AccountAssetsImportedAt[ref]; already {
			if !mappers.HasSuffixFamily(ref) {
				spotSeen[mappers.NormalizeAsset(ref)] = true
			}
			return
		}
		account, mapErr := mappers.RawBalanceToPSPAccount(currencies, ref, entry)
		if mapErr != nil {
			p.logger.WithField("rawCode", ref).Errorf("map account: %v", mapErr)
			return
		}
		if account == nil {
			return
		}
		state.AccountAssetsImportedAt[ref] = now.Format(time.RFC3339)
		accounts = append(accounts, *account)
		if !mappers.HasSuffixFamily(ref) {
			spotSeen[mappers.NormalizeAsset(ref)] = true
		}
	}

	// 1. One account per BalanceEx variant. Zero balances are NOT
	//    filtered: Kraken only returns a row for an asset the account
	//    holds (or has held), and emitting every variant via the same
	//    predicate as fetchNextBalances guarantees no balance ever
	//    references an account that wasn't emitted (see MAPPINGS §5/§6).
	heldSymbols := map[string]bool{}
	for rawCode, entry := range entries {
		symbol, ok := mappers.IncludeBalanceEntry(currencies, rawCode)
		if !ok {
			continue
		}
		heldSymbols[symbol] = true
		emit(rawCode, entry)
	}

	// 2. Guarantee a spot/trading account for every held symbol, even
	//    when value sits entirely in an earn variant. The spot code is
	//    the deterministic /Assets canonical code.
	for symbol := range heldSymbols {
		if spotSeen[symbol] {
			continue
		}
		spotCode, ok := assetCodes[symbol]
		if !ok {
			continue // unknown asset; can't form a spot reference
		}
		emit(spotCode, client.BalanceExEntry{Balance: "0", HoldTrade: "0"})
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
