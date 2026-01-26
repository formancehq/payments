package kraken

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/payments/internal/connectors/plugins/public/kraken/client"
	"github.com/formancehq/payments/internal/models"
)

type ordersState struct {
	Offset   int       `json:"offset"`
	LastSync time.Time `json:"last_sync"`
}

func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	var state ordersState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	// First fetch open orders
	openOrdersResp, err := p.client.GetOpenOrders(ctx, client.ListOrdersParams{})
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to get open orders: %w", err)
	}

	// Then fetch closed orders
	closedParams := client.ListOrdersParams{
		Offset: state.Offset,
	}
	if !state.LastSync.IsZero() {
		closedParams.Start = state.LastSync
	}

	closedOrdersResp, err := p.client.GetClosedOrders(ctx, closedParams)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to get closed orders: %w", err)
	}

	// Convert orders to PSPOrders
	pspOrders := make([]models.PSPOrder, 0)

	// Process open orders
	for orderID, order := range openOrdersResp.Orders {
		pspOrder, err := krakenOrderToPSPOrder(orderID, order)
		if err != nil {
			p.logger.Errorf("failed to convert open order %s: %v", orderID, err)
			continue
		}
		pspOrders = append(pspOrders, pspOrder)
	}

	// Process closed orders
	for orderID, order := range closedOrdersResp.Orders {
		pspOrder, err := krakenOrderToPSPOrder(orderID, order)
		if err != nil {
			p.logger.Errorf("failed to convert closed order %s: %v", orderID, err)
			continue
		}
		pspOrders = append(pspOrders, pspOrder)
	}

	// Determine if there are more pages
	hasMore := closedOrdersResp.Count > state.Offset+len(closedOrdersResp.Orders)

	// Update state
	newState := ordersState{
		Offset:   state.Offset + len(closedOrdersResp.Orders),
		LastSync: time.Now(),
	}
	if !hasMore {
		// Reset offset when we've fetched all orders
		newState.Offset = 0
	}

	stateBytes, err := json.Marshal(newState)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to marshal state: %w", err)
	}

	return models.FetchNextOrdersResponse{
		Orders:   pspOrders,
		NewState: stateBytes,
		HasMore:  hasMore,
	}, nil
}

func (p *Plugin) createOrder(ctx context.Context, req models.CreateOrderRequest) (models.CreateOrderResponse, error) {
	order := req.Order

	// Build pair from source/target assets
	pair := buildKrakenPair(order.SourceAsset, order.TargetAsset)

	// Determine order type
	orderType := "market"
	if order.Type == models.ORDER_TYPE_LIMIT {
		orderType = "limit"
	}

	// Determine side (buy or sell)
	side := "buy"
	if order.Direction == models.ORDER_DIRECTION_SELL {
		side = "sell"
	}

	// Convert quantity to string
	volume := order.BaseQuantityOrdered.String()

	createReq := client.CreateOrderRequest{
		OrderType:     orderType,
		Type:          side,
		Volume:        volume,
		Pair:          pair,
		ClientOrderID: order.Reference,
	}

	// Add limit price for limit orders
	if order.Type == models.ORDER_TYPE_LIMIT && order.LimitPrice != nil {
		createReq.Price = order.LimitPrice.String()
	}

	// Map time in force
	switch order.TimeInForce {
	case models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED:
		createReq.TimeInForce = "GTC"
	case models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		createReq.TimeInForce = "IOC"
	case models.TIME_IN_FORCE_FILL_OR_KILL:
		createReq.TimeInForce = "FOK"
	case models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME:
		createReq.TimeInForce = "GTD"
		// Set expiration time for GTD orders
		if order.ExpiresAt != nil {
			createReq.ExpireTm = fmt.Sprintf("+%d", int64(time.Until(*order.ExpiresAt).Seconds()))
		}
	}

	resp, err := p.client.CreateOrder(ctx, createReq)
	if err != nil {
		return models.CreateOrderResponse{}, fmt.Errorf("failed to create order: %w", err)
	}

	// Return the order ID for polling
	if len(resp.TxID) > 0 {
		orderID := resp.TxID[0]
		return models.CreateOrderResponse{
			PollingOrderID: &orderID,
		}, nil
	}

	return models.CreateOrderResponse{}, nil
}

func (p *Plugin) cancelOrder(ctx context.Context, req models.CancelOrderRequest) (models.CancelOrderResponse, error) {
	_, err := p.client.CancelOrder(ctx, req.OrderID)
	if err != nil {
		return models.CancelOrderResponse{}, fmt.Errorf("failed to cancel order: %w", err)
	}

	// Return an order with CANCELLED status
	// Note: Kraken doesn't return the full order details on cancel
	return models.CancelOrderResponse{
		Order: models.PSPOrder{
			Reference: req.OrderID,
			Status:    models.ORDER_STATUS_CANCELLED,
		},
	}, nil
}

