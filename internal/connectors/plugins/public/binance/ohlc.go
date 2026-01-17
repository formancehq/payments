package binance

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchOHLC(ctx context.Context, req models.GetOHLCRequest) (models.GetOHLCResponse, error) {
	// Convert pair format from "BTC/USD" to Binance format "BTCUSD"
	symbol := convertPairForBinance(req.Pair)

	// Convert interval string to Binance interval format
	interval := intervalToBinanceInterval(req.Interval)

	// Determine limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	klines, err := p.client.GetKlines(ctx, symbol, interval, limit)
	if err != nil {
		return models.GetOHLCResponse{}, fmt.Errorf("failed to get klines: %w", err)
	}

	entries := make([]models.OHLCEntry, 0, len(klines))
	for _, kline := range klines {
		if len(kline) < 6 {
			continue
		}

		// Kline format: [openTime, open, high, low, close, volume, closeTime, ...]
		openTime, _ := kline[0].(float64)
		openStr, _ := kline[1].(string)
		highStr, _ := kline[2].(string)
		lowStr, _ := kline[3].(string)
		closeStr, _ := kline[4].(string)
		volumeStr, _ := kline[5].(string)

		open, _ := parseDecimalString(openStr, 8)
		high, _ := parseDecimalString(highStr, 8)
		low, _ := parseDecimalString(lowStr, 8)
		closePrice, _ := parseDecimalString(closeStr, 8)
		volume, _ := parseDecimalString(volumeStr, 8)

		entries = append(entries, models.OHLCEntry{
			Timestamp: time.UnixMilli(int64(openTime)).UTC(),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
		})
	}

	return models.GetOHLCResponse{
		Data: models.OHLCData{
			Pair:     req.Pair,
			Interval: req.Interval,
			Entries:  entries,
		},
	}, nil
}

// intervalToBinanceInterval converts interval string to Binance interval format
func intervalToBinanceInterval(interval string) string {
	// Binance uses same format for most intervals
	switch interval {
	case "1m", "3m", "5m", "15m", "30m":
		return interval
	case "1h":
		return "1h"
	case "2h":
		return "2h"
	case "4h":
		return "4h"
	case "6h":
		return "6h"
	case "8h":
		return "8h"
	case "12h":
		return "12h"
	case "1d":
		return "1d"
	case "3d":
		return "3d"
	case "1w":
		return "1w"
	case "1M":
		return "1M"
	default:
		return "1h" // Default to 1 hour
	}
}
