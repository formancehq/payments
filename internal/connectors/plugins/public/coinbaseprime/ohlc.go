package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchOHLC(ctx context.Context, req models.GetOHLCRequest) (models.GetOHLCResponse, error) {
	// Convert pair format from "BTC/USD" to "BTC-USD" for Coinbase
	productID := strings.ReplaceAll(req.Pair, "/", "-")

	// Convert interval to Coinbase granularity (seconds)
	granularity := intervalToGranularity(req.Interval)

	candles, err := p.fetchCandlesFromExchange(ctx, productID, granularity, req.Since, req.Limit)
	if err != nil {
		return models.GetOHLCResponse{}, fmt.Errorf("failed to get OHLC data: %w", err)
	}

	return models.GetOHLCResponse{
		Data: models.OHLCData{
			Pair:     req.Pair,
			Interval: req.Interval,
			Entries:  candles,
		},
	}, nil
}

// intervalToGranularity converts interval string to Coinbase granularity in seconds
func intervalToGranularity(interval string) int {
	switch interval {
	case "1m":
		return 60
	case "5m":
		return 300
	case "15m":
		return 900
	case "30m":
		return 1800
	case "1h":
		return 3600
	case "4h":
		return 14400
	case "1d":
		return 86400
	case "1w":
		return 604800
	default:
		return 3600 // Default to 1 hour
	}
}

// fetchCandlesFromExchange fetches OHLC data from Coinbase Exchange public API
func (p *Plugin) fetchCandlesFromExchange(ctx context.Context, productID string, granularity int, since *time.Time, limit int) ([]models.OHLCEntry, error) {
	// Coinbase Exchange API: GET /products/{product_id}/candles
	url := fmt.Sprintf("https://api.exchange.coinbase.com/products/%s/candles?granularity=%d", productID, granularity)

	if since != nil {
		url += fmt.Sprintf("&start=%s", since.Format(time.RFC3339))
		// Coinbase requires end time if start is provided
		end := time.Now().UTC()
		url += fmt.Sprintf("&end=%s", end.Format(time.RFC3339))
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code %d: %s", resp.StatusCode, string(body))
	}

	// Coinbase returns candles as array of arrays: [[timestamp, low, high, open, close, volume], ...]
	var rawCandles [][]json.Number
	if err := json.NewDecoder(resp.Body).Decode(&rawCandles); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	entries := make([]models.OHLCEntry, 0, len(rawCandles))
	for _, candle := range rawCandles {
		if len(candle) < 6 {
			continue
		}

		timestamp, _ := candle[0].Int64()
		low, _ := parseDecimalString(candle[1].String(), 8)
		high, _ := parseDecimalString(candle[2].String(), 8)
		open, _ := parseDecimalString(candle[3].String(), 8)
		closePrice, _ := parseDecimalString(candle[4].String(), 8)
		volume, _ := parseDecimalString(candle[5].String(), 8)

		entries = append(entries, models.OHLCEntry{
			Timestamp: time.Unix(timestamp, 0).UTC(),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     closePrice,
			Volume:    volume,
		})
	}

	// Apply limit if specified
	if limit > 0 && len(entries) > limit {
		entries = entries[:limit]
	}

	return entries, nil
}
