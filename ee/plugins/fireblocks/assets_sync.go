package fireblocks

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/ee/plugins/fireblocks/client"
)

// assetInfo is the per-asset cache entry built from /v1/assets at install /
// refresh time. The map is keyed by uppercased legacyId.
type assetInfo struct {
	// Asset is the canonical Formance asset string ("USDT/6", "ETH/18", ...).
	Asset string
	// Precision is the decimals count used to scale string amounts.
	Precision int
	// BlockchainID is captured separately (in addition to being included in
	// Metadata) so balance aggregation can join chain ids for collapsed entries.
	BlockchainID string
	// LegacyID is preserved for the same reason — needed by aggregation.
	LegacyID string
	// Metadata is the per-asset slice of MetadataPrefix-namespaced kv pairs
	// to copy onto every PSPBalance / PSPPayment that uses this asset.
	Metadata map[string]string
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

// lookupAsset resolves a Fireblocks legacyId (case-insensitive) against the
// cached asset map. Callers should treat a `false` second return as "skip
// this entry with a log" — never as a fatal error.
func (p *Plugin) lookupAsset(legacyID string) (assetInfo, bool) {
	p.assetsMu.RLock()
	defer p.assetsMu.RUnlock()
	info, ok := p.assets[strings.ToUpper(legacyID)]
	return info, ok
}

func (p *Plugin) loadAssets(ctx context.Context) error {
	rawAssets, err := p.client.ListAssets(ctx)
	if err != nil {
		return err
	}

	out := make(map[string]assetInfo, len(rawAssets))
	var skipped int
	for _, a := range rawAssets {
		info, ok := buildAssetInfo(a)
		if !ok {
			p.logger.Infof("skipping fireblocks asset legacyId=%q id=%q class=%q",
				a.LegacyID, a.ID, a.AssetClass)
			skipped++
			continue
		}
		if a.LegacyID == "" {
			// Without a legacyId vault accounts cannot reference this asset
			// (vaults use legacyIds, never UUIDs), so caching it would be dead
			// weight. Fireblocks docs explicitly say "use only the legacy ID".
			skipped++
			continue
		}
		out[strings.ToUpper(a.LegacyID)] = info
	}

	p.assetsMu.Lock()
	p.assets = out
	p.assetsLastSync = time.Now()
	p.assetsMu.Unlock()

	p.logger.Infof("loaded %d fireblocks assets (%d skipped)", len(out), skipped)
	return nil
}

// buildAssetInfo turns a Fireblocks Asset into the cached entry. Returns
// ok=false for assets that must not be ingested (deprecated, NFT/SFT/VIRTUAL,
// unknown decimals, sanitization yields empty symbol).
func buildAssetInfo(a client.Asset) (assetInfo, bool) {
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

	asset := canonicalAsset(symbol, precision)
	if asset == "" {
		return assetInfo{}, false
	}

	return assetInfo{
		Asset:        asset,
		Precision:    precision,
		BlockchainID: a.BlockchainID,
		LegacyID:     a.LegacyID,
		Metadata:     buildAssetMetadata(a),
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

// canonicalAsset assembles "<sanitized>/<precision>" (or just "<sanitized>"
// when precision is 0), and returns "" when sanitization fails so callers can
// skip cleanly.
func canonicalAsset(symbol string, precision int) string {
	s := sanitizeSymbol(symbol)
	if s == "" {
		return ""
	}
	if precision == 0 {
		return s
	}
	return fmt.Sprintf("%s/%d", s, precision)
}

// buildAssetMetadata captures the Fireblocks-side context that's worth
// surfacing alongside each balance/payment. Optional fields are omitted when
// empty / default so the map stays small.
func buildAssetMetadata(a client.Asset) map[string]string {
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

	return m
}

// boolStr renders a bool as the strings "true"/"false" used by the metadata
// payload. Pulled out for symmetry with the int/string conversions below.
func boolStr(b bool) string { return strconv.FormatBool(b) }
