package coinbaseprime

import (
	"context"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
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

// getAssets guarantees the asset cache is fresh, then returns the current
// snapshot of currencies and networkSymbols. loadAssets publishes these maps
// by pointer swap under p.assetsMu's write lock and never mutates a
// previously-published map, so callers may read the returned snapshots
// freely without holding the lock.
func (p *Plugin) getAssets(ctx context.Context) (map[string]int, map[string]string, error) {
	if err := p.ensureAssetsFresh(ctx); err != nil {
		return nil, nil, err
	}
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	return p.currencies, p.networkSymbols, nil
}

// resolveAssetAndPrecision obtains a fresh snapshot of the asset cache
// (refreshing via ensureAssetsFresh if the TTL has expired) and resolves the
// given Coinbase Prime symbol to a Formance asset string and its precision.
// Network-scoped symbols (e.g. "BASEUSDC") are folded to their base symbol
// ("USDC") via the networkSymbols map.
//
// Returns (asset, precision, true, nil) on a hit, ("", 0, false, nil) when
// the symbol is unknown in the current snapshot, and (_, _, false, err) if
// the freshness refresh itself failed (caller should propagate the error).
//
// Lives here (rather than per-capability files) because payments, orders,
// and conversions all share this resolution path.
func (p *Plugin) resolveAssetAndPrecision(ctx context.Context, symbol string) (string, int, bool, error) {
	currencies, networkSymbols, err := p.getAssets(ctx)
	if err != nil {
		return "", 0, false, err
	}

	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	// Resolve network-scoped symbols (e.g. "BASEUSDC") to their base symbol ("USDC")
	if base, ok := networkSymbols[symbol]; ok {
		symbol = base
	}

	precision, err := currency.GetPrecision(currencies, symbol)
	if err != nil {
		return "", 0, false, nil
	}

	return currency.FormatAsset(currencies, symbol), precision, true, nil
}
