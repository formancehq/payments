package binance

import (
	"context"
	"fmt"
	"strings"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchOrderBook(ctx context.Context, req models.GetOrderBookRequest) (models.GetOrderBookResponse, error) {
	// Convert pair format from "BTC/USD" to Binance format "BTCUSDT"
	symbol := convertPairForBinance(req.Pair)

	// Default depth limit
	limit := req.Depth
	if limit <= 0 {
		limit = 100
	}

	orderBookResp, err := p.client.GetOrderBook(ctx, symbol, limit)
	if err != nil {
		return models.GetOrderBookResponse{}, fmt.Errorf("failed to get order book: %w", err)
	}

	// Parse bids (buyers)
	bids := make([]models.OrderBookEntry, 0, len(orderBookResp.Bids))
	for _, bid := range orderBookResp.Bids {
		if len(bid) < 2 {
			continue
		}
		price, err := parseDecimalString(bid[0], 8)
		if err != nil {
			continue
		}
		quantity, err := parseDecimalString(bid[1], 8)
		if err != nil {
			continue
		}
		bids = append(bids, models.OrderBookEntry{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Parse asks (sellers)
	asks := make([]models.OrderBookEntry, 0, len(orderBookResp.Asks))
	for _, ask := range orderBookResp.Asks {
		if len(ask) < 2 {
			continue
		}
		price, err := parseDecimalString(ask[0], 8)
		if err != nil {
			continue
		}
		quantity, err := parseDecimalString(ask[1], 8)
		if err != nil {
			continue
		}
		asks = append(asks, models.OrderBookEntry{
			Price:    price,
			Quantity: quantity,
		})
	}

	return models.GetOrderBookResponse{
		OrderBook: models.OrderBook{
			Pair: req.Pair,
			Bids: bids,
			Asks: asks,
		},
	}, nil
}

// convertPairForBinance converts a standard pair format to Binance format
// e.g., "BTC/USD" -> "BTCUSD", "BTC/USDT" -> "BTCUSDT"
func convertPairForBinance(pair string) string {
	// Remove separator and convert to uppercase
	result := ""
	for _, c := range pair {
		if c != '/' && c != '-' && c != '_' {
			result += string(c)
		}
	}
	return strings.ToUpper(result)
}
