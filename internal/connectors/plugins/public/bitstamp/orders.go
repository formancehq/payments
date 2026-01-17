package bitstamp

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/bitstamp/client"
	"github.com/formancehq/payments/internal/models"
)

type ordersState struct {
	LastSync time.Time `json:"last_sync"`
}

func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	var state ordersState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	// Fetch open orders from Bitstamp
	openOrders, err := p.client.GetOpenOrders(ctx, "")
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to get open orders: %w", err)
	}

	// Convert orders to PSPOrders
	pspOrders := make([]models.PSPOrder, 0, len(openOrders))

	for _, order := range openOrders {
		pspOrder, err := bitstampOrderToPSPOrder(order)
		if err != nil {
			p.logger.Errorf("failed to convert order %s: %v", order.ID, err)
			continue
		}
		pspOrders = append(pspOrders, pspOrder)
	}

	// Update state
	newState := ordersState{
		LastSync: time.Now(),
	}

	stateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to marshal state: %w", err)
	}

	return models.FetchNextOrdersResponse{
		Orders:   pspOrders,
		NewState: stateBytes,
		HasMore:  false,
	}, nil
}

func (p *Plugin) createOrder(ctx context.Context, req models.CreateOrderRequest) (models.CreateOrderResponse, error) {
	order := req.Order

	// Bitstamp does not support STOP_LIMIT orders
	if order.Type == models.ORDER_TYPE_STOP_LIMIT {
		return models.CreateOrderResponse{}, fmt.Errorf("STOP_LIMIT orders are not supported by Bitstamp")
	}

	// Build market pair from source/target assets (e.g., "btcusd")
	market := buildBitstampMarket(order.SourceAsset, order.TargetAsset)

	// Convert quantity to string
	volume := formatBigIntAsDecimal(order.BaseQuantityOrdered, order.SourceAsset)

	createReq := client.CreateOrderRequest{
		Market:        market,
		Amount:        volume,
		ClientOrderID: order.Reference,
	}

	// Add limit price for limit orders
	if order.Type == models.ORDER_TYPE_LIMIT && order.LimitPrice != nil {
		createReq.Price = formatBigIntAsDecimal(order.LimitPrice, order.TargetAsset)
	}

	// Map time in force
	switch order.TimeInForce {
	case models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		createReq.IOCOrder = true
	case models.TIME_IN_FORCE_FILL_OR_KILL:
		createReq.FOKOrder = true
	case models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME:
		createReq.GtdOrder = true
		if order.ExpiresAt != nil {
			createReq.ExpireTime = order.ExpiresAt.Unix()
		}
	// GTC is the default, no special flag needed
	}

	var resp *client.CreateOrderResponse
	var err error

	// Call the appropriate endpoint based on order type and direction
	if order.Type == models.ORDER_TYPE_MARKET {
		if order.Direction == models.ORDER_DIRECTION_BUY {
			resp, err = p.client.CreateMarketBuyOrder(ctx, createReq)
		} else {
			resp, err = p.client.CreateMarketSellOrder(ctx, createReq)
		}
	} else {
		// Limit order
		if order.Direction == models.ORDER_DIRECTION_BUY {
			resp, err = p.client.CreateLimitBuyOrder(ctx, createReq)
		} else {
			resp, err = p.client.CreateLimitSellOrder(ctx, createReq)
		}
	}

	if err != nil {
		return models.CreateOrderResponse{}, fmt.Errorf("failed to create order: %w", err)
	}

	// Return the order ID for polling
	orderID := resp.ID
	return models.CreateOrderResponse{
		PollingOrderID: &orderID,
	}, nil
}

func (p *Plugin) cancelOrder(ctx context.Context, req models.CancelOrderRequest) (models.CancelOrderResponse, error) {
	_, err := p.client.CancelOrder(ctx, req.OrderID)
	if err != nil {
		return models.CancelOrderResponse{}, fmt.Errorf("failed to cancel order: %w", err)
	}

	// Return an order with CANCELLED status
	return models.CancelOrderResponse{
		Order: models.PSPOrder{
			Reference: req.OrderID,
			Status:    models.ORDER_STATUS_CANCELLED,
		},
	}, nil
}

