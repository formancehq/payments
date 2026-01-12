package coinbaseprime

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchQuote(ctx context.Context, req models.GetQuoteRequest) (models.GetQuoteResponse, error) {
	// Convert pair format to Coinbase format (e.g., "BTC-USD")
	productID := req.SourceAsset + "-" + req.TargetAsset

	// Get the order book to derive the quote
	orderBookResp, err := p.client.GetOrderBook(ctx, productID, 1)
	if err != nil {
		return models.GetQuoteResponse{}, fmt.Errorf("failed to get order book for quote: %w", err)
	}

	var price *big.Int
	now := time.Now().UTC()

	// For BUY orders, we use the best ask price
	// For SELL orders, we use the best bid price
	if req.Direction == "BUY" {
		if len(orderBookResp.Asks) == 0 {
			return models.GetQuoteResponse{}, fmt.Errorf("no asks available for quote")
		}
		price, err = parseDecimalString(orderBookResp.Asks[0].Price, 8)
		if err != nil {
			return models.GetQuoteResponse{}, fmt.Errorf("failed to parse ask price: %w", err)
		}
	} else {
		if len(orderBookResp.Bids) == 0 {
			return models.GetQuoteResponse{}, fmt.Errorf("no bids available for quote")
		}
		price, err = parseDecimalString(orderBookResp.Bids[0].Price, 8)
		if err != nil {
			return models.GetQuoteResponse{}, fmt.Errorf("failed to parse bid price: %w", err)
		}
	}

	// Calculate total price (price * quantity)
	totalPrice := new(big.Int).Mul(price, req.Quantity)
	totalPrice.Div(totalPrice, big.NewInt(100000000)) // Adjust for precision

	quote := models.Quote{
		SourceAsset: req.SourceAsset,
		TargetAsset: req.TargetAsset,
		Direction:   req.Direction,
		Quantity:    req.Quantity,
		Price:       price,
		TotalPrice:  totalPrice,
		ExpiresAt:   now.Add(30 * time.Second), // Quote expires in 30 seconds
		Timestamp:   now,
	}

	return models.GetQuoteResponse{
		Quote: quote,
	}, nil
}
