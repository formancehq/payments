package coinbaseprime

import (
	"context"
	"time"
)

// ensureAssetsFresh reloads the currencies/networkSymbols maps from
// GetPortfolio/GetAssets at most once per assetRefreshInterval. Follows the
// Fireblocks pattern (ee/plugins/fireblocks/assets_sync.go): a fast-path read
// lock check, then a refresh mutex with a second check to collapse concurrent
// callers into a single API call.
//
// Wallets are not refreshed here — they are populated at Install and then
// merged incrementally from fetchNextAccounts pagination.
func (p *Plugin) ensureAssetsFresh(ctx context.Context) error {
	p.assetsMu.RLock()
	needsRefresh := len(p.currencies) == 0 || time.Since(p.assetsLastSync) >= assetRefreshInterval
	p.assetsMu.RUnlock()
	if !needsRefresh {
		return nil
	}

	p.assetsRefreshMu.Lock()
	defer p.assetsRefreshMu.Unlock()

	p.assetsMu.RLock()
	needsRefresh = len(p.currencies) == 0 || time.Since(p.assetsLastSync) >= assetRefreshInterval
	p.assetsMu.RUnlock()
	if !needsRefresh {
		return nil
	}

	return p.loadAssets(ctx)
}