func krakenOrderToPSPOrder(orderID string, order client.Order) (models.PSPOrder, error) {
	raw, _ := json.Marshal(order)

	// Parse pair to get source/target assets
	sourceAsset, targetAsset := parseKrakenPair(order.Descr.Pair)

	// Map direction
	direction := models.ORDER_DIRECTION_BUY
	if strings.ToLower(order.Descr.Type) == "sell" {
		direction = models.ORDER_DIRECTION_SELL
	}

	// Map order type
	orderType := models.ORDER_TYPE_MARKET
	if strings.ToLower(order.Descr.OrderType) == "limit" {
		orderType = models.ORDER_TYPE_LIMIT
	}

	// Map status
	status := mapKrakenStatus(order.Status, order.Vol, order.VolExec)

	// Map time in force (Kraken doesn't expose this in order response, default to GTC)
	timeInForce := models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED

	// Parse quantities
	baseQuantityOrdered, err := parseKrakenAmount(order.Vol, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse volume: %w", err)
	}

	baseQuantityFilled, err := parseKrakenAmount(order.VolExec, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse executed volume: %w", err)
	}

	// Parse fee
	var fee *big.Int
	if order.Fee != "" {
		fee, err = parseKrakenAmount(order.Fee, targetAsset)
		if err != nil {
			fee = big.NewInt(0)
		}
	}

	// Parse limit price
	var limitPrice *big.Int
	if order.Descr.Price != "" && order.Descr.Price != "0" {
		limitPrice, err = parseKrakenAmount(order.Descr.Price, targetAsset)
		if err != nil {
			limitPrice = nil
		}
	}

	// Parse created time (Kraken uses UNIX timestamp)
	createdAt := time.Unix(int64(order.OpenTime), 0)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	// Use client order ID if available, otherwise use transaction ID
	reference := orderID
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
		Fee:                 fee,
		Status:              status,
		TimeInForce:         timeInForce,
		Raw:                 raw,
	}, nil
}

func mapKrakenStatus(krakenStatus, vol, volExec string) models.OrderStatus {
	switch strings.ToLower(krakenStatus) {
	case "pending":
		return models.ORDER_STATUS_PENDING
	case "open":
		// Check if partially filled
		if volExec != "" && volExec != "0" && volExec != "0.00000000" && vol != volExec {
			return models.ORDER_STATUS_PARTIALLY_FILLED
		}
		return models.ORDER_STATUS_OPEN
	case "closed":
		return models.ORDER_STATUS_FILLED
	case "canceled", "cancelled":
		return models.ORDER_STATUS_CANCELLED
	case "expired":
		return models.ORDER_STATUS_EXPIRED
	default:
		return models.ORDER_STATUS_PENDING
	}
}

// buildKrakenPair builds a Kraken trading pair from source and target assets
func buildKrakenPair(sourceAsset, targetAsset string) string {
	// Kraken uses different formats for pairs
	// For most pairs: XXBTZUSD (BTC/USD)
	// For newer pairs: BTCUSD
	return sourceAsset + targetAsset
}

// parseKrakenPair extracts source and target assets from a Kraken pair
func parseKrakenPair(pair string) (string, string) {
	// Kraken pairs can be in various formats:
	// XXBTZUSD -> XBT/USD
	// XETHZEUR -> ETH/EUR
	// BTCUSD -> BTC/USD

	// Common quote currencies
	quoteCurrencies := []string{"USD", "EUR", "GBP", "CAD", "JPY", "AUD", "CHF", "USDT", "USDC"}

	// Normalize the pair first
	pair = normalizeKrakenAsset(pair)

	// Try to split by common quote currencies
	for _, quote := range quoteCurrencies {
		if strings.HasSuffix(pair, quote) {
			base := strings.TrimSuffix(pair, quote)
			return normalizeKrakenAsset(base), quote
		}
		// Also check with Z prefix for fiat
		if strings.HasSuffix(pair, "Z"+quote) {
			base := strings.TrimSuffix(pair, "Z"+quote)
			return normalizeKrakenAsset(base), quote
		}
	}

	// If we can't parse, return as-is with best guess
	// Assume last 3-4 characters are the quote currency
	if len(pair) > 3 {
		return pair[:len(pair)-3], pair[len(pair)-3:]
	}

	return pair, ""
}
