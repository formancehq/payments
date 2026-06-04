package krakenpro

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/krakenpro/client"
	"github.com/formancehq/payments/ee/plugins/krakenpro/mappers"
)

// Asset / pair cache layer — read-heavy, mutated only by refreshAssets
// under p.assetsRefresh. Published maps are immutable after publication
// so readers can grab a snapshot pointer without holding the read lock.
// Mirrors the coinbaseprime layout (`assets_sync.go`) so the family-of-
// connectors mental model carries over.

// refreshAssets reloads the currencies + pairs caches from the public
// endpoints. Callers must hold p.assetsRefresh; this function builds
// fresh maps locally and atomically swaps them in under p.assetsMu's
// write lock.
func (p *Plugin) refreshAssets(ctx context.Context) error {
	assets, err := p.client.GetAssets(ctx)
	if err != nil {
		return fmt.Errorf("get assets: %w", err)
	}
	pairs, err := p.client.GetAssetPairs(ctx)
	if err != nil {
		return fmt.Errorf("get asset pairs: %w", err)
	}

	currencies := make(map[string]int, len(assets))
	assetCodes := make(map[string]string, len(assets))
	for raw, info := range assets {
		symbol := mappers.NormalizeAsset(raw)
		if symbol == "" {
			continue
		}
		// `decimals` is authoritative (`display_decimals` is UI-only). Keep
		// the largest seen per symbol since suffix-family rows can differ
		// and the engine stores one fixed precision per asset.
		if existing, ok := currencies[symbol]; !ok || existing < info.Decimals {
			currencies[symbol] = info.Decimals
		}
		// The suffix-free /Assets key is the spot code (XXBT, ADA) — the
		// deterministic spot account reference even when BalanceEx only
		// returns an earn variant.
		if !mappers.HasSuffixFamily(raw) {
			if _, ok := assetCodes[symbol]; !ok {
				assetCodes[symbol] = strings.ToUpper(strings.TrimSpace(raw))
			}
		}
	}

	p.assetsMu.Lock()
	p.currencies = currencies
	p.assetPairs = pairs
	p.assetCodes = assetCodes
	p.assetsLoaded = time.Now()
	p.assetsMu.Unlock()
	p.logger.Infof("loaded %d Kraken currencies, %d pairs", len(currencies), len(pairs))
	return nil
}

// ensureAssets refreshes the asset cache at most once per TTL and
// returns snapshots of the caches. Double-checked-locking pattern
// matches the Bitstamp / Coinbase Prime precedent.
func (p *Plugin) ensureAssets(ctx context.Context) (map[string]int, map[string]client.AssetPair, error) {
	if !p.needsAssetRefresh() {
		return p.snapshotAssets(), p.snapshotPairs(), nil
	}

	p.assetsRefresh.Lock()
	defer p.assetsRefresh.Unlock()

	if !p.needsAssetRefresh() {
		return p.snapshotAssets(), p.snapshotPairs(), nil
	}
	if err := p.refreshAssets(ctx); err != nil {
		return nil, nil, err
	}
	return p.snapshotAssets(), p.snapshotPairs(), nil
}

// needsAssetRefresh is the TTL check under a read lock.
func (p *Plugin) needsAssetRefresh() bool {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	return len(p.currencies) == 0 || time.Since(p.assetsLoaded) >= assetRefreshTTL
}

// snapshotAssets / snapshotPairs return the published caches. Maps are
// never mutated after publication (refreshAssets builds fresh ones)
// so callers can iterate lock-free.
func (p *Plugin) snapshotAssets() map[string]int {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	return p.currencies
}

func (p *Plugin) snapshotPairs() map[string]client.AssetPair {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	return p.assetPairs
}

// snapshotAssetCodes returns the canonical symbol → spot-code map.
func (p *Plugin) snapshotAssetCodes() map[string]string {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	return p.assetCodes
}
