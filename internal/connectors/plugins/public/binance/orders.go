package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/binance/client"
	"github.com/formancehq/payments/internal/models"
)

type ordersState struct {
	LastSyncTime time.Time         `json:"last_sync_time"`
	SymbolStates map[string]int64  `json:"symbol_states"` // symbol -> last order ID
}

func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	var state ordersState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}
	if state.SymbolStates == nil {
		state.SymbolStates = make(map[string]int64)
	}

	// Fetch open orders first
	openOrders, err := p.client.GetOpenOrders(ctx, "")
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to get open orders: %w", err)
	}

	// Convert orders to PSPOrders
	pspOrders := make([]models.PSPOrder, 0, len(openOrders))

	for _, order := range openOrders {
		pspOrder, err := binanceOrderToPSPOrder(order)
		if err != nil {
			p.logger.Errorf("failed to convert order %d: %v", order.OrderID, err)
			continue
		}
		pspOrders = append(pspOrders, pspOrder)
	}

	// Update state
	newState := ordersState{
		LastSyncTime: time.Now(),
		SymbolStates: state.SymbolStates,
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

	// Build symbol from source/target assets (e.g., "BTCUSDT")
	symbol := buildBinanceSymbol(order.SourceAsset, order.TargetAsset)

	// Determine side
	side := "BUY"
	if order.Direction == models.ORDER_DIRECTION_SELL {
		side = "SELL"
	}

	// Determine order type
	orderType := "MARKET"
	switch order.Type {
	case models.ORDER_TYPE_LIMIT:
		orderType = "LIMIT"
	case models.ORDER_TYPE_STOP_LIMIT:
		orderType = "STOP_LOSS_LIMIT"
	}

	// Convert quantity to string
	quantity := formatBigIntAsDecimal(order.BaseQuantityOrdered, order.SourceAsset)

	createReq := client.CreateOrderRequest{
		Symbol:           symbol,
		Side:             side,
		Type:             orderType,
		Quantity:         quantity,
		NewClientOrderID: order.Reference,
	}

	// Add limit price for limit and stop_limit orders
	if (order.Type == models.ORDER_TYPE_LIMIT || order.Type == models.ORDER_TYPE_STOP_LIMIT) && order.LimitPrice != nil {
		createReq.Price = formatBigIntAsDecimal(order.LimitPrice, order.TargetAsset)
	}

	// Add stop price for stop_limit orders
	if order.Type == models.ORDER_TYPE_STOP_LIMIT && order.StopPrice != nil {
		createReq.StopPrice = formatBigIntAsDecimal(order.StopPrice, order.TargetAsset)
	}

	// Map time in force
	switch order.TimeInForce {
	case models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED:
		createReq.TimeInForce = "GTC"
	case models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		createReq.TimeInForce = "IOC"
	case models.TIME_IN_FORCE_FILL_OR_KILL:
		createReq.TimeInForce = "FOK"
	default:
		// GTC is the default for limit orders
		if orderType == "LIMIT" {
			createReq.TimeInForce = "GTC"
		}
	}

	resp, err := p.client.CreateOrder(ctx, createReq)
	if err != nil {
		return models.CreateOrderResponse{}, fmt.Errorf("failed to create order: %w", err)
	}

	// Return the order ID for polling
	orderID := strconv.FormatInt(resp.OrderID, 10)
	return models.CreateOrderResponse{
		PollingOrderID: &orderID,
	}, nil
}

func (p *Plugin) cancelOrder(ctx context.Context, req models.CancelOrderRequest) (models.CancelOrderResponse, error) {
	// Parse order ID and symbol from the request
	// The order ID format should be "symbol:orderID" or just the order ID with symbol provided
	parts := strings.Split(req.OrderID, ":")
	var symbol string
	var orderID int64
	var err error

	if len(parts) == 2 {
		symbol = parts[0]
		orderID, err = strconv.ParseInt(parts[1], 10, 64)
	} else {
		// Try parsing as just order ID - will need symbol from elsewhere
		orderID, err = strconv.ParseInt(req.OrderID, 10, 64)
		// Default to empty symbol - this will likely fail but it's the best we can do
		symbol = ""
	}

	if err != nil {
		return models.CancelOrderResponse{}, fmt.Errorf("invalid order ID format: %w", err)
	}

	if symbol == "" {
		return models.CancelOrderResponse{}, fmt.Errorf("symbol is required for cancellation (use format symbol:orderID)")
	}

	_, err = p.client.CancelOrder(ctx, symbol, orderID)
	if err != nil {
		return models.CancelOrderResponse{}, fmt.Errorf("failed to cancel order: %w", err)
	}

	return models.CancelOrderResponse{
		Order: models.PSPOrder{
			Reference: req.OrderID,
			Status:    models.ORDER_STATUS_CANCELLED,
		},
	}, nil
}

