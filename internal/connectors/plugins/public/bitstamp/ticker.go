package bitstamp

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTicker(ctx context.Context, req models.GetTickerRequest) (models.GetTickerResponse, error) {
	// Convert pair format from "BTC/USD" to Bitstamp format "btcusd"
	market := convertPairForBitstamp(req.Pair)

	tickerResp, err := p.client.GetTicker(ctx, market)
	if err != nil {
		return models.GetTickerResponse{}, fmt.Errorf("failed to get ticker: %w", err)
	}

	// Parse all the ticker values
	lastPrice, _ := parseDecimalString(tickerResp.Last, 8)
	bidPrice, _ := parseDecimalString(tickerResp.Bid, 8)
	askPrice, _ := parseDecimalString(tickerResp.Ask, 8)
	volume24h, _ := parseDecimalString(tickerResp.Volume, 8)
	high24h, _ := parseDecimalString(tickerResp.High, 8)
	low24h, _ := parseDecimalString(tickerResp.Low, 8)
	openPrice, _ := parseDecimalString(tickerResp.Open, 8)

	// Calculate price change
	var priceChange *big.Int
	if lastPrice != nil && openPrice != nil {
		priceChange = new(big.Int).Sub(lastPrice, openPrice)
	}

	// Parse timestamp
	var timestamp time.Time
	if ts, err := strconv.ParseInt(tickerResp.Timestamp, 10, 64); err == nil {
		timestamp = time.Unix(ts, 0).UTC()
	} else {
		timestamp = time.Now().UTC()
	}

	return models.GetTickerResponse{
		Ticker: models.Ticker{
			Pair:        req.Pair,
			LastPrice:   lastPrice,
			BidPrice:    bidPrice,
			AskPrice:    askPrice,
			Volume24h:   volume24h,
			High24h:     high24h,
			Low24h:      low24h,
			PriceChange: priceChange,
			OpenPrice:   openPrice,
			Timestamp:   timestamp,
		},
	}, nil
}
