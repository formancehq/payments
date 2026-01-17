package bitstamp

import (
	"context"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTradableAssets(ctx context.Context, req models.GetTradableAssetsRequest) (models.GetTradableAssetsResponse, error) {
	tradingPairs, err := p.client.GetTradingPairs(ctx)
	if err != nil {
		return models.GetTradableAssetsResponse{}, fmt.Errorf("failed to get trading pairs: %w", err)
	}

	// Create a map for filtering if specific pairs are requested
	pairFilter := make(map[string]bool)
	if len(req.Pairs) > 0 {
		for _, pair := range req.Pairs {
			// Add original pair
			pairFilter[strings.ToUpper(pair)] = true
			// Add Bitstamp-normalized format
			normalized := strings.ToLower(strings.ReplaceAll(pair, "/", ""))
			pairFilter[normalized] = true
		}
	}

	assets := make([]models.TradableAsset, 0, len(tradingPairs))
	for _, pair := range tradingPairs {
		// Skip if trading is disabled
		if pair.Trading != "Enabled" {
			continue
		}

		// Skip if filtering and pair not in filter
		if len(pairFilter) > 0 {
			matched := pairFilter[strings.ToUpper(pair.Name)] ||
				pairFilter[pair.URLSymbol] ||
				pairFilter[strings.ToLower(pair.URLSymbol)]
			if !matched {
				continue
			}
		}

		// Parse the pair name to get base/quote (e.g., "BTC/USD")
		baseAsset, quoteAsset := parseBitstampPairName(pair.Name)

		assets = append(assets, models.TradableAsset{
			Pair:           pair.Name,
			BaseAsset:      baseAsset,
			QuoteAsset:     quoteAsset,
			MinOrderSize:   pair.MinimumOrder,
			MaxOrderSize:   "", // Bitstamp doesn't provide max order size in trading pairs info
			PricePrecision: pair.CounterDecimals,
			SizePrecision:  pair.BaseDecimals,
			Status:         pair.Trading,
		})
	}

	return models.GetTradableAssetsResponse{
		Assets: assets,
	}, nil
}

// parseBitstampPairName extracts base and quote assets from a Bitstamp pair name
func parseBitstampPairName(name string) (string, string) {
	// Bitstamp pair names are like "BTC/USD"
	parts := strings.Split(name, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}

	// Fallback: try common quote currencies
	quoteCurrencies := []string{"USD", "EUR", "GBP", "USDT", "USDC", "PAX"}
	for _, quote := range quoteCurrencies {
		if strings.HasSuffix(name, quote) {
			base := strings.TrimSuffix(name, quote)
			return base, quote
		}
	}

	return name, ""
}
