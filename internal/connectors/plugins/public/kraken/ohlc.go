package kraken

import (
	"context"
	"fmt"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchOHLC(ctx context.Context, req models.GetOHLCRequest) (models.GetOHLCResponse, error) {
	// Convert pair format from "BTC/USD" to Kraken format "XBTUSD"
	pair := convertPairForKraken(req.Pair)

	// Convert interval string to Kraken interval (minutes)
	interval := intervalToKrakenInterval(req.Interval)

	// Convert since time to Unix timestamp
	var since *int64
	if req.Since != nil {
		ts := req.Since.Unix()
		since = &ts
	}

	ohlcResp, err := p.client.GetOHLC(ctx, pair, interval, since)
	if err != nil {
		return models.GetOHLCResponse{}, fmt.Errorf("failed to get OHLC data: %w", err)
	}

	entries := make([]models.OHLCEntry, 0, len(ohlcResp.Entries))
	for _, entry := range ohlcResp.Entries {
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

	// Apply limit if specified
	if req.Limit > 0 && len(entries) > req.Limit {
		entries = entries[:req.Limit]
	}

	return models.GetOHLCResponse{
		Data: models.OHLCData{
			Pair:     req.Pair,
			Interval: req.Interval,
			Entries:  entries,
		},
	}, nil
}

// intervalToKrakenInterval converts interval string to Kraken interval in minutes
func intervalToKrakenInterval(interval string) int {
	switch interval {
	case "1m":
		return 1
	case "5m":
		return 5
	case "15m":
		return 15
	case "30m":
		return 30
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	case "1w":
		return 10080
	default:
		return 60 // Default to 1 hour
	}
}