func bitstampOrderToPSPOrder(order client.Order) (models.PSPOrder, error) {
	raw, _ := json.Marshal(order)

	// Parse pair to get source/target assets
	sourceAsset, targetAsset := parseBitstampPair(order.CurrencyPair)

	// Map direction (0 = buy, 1 = sell)
	direction := models.ORDER_DIRECTION_BUY
	if order.Type == "1" {
		direction = models.ORDER_DIRECTION_SELL
	}

	// Bitstamp open orders are always limit orders
	orderType := models.ORDER_TYPE_LIMIT

	// Map status
	status := mapBitstampStatus(order.Status, order.Amount, order.AmountAtCreate)

	// Map time in force (Bitstamp doesn't expose this in order response, default to GTC)
	timeInForce := models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED

	// Parse quantities
	baseQuantityOrdered, err := parseBitstampAmount(order.AmountAtCreate, sourceAsset)
	if err != nil {
		// Fall back to current amount if amount_at_create not available
		baseQuantityOrdered, err = parseBitstampAmount(order.Amount, sourceAsset)
		if err != nil {
			return models.PSPOrder{}, fmt.Errorf("failed to parse amount: %w", err)
		}
	}

	// Calculate filled amount
	currentAmount, _ := parseBitstampAmount(order.Amount, sourceAsset)
	baseQuantityFilled := new(big.Int).Sub(baseQuantityOrdered, currentAmount)
	if baseQuantityFilled.Sign() < 0 {
		baseQuantityFilled = big.NewInt(0)
	}

	// Parse limit price
	var limitPrice *big.Int
	if order.Price != "" && order.Price != "0" {
		limitPrice, err = parseBitstampAmount(order.Price, targetAsset)
		if err != nil {
			limitPrice = nil
		}
	}

	// Parse created time
	createdAt, err := time.Parse("2006-01-02 15:04:05", order.DateTime)
	if err != nil {
		createdAt = time.Now()
	}

	// Use client order ID if available, otherwise use ID
	reference := order.ID
	if order.ClientOrderID != "" {
		reference = order.ClientOrderID
	}

	return models.PSPOrder{
		Reference:           reference,
		CreatedAt:           createdAt,
		Direction:           direction,
		Type:                orderType,
		SourceAsset:         sourceAsset,
		TargetAsset:         targetAsset,
		BaseQuantityOrdered: baseQuantityOrdered,
		BaseQuantityFilled:  baseQuantityFilled,
		LimitPrice:          limitPrice,
		Fee:                 nil, // Fee is only available in transactions
		Status:              status,
		TimeInForce:         timeInForce,
		Raw:                 raw,
	}, nil
}

func mapBitstampStatus(status, currentAmount, originalAmount string) models.OrderStatus {
	switch strings.ToLower(status) {
	case "open":
		// Check if partially filled
		if currentAmount != "" && originalAmount != "" && currentAmount != originalAmount {
			return models.ORDER_STATUS_PARTIALLY_FILLED
		}
		return models.ORDER_STATUS_OPEN
	case "finished":
		return models.ORDER_STATUS_FILLED
	case "canceled", "cancelled":
		return models.ORDER_STATUS_CANCELLED
	case "expired":
		return models.ORDER_STATUS_EXPIRED
	default:
		// Open orders returned from GetOpenOrders are implicitly open
		return models.ORDER_STATUS_OPEN
	}
}

// buildBitstampMarket builds a Bitstamp market pair from source and target assets
func buildBitstampMarket(sourceAsset, targetAsset string) string {
	// Bitstamp uses lowercase pairs without separator (e.g., "btcusd")
	return strings.ToLower(sourceAsset + targetAsset)
}

// parseBitstampPair extracts source and target assets from a Bitstamp pair
func parseBitstampPair(pair string) (string, string) {
	// Bitstamp pairs can be in formats like:
	// "BTC/USD" or "btcusd" or "BTC-USD"
	pair = strings.ToUpper(pair)

	// Try splitting by common separators
	if strings.Contains(pair, "/") {
		parts := strings.Split(pair, "/")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}
	if strings.Contains(pair, "-") {
		parts := strings.Split(pair, "-")
		if len(parts) == 2 {
			return parts[0], parts[1]
		}
	}

	// Common quote currencies
	quoteCurrencies := []string{"USD", "EUR", "GBP", "USDT", "USDC"}

	// Try to split by common quote currencies
	for _, quote := range quoteCurrencies {
		if strings.HasSuffix(pair, quote) {
			base := strings.TrimSuffix(pair, quote)
			return base, quote
		}
	}

	// If we can't parse, return as-is with best guess
	if len(pair) > 3 {
		return pair[:len(pair)-3], pair[len(pair)-3:]
	}

	return pair, ""
}

// formatBigIntAsDecimal converts a big.Int to a decimal string with the appropriate precision
func formatBigIntAsDecimal(amount *big.Int, asset string) string {
	if amount == nil {
		return "0"
	}

	precision := GetPrecision(asset)
	if precision == 0 {
		return amount.String()
	}

	// Create divisor (10^precision)
	divisor := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(precision)), nil)

	// Get integer and fractional parts
	intPart := new(big.Int).Div(amount, divisor)
	fracPart := new(big.Int).Mod(amount, divisor)

	// Format fractional part with leading zeros
	fracStr := fracPart.String()
	for len(fracStr) < precision {
		fracStr = "0" + fracStr
	}

	// Trim trailing zeros
	fracStr = strings.TrimRight(fracStr, "0")
	if fracStr == "" {
		return intPart.String()
	}

	return intPart.String() + "." + fracStr
}
