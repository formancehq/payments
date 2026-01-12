package binance

import (
	"context"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTradableAssets(ctx context.Context, req models.GetTradableAssetsRequest) (models.GetTradableAssetsResponse, error) {
	exchangeInfo, err := p.client.GetExchangeInfo(ctx)
	if err != nil {
		return models.GetTradableAssetsResponse{}, fmt.Errorf("failed to get exchange info: %w", err)
	}

	// Create a map for filtering if specific pairs are requested
	pairFilter := make(map[string]bool)
	if len(req.Pairs) > 0 {
		for _, pair := range req.Pairs {
			// Add original pair
			pairFilter[strings.ToUpper(pair)] = true
			// Add Binance-normalized format
			normalized := convertPairForBinance(pair)
			pairFilter[normalized] = true
		}
	}

	assets := make([]models.TradableAsset, 0, len(exchangeInfo.Symbols))
	for _, symbol := range exchangeInfo.Symbols {
		// Only include spot trading pairs
		if !symbol.IsSpotTradingAllowed {
			continue
		}

		// Skip if not trading
		if symbol.Status != "TRADING" {
			continue
		}

		// Skip if filtering and pair not in filter
		if len(pairFilter) > 0 {
			standardPair := symbol.BaseAsset + "/" + symbol.QuoteAsset
			matched := pairFilter[standardPair] ||
				pairFilter[symbol.Symbol] ||
				pairFilter[strings.ToUpper(standardPair)]
			if !matched {
				continue
			}
		}

		// Extract min order size from filters
		minOrderSize := ""
		for _, filter := range symbol.Filters {
			if filter.FilterType == "LOT_SIZE" {
				minOrderSize = filter.MinQty
				break
			}
		}

		// Create standard pair format
		pair := symbol.BaseAsset + "/" + symbol.QuoteAsset

		assets = append(assets, models.TradableAsset{
			Pair:           pair,
			BaseAsset:      symbol.BaseAsset,
			QuoteAsset:     symbol.QuoteAsset,
			MinOrderSize:   minOrderSize,
			MaxOrderSize:   "", // Can be extracted from LOT_SIZE filter if needed
			PricePrecision: symbol.QuotePrecision,
			SizePrecision:  symbol.BaseAssetPrecision,
			Status:         symbol.Status,
		})
	}

	return models.GetTradableAssetsResponse{
		Assets: assets,
	}, nil
}
