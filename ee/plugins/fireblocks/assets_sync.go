package fireblocks

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
)

// assetInfo is the per-asset cache entry built from /v1/assets at refresh time.
type assetInfo struct {
	Asset        string // canonical Formance asset, e.g. "USDT/6"
	Precision    int
	BlockchainID string
	LegacyID     string
	Metadata     map[string]string // copied onto every PSPPayment using this asset
}

func (p *Plugin) ensureAssetsFresh(ctx context.Context) error {
	p.assetsMu.RLock()
	needsRefresh := p.assets == nil || time.Since(p.assetsLastSync) >= assetRefreshInterval
	p.assetsMu.RUnlock()
	if !needsRefresh {
		return nil
	}

	p.assetsRefreshMu.Lock()
	defer p.assetsRefreshMu.Unlock()

	p.assetsMu.RLock()
	needsRefresh = p.assets == nil || time.Since(p.assetsLastSync) >= assetRefreshInterval
	p.assetsMu.RUnlock()
	if !needsRefresh {
		return nil
	}

	return p.loadAssets(ctx)
}

// lookupAsset resolves a Fireblocks legacyId (case-insensitive) against the cache.
func (p *Plugin) lookupAsset(legacyID string) (assetInfo, bool) {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	info, ok := p.assets[strings.ToUpper(legacyID)]
	return info, ok
}

func (p *Plugin) loadAssets(ctx context.Context) error {
	blockchains, err := p.client.ListBlockchains(ctx)
	if err != nil {
		return fmt.Errorf("listing blockchains: %w", err)
	}
	testnetByBlockchain := make(map[string]bool, len(blockchains))
	for _, b := range blockchains {
		if b.Onchain != nil && b.Onchain.Test {
			testnetByBlockchain[b.ID] = true
		}
	}

	rawAssets, err := p.client.ListAssets(ctx)
	if err != nil {
		return err
	}

	out := make(map[string]assetInfo, len(rawAssets))
	var skipped int
	for _, a := range rawAssets {
		info, ok := buildAssetInfo(a, testnetByBlockchain[a.BlockchainID])
		if !ok {
			p.logger.Infof("skipping fireblocks asset legacyId=%q id=%q class=%q",
				a.LegacyID, a.ID, a.AssetClass)
			skipped++
			continue
		}
		if a.LegacyID == "" {
			// Vaults reference assets by legacyId only.
			skipped++
			continue
		}
		out[strings.ToUpper(a.LegacyID)] = info
	}

	p.assetsMu.Lock()
	p.assets = out
	p.assetsLastSync = time.Now()
	p.assetsMu.Unlock()

	p.logger.Infof("loaded %d fireblocks assets (%d skipped, %d testnet blockchains)",
		len(out), skipped, len(testnetByBlockchain))
	return nil
}

// buildAssetInfo turns a Fireblocks Asset into the cached entry. Returns
// ok=false for assets that must not be ingested (deprecated, NFT/SFT/VIRTUAL,
// unknown decimals, sanitization yields empty symbol). isTestnet stamps the
// canonical asset with a `_TEST` suffix so testnet holdings never aggregate
// with their mainnet equivalents.
func buildAssetInfo(a client.Asset, isTestnet bool) (assetInfo, bool) {
	if a.Metadata != nil && a.Metadata.Deprecated {
		return assetInfo{}, false
	}

	switch a.AssetClass {
	case client.AssetClassNFT, client.AssetClassSFT, client.AssetClassVirtual:
		return assetInfo{}, false
	}

	precision, ok := pickPrecision(a)
	if !ok {
		return assetInfo{}, false
	}

	symbol := a.DisplaySymbol
	if symbol == "" && a.Onchain != nil {
		symbol = a.Onchain.Symbol
	}
	if symbol == "" {
		// Last-resort fallback so we keep some signal; legacyId is the only
		// other string guaranteed to identify the asset.
		symbol = a.LegacyID
	}

	asset := canonicalAsset(symbol, precision, isTestnet)
	if asset == "" {
		return assetInfo{}, false
	}

	return assetInfo{
		Asset:        asset,
		Precision:    precision,
		BlockchainID: a.BlockchainID,
		LegacyID:     a.LegacyID,
		Metadata:     buildAssetMetadata(a, isTestnet),
	}, true
}

// pickPrecision selects decimals based on assetClass: FIAT uses the top-level
// `decimals`, NATIVE/FT/unknown fall through to onchain. Negative or absent
// values cause a skip.
func pickPrecision(a client.Asset) (int, bool) {
	if a.AssetClass == client.AssetClassFiat {
		if a.Decimals != nil && *a.Decimals >= 0 {
			return *a.Decimals, true
		}
		// Some FIAT assets ship decimals via onchain too; try that next.
	}
	if a.Onchain != nil && a.Onchain.Decimals >= 0 {
		return a.Onchain.Decimals, true
	}
	if a.Decimals != nil && *a.Decimals >= 0 {
		return *a.Decimals, true
	}
	return 0, false
}

// sanitizeSymbol turns an arbitrary Fireblocks displaySymbol into a Ledger-
// valid base: uppercase ASCII letters/digits only, first char must be a
// letter, capped at 17 chars. Returns "" if no valid prefix can be derived.
func sanitizeSymbol(symbol string) string {
	var b strings.Builder
	b.Grow(len(symbol))
	seenLetter := false
	for _, r := range strings.ToUpper(symbol) {
		if b.Len() >= 17 {
			break
		}
		switch {
		case r >= 'A' && r <= 'Z':
			seenLetter = true
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			if seenLetter {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}

// canonicalAsset assembles "<sanitized>[_TEST]/<precision>" (or the bare base
// when precision is 0). isTestnet appends a `_TEST` segment so testnet and
// mainnet assets never collide downstream. Ledger's regex permits the base
// (capped at 17 chars by sanitizeSymbol) and the suffix independently.
func canonicalAsset(symbol string, precision int, isTestnet bool) string {
	s := sanitizeSymbol(symbol)
	if s == "" {
		return ""
	}
	if isTestnet {
		s += "_TEST"
	}
	if precision == 0 {
		return s
	}
	return fmt.Sprintf("%s/%d", s, precision)
}

// buildAssetMetadata captures the Fireblocks-side context that's worth
// surfacing alongside each balance/payment. Optional fields are omitted when
// empty / default so the map stays small.
func buildAssetMetadata(a client.Asset, isTestnet bool) map[string]string {
	m := map[string]string{}
	setIfPresent := func(key, value string) {
		if value != "" {
			m[MetadataPrefix+key] = value
		}
	}

	setIfPresent("legacy_id", a.LegacyID)
	setIfPresent("asset_uuid", a.ID)
	setIfPresent("display_name", a.DisplayName)
	setIfPresent("display_symbol", a.DisplaySymbol)
	setIfPresent("blockchain_id", a.BlockchainID)
	setIfPresent("asset_class", a.AssetClass)

	if a.Onchain != nil {
		setIfPresent("contract_address", a.Onchain.Address)
		if len(a.Onchain.Standards) > 0 {
			m[MetadataPrefix+"token_standard"] = strings.Join(a.Onchain.Standards, ",")
		}
	}

	if a.Metadata != nil {
		if a.Metadata.Verified {
			m[MetadataPrefix+"verified"] = "true"
		}
		if len(a.Metadata.Features) > 0 {
			m[MetadataPrefix+"features"] = strings.Join(a.Metadata.Features, ",")
		}
	}
	if isTestnet {
		m[MetadataPrefix+"testnet"] = "true"
	}

	return m
}