func binanceOrderToPSPOrder(order client.Order) (models.PSPOrder, error) {
	raw, _ := json.Marshal(order)

	// Parse symbol to get source/target assets
	sourceAsset, targetAsset := parseBinanceSymbol(order.Symbol)

	// Map direction
	direction := models.ORDER_DIRECTION_BUY
	if strings.ToUpper(order.Side) == "SELL" {
		direction = models.ORDER_DIRECTION_SELL
	}

	// Map order type
	orderType := models.ORDER_TYPE_MARKET
	switch strings.ToUpper(order.Type) {
	case "LIMIT", "LIMIT_MAKER":
		orderType = models.ORDER_TYPE_LIMIT
	case "STOP_LOSS_LIMIT", "TAKE_PROFIT_LIMIT":
		orderType = models.ORDER_TYPE_STOP_LIMIT
	}

	// Map status
	status := mapBinanceStatus(order.Status, order.OrigQty, order.ExecutedQty)

	// Map time in force
	timeInForce := mapBinanceTimeInForce(order.TimeInForce)

	// Parse quantities
	baseQuantityOrdered, err := parseBinanceAmount(order.OrigQty, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse quantity: %w", err)
	}

	baseQuantityFilled, err := parseBinanceAmount(order.ExecutedQty, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse executed quantity: %w", err)
	}

	// Parse limit price
	var limitPrice *big.Int
	if order.Price != "" && order.Price != "0.00000000" {
		limitPrice, err = parseBinanceAmount(order.Price, targetAsset)
		if err != nil {
			limitPrice = nil
		}
	}

	// Parse stop price for STOP_LIMIT orders
	var stopPrice *big.Int
	if order.StopPrice != "" && order.StopPrice != "0.00000000" {
		stopPrice, err = parseBinanceAmount(order.StopPrice, targetAsset)
		if err != nil {
			stopPrice = nil
		}
	}

	// Parse created time
	createdAt := time.UnixMilli(order.Time)

	// Use client order ID if available, otherwise use order ID
	reference := order.ClientOrderID
	if reference == "" {
		reference = strconv.FormatInt(order.OrderID, 10)
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
		StopPrice:           stopPrice,
		Fee:                 nil, // Fee is not in order response
		Status:              status,
		TimeInForce:         timeInForce,
		Raw:                 raw,
	}, nil
}

func mapBinanceStatus(status, origQty, executedQty string) models.OrderStatus {
	switch strings.ToUpper(status) {
	case "NEW":
		return models.ORDER_STATUS_OPEN
	case "PARTIALLY_FILLED":
		return models.ORDER_STATUS_PARTIALLY_FILLED
	case "FILLED":
		return models.ORDER_STATUS_FILLED
	case "CANCELED":
		return models.ORDER_STATUS_CANCELLED
	case "PENDING_CANCEL":
		return models.ORDER_STATUS_OPEN
	case "REJECTED":
		return models.ORDER_STATUS_FAILED
	case "EXPIRED":
		return models.ORDER_STATUS_EXPIRED
	default:
		return models.ORDER_STATUS_PENDING
	}
}

func mapBinanceTimeInForce(tif string) models.TimeInForce {
	switch strings.ToUpper(tif) {
	case "GTC":
		return models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
	case "IOC":
		return models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL
	case "FOK":
		return models.TIME_IN_FORCE_FILL_OR_KILL
	default:
		return models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
	}
}

// buildBinanceSymbol builds a Binance symbol from source and target assets
func buildBinanceSymbol(sourceAsset, targetAsset string) string {
	// Binance uses concatenated symbols (e.g., "BTCUSDT")
	return strings.ToUpper(sourceAsset + targetAsset)
}

// parseBinanceSymbol extracts source and target assets from a Binance symbol
func parseBinanceSymbol(symbol string) (string, string) {
	// Common quote assets in Binance
	quoteAssets := []string{"USDT", "BUSD", "USDC", "TUSD", "FDUSD", "USD", "EUR", "GBP", "BTC", "ETH", "BNB"}

	symbol = strings.ToUpper(symbol)

	for _, quote := range quoteAssets {
		if strings.HasSuffix(symbol, quote) {
			base := strings.TrimSuffix(symbol, quote)
			return base, quote
		}
	}

	// If we can't parse, return as-is with best guess
	if len(symbol) > 3 {
		return symbol[:len(symbol)-4], symbol[len(symbol)-4:]
	}

	return symbol, ""
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
