package bitstamp

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchOHLC(ctx context.Context, req models.GetOHLCRequest) (models.GetOHLCResponse, error) {
	// Convert pair format from "BTC/USD" to Bitstamp format "btcusd"
	market := convertPairForBitstamp(req.Pair)

	// Convert interval string to Bitstamp step (seconds)
	step := intervalToBitstampStep(req.Interval)

	// Determine limit
	limit := req.Limit
	if limit <= 0 {
		limit = 100 // Default limit
	}

	ohlcResp, err := p.client.GetOHLC(ctx, market, step, limit)
	if err != nil {
		return models.GetOHLCResponse{}, fmt.Errorf("failed to get OHLC data: %w", err)
	}

	entries := make([]models.OHLCEntry, 0, len(ohlcResp.Data.OHLC))
	for _, entry := range ohlcResp.Data.OHLC {
		open, _ := parseDecimalString(entry.Open, 8)
		high, _ := parseDecimalString(entry.High, 8)
		low, _ := parseDecimalString(entry.Low, 8)
		closePrice, _ := parseDecimalString(entry.Close, 8)
		volume, _ := parseDecimalString(entry.Volume, 8)

		entries = append(entries, models.OHLCEntry{
			Timestamp: time.Unix(entry.Timestamp, 0).UTC(),
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

// intervalToBitstampStep converts interval string to Bitstamp step in seconds
func intervalToBitstampStep(interval string) int {
	switch interval {
	case "1m":
		return 60
	case "3m":
		return 180
	case "5m":
		return 300
	case "15m":
		return 900
	case "30m":
		return 1800
	case "1h":
		return 3600
	case "2h":
		return 7200
	case "4h":
		return 14400
	case "6h":
		return 21600
	case "12h":
		return 43200
	case "1d":
		return 86400
	case "3d":
		return 259200
	default:
		return 3600 // Default to 1 hour
	}
}
