package fireblocks

import (
	"context"
	"time"
)

func (p *Plugin) ensureAssetsFresh(ctx context.Context) error {
	p.assetsMu.RLock()
	needsRefresh := p.assetDecimals == nil || time.Since(p.assetsLastSync) >= assetRefreshInterval
	p.assetsMu.RUnlock()
	if !needsRefresh {
		return nil
	}

	p.assetsRefreshMu.Lock()
	defer p.assetsRefreshMu.Unlock()

	p.assetsMu.RLock()
	needsRefresh = p.assetDecimals == nil || time.Since(p.assetsLastSync) >= assetRefreshInterval
	p.assetsMu.RUnlock()
	if !needsRefresh {
		return nil
	}

	return p.loadAssets(ctx)
}

func (p *Plugin) getAssetDecimals() map[string]int {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	return p.assetDecimals
}

func (p *Plugin) loadAssets(ctx context.Context) error {
	assets, err := p.client.ListAssets(ctx)
	if err != nil {
		return err
	}

	assetDecimals := make(map[string]int, len(assets))
	var skipped int
	for _, asset := range assets {
		if asset.LegacyID == "" && asset.ID == "" {
			p.logger.Infof("skipping asset with empty identifiers")
			skipped++
			continue
		}

		identifier := asset.LegacyID
		if identifier == "" {
			identifier = asset.ID
		}

		var decimals int
		var hasDecimals bool

		if asset.Onchain != nil {
			decimals = asset.Onchain.Decimals
			hasDecimals = true
		} else if asset.Decimals != nil {
			// For fiat assets without onchain data, use top-level decimals when provided.
			decimals = *asset.Decimals
			hasDecimals = true
		}

		if !hasDecimals {
			p.logger.Infof("skipping asset %q: no decimals information", identifier)
			skipped++
			continue
		}

		if decimals < 0 {
			p.logger.Infof("skipping asset %q: invalid decimals %d", identifier, decimals)
			skipped++
			continue
		}

		if asset.LegacyID != "" {
			assetDecimals[asset.LegacyID] = decimals
		}
		if asset.ID != "" {
			assetDecimals[asset.ID] = decimals
		}
	}

	p.assetsMu.Lock()
	p.assetDecimals = assetDecimals
	p.assetsLastSync = time.Now()
	p.assetsMu.Unlock()

	p.logger.Infof("loaded %d assets from Fireblocks (%d skipped)", len(assetDecimals), skipped)
	return nil
}
