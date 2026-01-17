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

// wsClientState holds the WebSocket client state for the plugin
type wsClientState struct {
	client     *client.WebSocketClient
	isRunning  bool
	handler    models.OrderUpdateHandler
	stopCancel context.CancelFunc
}

// startOrderWebSocket starts a WebSocket connection for real-time order updates
func (p *Plugin) startOrderWebSocket(ctx context.Context, req models.StartOrderWebSocketRequest) (models.StartOrderWebSocketResponse, error) {
	p.wsMu.Lock()
	defer p.wsMu.Unlock()

	if p.wsClient != nil && p.wsClient.isRunning {
		return models.StartOrderWebSocketResponse{}, fmt.Errorf("WebSocket connection already active")
	}

	// Create WebSocket client
	wsClient := client.NewWebSocketClient(
		p.httpClient,
		p.baseURL,
		p.config.APIKey,
		p.config.SecretKey,
		p.config.TestNet,
	)

	// Configure reconnection settings from request config
	wsClient.SetAutoReconnect(req.Config.AutoReconnect)

	// Create cancellation context for the WebSocket connection
	wsCtx, wsCancel := context.WithCancel(ctx)

	// Store the WebSocket client state
	p.wsClient = &wsClientState{
		client:     wsClient,
		isRunning:  true,
		handler:    req.Handler,
		stopCancel: wsCancel,
	}

	// Create callback that converts Binance events to PSPOrder
	callback := func(event client.OrderUpdateEvent) {
		pspOrder := binanceWSEventToPSPOrder(event)
		if req.Handler != nil {
			req.Handler(pspOrder)
		}
	}

	// Start the WebSocket connection
	if err := wsClient.Start(wsCtx, callback); err != nil {
		p.wsClient = nil
		wsCancel()
		return models.StartOrderWebSocketResponse{}, fmt.Errorf("failed to start WebSocket: %w", err)
	}

	p.logger.Infof("WebSocket connection established for order updates")

	// Return stop function
	stopFunc := func() {
		p.stopOrderWebSocket(context.Background())
	}

	return models.StartOrderWebSocketResponse{
		StopFunc: stopFunc,
	}, nil
}

// stopOrderWebSocket stops the WebSocket connection
func (p *Plugin) stopOrderWebSocket(ctx context.Context) {
	p.wsMu.Lock()
	defer p.wsMu.Unlock()

	if p.wsClient == nil || !p.wsClient.isRunning {
		return
	}

	// Cancel the context
	if p.wsClient.stopCancel != nil {
		p.wsClient.stopCancel()
	}

	// Stop the WebSocket client
	p.wsClient.client.Stop(ctx)
	p.wsClient.isRunning = false
	p.wsClient = nil

	p.logger.Infof("WebSocket connection closed")
}

// binanceWSEventToPSPOrder converts a Binance WebSocket order update event to a PSPOrder
func binanceWSEventToPSPOrder(event client.OrderUpdateEvent) models.PSPOrder {
	raw, _ := json.Marshal(event)

	// Parse symbol to get source/target assets
	sourceAsset, targetAsset := parseBinanceSymbol(event.Symbol)

	// Map direction
	direction := models.ORDER_DIRECTION_BUY
	if strings.ToUpper(event.Side) == "SELL" {
		direction = models.ORDER_DIRECTION_SELL
	}

	// Map order type
	orderType := models.ORDER_TYPE_MARKET
	switch strings.ToUpper(event.OrderType) {
	case "LIMIT", "LIMIT_MAKER":
		orderType = models.ORDER_TYPE_LIMIT
	case "STOP_LOSS_LIMIT", "TAKE_PROFIT_LIMIT":
		orderType = models.ORDER_TYPE_STOP_LIMIT
	}

	// Map status
	status := mapBinanceWSStatus(event.OrderStatus, event.ExecutionType)

	// Map time in force
	timeInForce := mapBinanceTimeInForce(event.TimeInForce)

	// Parse quantities
	baseQuantityOrdered := parseDecimalToBigInt(event.Quantity, sourceAsset)
	baseQuantityFilled := parseDecimalToBigInt(event.CumulativeFilledQty, sourceAsset)

	// Parse limit price
	var limitPrice *big.Int
	if event.Price != "" && event.Price != "0.00000000" {
		limitPrice = parseDecimalToBigInt(event.Price, targetAsset)
	}

	// Parse stop price
	var stopPrice *big.Int
	if event.StopPrice != "" && event.StopPrice != "0.00000000" {
		stopPrice = parseDecimalToBigInt(event.StopPrice, targetAsset)
	}

	// Parse fee
	var fee *big.Int
	var feeAsset *string
	if event.CommissionAmount != "" && event.CommissionAmount != "0" {
		fee = parseDecimalToBigInt(event.CommissionAmount, event.CommissionAsset)
		feeAsset = &event.CommissionAsset
	}

	// Parse created time
	createdAt := time.UnixMilli(event.OrderCreationTime)

	// Use client order ID if available, otherwise use order ID
	reference := event.ClientOrderID
	if reference == "" {
		reference = strconv.FormatInt(event.OrderID, 10)
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
		Fee:                 fee,
		FeeAsset:            feeAsset,
		Status:              status,
		TimeInForce:         timeInForce,
		Raw:                 raw,
	}
}

// mapBinanceWSStatus maps Binance WebSocket order status to Formance status
func mapBinanceWSStatus(orderStatus, executionType string) models.OrderStatus {
	switch strings.ToUpper(orderStatus) {
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

// parseDecimalToBigInt converts a decimal string to big.Int with the appropriate precision
func parseDecimalToBigInt(decimalStr string, asset string) *big.Int {
	if decimalStr == "" {
		return big.NewInt(0)
	}

	precision := GetPrecision(asset)
	if precision == 0 {
		precision = 8 // Default precision
	}

	// Split by decimal point
	parts := strings.Split(decimalStr, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}

	// Pad or truncate fractional part
	if len(fracPart) > precision {
		fracPart = fracPart[:precision]
	} else {
		for len(fracPart) < precision {
			fracPart += "0"
		}
	}

	// Combine and parse
	combined := intPart + fracPart
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		combined = "0"
	}

	result := new(big.Int)
	result.SetString(combined, 10)
	return result
}
