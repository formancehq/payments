package coinbaseprime

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/models"
)

func (p *Plugin) fetchOrderBook(ctx context.Context, req models.GetOrderBookRequest) (models.GetOrderBookResponse, error) {
	// Convert pair format from "BTC/USD" to "BTC-USD" for Coinbase
	productID := strings.ReplaceAll(req.Pair, "/", "-")

	orderBookResp, err := p.client.GetOrderBook(ctx, productID, req.Depth)
	if err != nil {
		return models.GetOrderBookResponse{}, fmt.Errorf("failed to get order book: %w", err)
	}

	orderBook := models.OrderBook{
		Pair:      req.Pair,
		Timestamp: time.Now().UTC(),
		Bids:      make([]models.OrderBookEntry, 0, len(orderBookResp.Bids)),
		Asks:      make([]models.OrderBookEntry, 0, len(orderBookResp.Asks)),
	}

	// Parse bids
	for _, bid := range orderBookResp.Bids {
		price, err := parseOrderBookPrice(bid.Price)
		if err != nil {
			continue
		}
		quantity, err := parseOrderBookQuantity(bid.Size)
		if err != nil {
			continue
		}
		orderBook.Bids = append(orderBook.Bids, models.OrderBookEntry{
			Price:    price,
			Quantity: quantity,
		})
	}

	// Parse asks
	for _, ask := range orderBookResp.Asks {
		price, err := parseOrderBookPrice(ask.Price)
		if err != nil {
			continue
		}
		quantity, err := parseOrderBookQuantity(ask.Size)
		if err != nil {
			continue
		}
		orderBook.Asks = append(orderBook.Asks, models.OrderBookEntry{
			Price:    price,
			Quantity: quantity,
		})
	}

	return models.GetOrderBookResponse{
		OrderBook: orderBook,
	}, nil
}

// parseOrderBookPrice parses a price string and converts it to a big.Int
// with 8 decimal places of precision
func parseOrderBookPrice(priceStr string) (*big.Int, error) {
	return parseDecimalString(priceStr, 8)
}

// parseOrderBookQuantity parses a quantity string and converts it to a big.Int
// with 8 decimal places of precision
func parseOrderBookQuantity(quantityStr string) (*big.Int, error) {
	return parseDecimalString(quantityStr, 8)
}

// parseDecimalString parses a decimal string and converts it to a big.Int
// with the specified precision
func parseDecimalString(str string, precision int) (*big.Int, error) {
	if str == "" {
		return big.NewInt(0), nil
	}

	parts := strings.Split(str, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}

	// Truncate or pad fractional part to target precision
	if len(fracPart) > precision {
		fracPart = fracPart[:precision]
	} else {
		for len(fracPart) < precision {
			fracPart += "0"
		}
	}

	// Combine into a single string without decimal point
	combined := intPart + fracPart

	// Remove leading zeros (but keep at least one digit)
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		combined = "0"
	}

	// Parse as big.Int
	amount := new(big.Int)
	_, ok := amount.SetString(combined, 10)
	if !ok {
		return nil, fmt.Errorf("failed to parse amount: %s", str)
	}

	return amount, nil
}
