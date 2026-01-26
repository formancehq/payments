package kraken

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTicker(ctx context.Context, req models.GetTickerRequest) (models.GetTickerResponse, error) {
	// Convert pair format from "BTC/USD" to Kraken format "XBTUSD"
	pair := convertPairForKraken(req.Pair)

	tickerResp, err := p.client.GetTicker(ctx, pair)
	if err != nil {
		return models.GetTickerResponse{}, fmt.Errorf("failed to get ticker: %w", err)
	}

	tickerData := tickerResp.Data

	// Parse values
	var lastPrice, bidPrice, askPrice, volume24h, high24h, low24h, openPrice *big.Int

	if len(tickerData.C) > 0 {
		lastPrice, _ = parseDecimalString(tickerData.C[0], 8)
	}
	if len(tickerData.B) > 0 {
		bidPrice, _ = parseDecimalString(tickerData.B[0], 8)
	}
	if len(tickerData.A) > 0 {
		askPrice, _ = parseDecimalString(tickerData.A[0], 8)
	}
	if len(tickerData.V) > 1 {
		volume24h, _ = parseDecimalString(tickerData.V[1], 8)
	}
	if len(tickerData.H) > 1 {
		high24h, _ = parseDecimalString(tickerData.H[1], 8)
	}
	if len(tickerData.L) > 1 {
		low24h, _ = parseDecimalString(tickerData.L[1], 8)
	}
	openPrice, _ = parseDecimalString(tickerData.O, 8)

	// Calculate price change
	var priceChange *big.Int
	if openPrice != nil && lastPrice != nil && openPrice.Cmp(big.NewInt(0)) != 0 {
		priceChange = new(big.Int).Sub(lastPrice, openPrice)
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
			OpenPrice:   openPrice,
			PriceChange: priceChange,
			Timestamp:   time.Now().UTC(),
		},
	}, nil
}
