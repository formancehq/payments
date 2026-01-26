package coinbaseprime

import (
	"context"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTradableAssets(ctx context.Context, req models.GetTradableAssetsRequest) (models.GetTradableAssetsResponse, error) {
	products, err := p.client.GetProducts(ctx)
	if err != nil {
		return models.GetTradableAssetsResponse{}, fmt.Errorf("failed to get products: %w", err)
	}

	// Create a map for filtering if specific pairs are requested
	pairFilter := make(map[string]bool)
	if len(req.Pairs) > 0 {
		for _, pair := range req.Pairs {
			// Normalize pair format (BTC/USD -> BTC-USD)
			normalized := strings.ReplaceAll(pair, "/", "-")
			pairFilter[normalized] = true
		}
	}

	assets := make([]models.TradableAsset, 0, len(products))
	for _, product := range products {
		// Skip if filtering and product not in filter
		if len(pairFilter) > 0 && !pairFilter[product.ID] {
			continue
		}

		// Skip products that are not tradable
		if product.TradingDisabled || product.CancelOnly {
			continue
		}

		// Convert Coinbase format (BTC-USD) to standard format (BTC/USD)
		pair := strings.ReplaceAll(product.ID, "-", "/")

		// Calculate precision from increment values
		pricePrecision := countDecimalPlaces(product.QuoteIncrement)
		sizePrecision := countDecimalPlaces(product.BaseIncrement)

		assets = append(assets, models.TradableAsset{
			Pair:           pair,
			BaseAsset:      product.BaseCurrency,
			QuoteAsset:     product.QuoteCurrency,
			MinOrderSize:   product.BaseMinSize,
			MaxOrderSize:   product.BaseMaxSize,
			PricePrecision: pricePrecision,
			SizePrecision:  sizePrecision,
			Status:         product.Status,
		})
	}

	return models.GetTradableAssetsResponse{
		Assets: assets,
	}, nil
}

// countDecimalPlaces counts the number of decimal places in a string like "0.00000001"
func countDecimalPlaces(s string) int {
	if s == "" {
		return 0
	}

	parts := strings.Split(s, ".")
	if len(parts) < 2 {
		return 0
	}

	return len(parts[1])
}
