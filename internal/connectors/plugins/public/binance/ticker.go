package binance

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchTicker(ctx context.Context, req models.GetTickerRequest) (models.GetTickerResponse, error) {
	// Convert pair format from "BTC/USD" to Binance format "BTCUSD"
	symbol := convertPairForBinance(req.Pair)

	tickerResp, err := p.client.GetTicker24hr(ctx, symbol)
	if err != nil {
		return models.GetTickerResponse{}, fmt.Errorf("failed to get ticker: %w", err)
	}

	// Parse all the ticker values
	lastPrice, _ := parseDecimalString(tickerResp.LastPrice, 8)
	bidPrice, _ := parseDecimalString(tickerResp.BidPrice, 8)
	askPrice, _ := parseDecimalString(tickerResp.AskPrice, 8)
	volume24h, _ := parseDecimalString(tickerResp.Volume, 8)
	high24h, _ := parseDecimalString(tickerResp.HighPrice, 8)
	low24h, _ := parseDecimalString(tickerResp.LowPrice, 8)
	openPrice, _ := parseDecimalString(tickerResp.OpenPrice, 8)
	priceChange, _ := parseDecimalString(tickerResp.PriceChange, 8)

	// Handle negative price change
	if tickerResp.PriceChange != "" && tickerResp.PriceChange[0] == '-' {
		priceChange = new(big.Int).Neg(priceChange)
	}

	// Parse timestamp
	timestamp := time.UnixMilli(tickerResp.CloseTime).UTC()

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
