package binance

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchQuote(ctx context.Context, req models.GetQuoteRequest) (models.GetQuoteResponse, error) {
	// Convert pair format from "BTC/USD" to Binance format "BTCUSD"
	symbol := convertPairForBinance(req.SourceAsset + req.TargetAsset)

	// Binance doesn't have a dedicated quote endpoint, so we use the order book
	// to calculate an estimated price
	orderBookResp, err := p.client.GetOrderBook(ctx, symbol, 5)
	if err != nil {
		return models.GetQuoteResponse{}, fmt.Errorf("failed to get order book: %w", err)
	}

	var price *big.Int
	var fee *big.Int

	// For BUY orders, we look at asks (sellers)
	// For SELL orders, we look at bids (buyers)
	if req.Direction == "BUY" {
		if len(orderBookResp.Asks) > 0 {
			price, err = parseDecimalString(orderBookResp.Asks[0][0], 8)
			if err != nil {
				return models.GetQuoteResponse{}, fmt.Errorf("failed to parse ask price: %w", err)
			}
		}
	} else {
		if len(orderBookResp.Bids) > 0 {
			price, err = parseDecimalString(orderBookResp.Bids[0][0], 8)
			if err != nil {
				return models.GetQuoteResponse{}, fmt.Errorf("failed to parse bid price: %w", err)
			}
		}
	}

	if price == nil {
		return models.GetQuoteResponse{}, fmt.Errorf("no price available for %s/%s", req.SourceAsset, req.TargetAsset)
	}

	// Estimate fee (Binance has tiered fees, use a conservative 0.1% estimate for spot)
	// Fee = (price * quantity) * 0.001
	if req.Quantity != nil {
		totalCost := new(big.Int).Mul(price, req.Quantity)
		fee = new(big.Int).Div(totalCost, big.NewInt(1000)) // 0.1% = 1/1000
	} else {
		fee = big.NewInt(0)
	}

	// Quote expires in 30 seconds (indicative only since we're using order book)
	expiresAt := time.Now().Add(30 * time.Second)

	return models.GetQuoteResponse{
		Quote: models.Quote{
			SourceAsset: req.SourceAsset,
			TargetAsset: req.TargetAsset,
			Price:       price,
			Fee:         fee,
			ExpiresAt:   expiresAt,
		},
	}, nil
}
