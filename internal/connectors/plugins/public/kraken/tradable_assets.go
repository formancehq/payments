package kraken

import (
	"context"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTradableAssets(ctx context.Context, req models.GetTradableAssetsRequest) (models.GetTradableAssetsResponse, error) {
	assetPairs, err := p.client.GetAssetPairs(ctx)
	if err != nil {
		return models.GetTradableAssetsResponse{}, fmt.Errorf("failed to get asset pairs: %w", err)
	}

	// Create a map for filtering if specific pairs are requested
	// Support multiple formats: "BTC/USD", "XBTUSD", "XXBTZUSD"
	pairFilter := make(map[string]bool)
	if len(req.Pairs) > 0 {
		for _, pair := range req.Pairs {
			// Add original pair
			pairFilter[pair] = true
			// Add Kraken-normalized format (e.g., "BTC/USD" -> "XBTUSD")
			normalized := convertPairForKraken(pair)
			pairFilter[normalized] = true
		}
	}

	assets := make([]models.TradableAsset, 0, len(assetPairs))
	for pairName, assetPair := range assetPairs {
		// Skip if filtering and pair not in filter
		// Check against multiple formats: key, altname, wsname
		if len(pairFilter) > 0 {
			matched := pairFilter[pairName] || pairFilter[assetPair.Altname] || pairFilter[assetPair.WSName]
			if !matched {
				continue
			}
		}

		// Skip pairs that are not online
		if assetPair.Status != "" && assetPair.Status != "online" {
			continue
		}

		// Convert Kraken asset names to standard format
		baseAsset := krakenToStandard(assetPair.Base)
		quoteAsset := krakenToStandard(assetPair.Quote)

		// Create standard pair format (e.g., "BTC/USD")
		pair := baseAsset + "/" + quoteAsset

		// If wsname is available, use it for a cleaner format
		if assetPair.WSName != "" {
			pair = assetPair.WSName
		}

		assets = append(assets, models.TradableAsset{
			Pair:           pair,
			BaseAsset:      baseAsset,
			QuoteAsset:     quoteAsset,
			MinOrderSize:   assetPair.OrderMin,
			MaxOrderSize:   "", // Kraken doesn't provide max order size
			PricePrecision: assetPair.PairDecimals,
			SizePrecision:  assetPair.LotDecimals,
			Status:         assetPair.Status,
		})
	}

	return models.GetTradableAssetsResponse{
		Assets: assets,
	}, nil
}

// krakenToStandard converts Kraken's internal asset naming to standard names
func krakenToStandard(asset string) string {
	krakenToStandard := map[string]string{
		"XBT":   "BTC",
		"XXBT":  "BTC",
		"XDG":   "DOGE",
		"XXDG":  "DOGE",
		"ZUSD":  "USD",
		"ZEUR":  "EUR",
		"ZGBP":  "GBP",
		"ZJPY":  "JPY",
		"ZCAD":  "CAD",
		"ZAUD":  "AUD",
		"XETH":  "ETH",
		"XXRP":  "XRP",
		"XLTC":  "LTC",
		"XMLN":  "MLN",
		"XREP":  "REP",
		"XXLM":  "XLM",
		"XZEC":  "ZEC",
		"XXMR":  "XMR",
		"XETC":  "ETC",
		"DASH":  "DASH",
		"USDT":  "USDT",
		"USDC":  "USDC",
	}

	// Remove leading X or Z if present for standard 3-letter codes
	if standardName, ok := krakenToStandard[asset]; ok {
		return standardName
	}

	// If not found, strip common prefixes
	if strings.HasPrefix(asset, "X") && len(asset) == 4 {
		return asset[1:]
	}
	if strings.HasPrefix(asset, "Z") && len(asset) == 4 {
		return asset[1:]
	}

	return asset
}
