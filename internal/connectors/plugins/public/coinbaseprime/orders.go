package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/coinbase-samples/prime-sdk-go/model"
	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/internal/connectors/plugins/public/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

type ordersState struct {
	Cursor   string    `json:"cursor"`
	LastDate time.Time `json:"last_date"`
}

func (p *Plugin) fetchNextOrders(ctx context.Context, req models.FetchNextOrdersRequest) (models.FetchNextOrdersResponse, error) {
	var state ordersState
	if req.State != nil {
		if err := json.Unmarshal(req.State, &state); err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to unmarshal state: %w", err)
		}
	}

	params := client.ListOrdersParams{
		Cursor:        state.Cursor,
		Limit:         req.PageSize,
		SortDirection: "ASC",
	}

	if !state.LastDate.IsZero() {
		params.StartDate = state.LastDate
	}

	ordersResp, err := p.client.ListOrders(ctx, params)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to list orders: %w", err)
	}

	pspOrders := make([]models.PSPOrder, 0, len(ordersResp.Orders))
	var lastDate time.Time
	for _, order := range ordersResp.Orders {
		pspOrder, err := sdkOrderToPSPOrder(order)
		if err != nil {
			p.logger.Errorf("failed to convert order %s: %v", order.Id, err)
			continue
		}
		pspOrders = append(pspOrders, pspOrder)

		orderCreatedAt, _ := time.Parse(time.RFC3339, order.Created)
		if orderCreatedAt.After(lastDate) {
			lastDate = orderCreatedAt
		}
	}

	var newCursor string
	hasMore := false
	if ordersResp.Pagination != nil {
		newCursor = ordersResp.Pagination.NextCursor
		hasMore = ordersResp.Pagination.HasNext
	}

	newState := ordersState{
		Cursor:   newCursor,
		LastDate: lastDate,
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

	// Parse product ID from source/target assets (e.g., "BTC-USD")
	productID := fmt.Sprintf("%s-%s", order.SourceAsset, order.TargetAsset)

	// Determine side based on direction
	side := "BUY"
	if order.Direction == models.ORDER_DIRECTION_SELL {
		side = "SELL"
	}

	// Determine order type
	orderType := "MARKET"
	if order.Type == models.ORDER_TYPE_LIMIT {
		orderType = "LIMIT"
	}

	// Convert quantity to string
	quantity := order.BaseQuantityOrdered.String()

	createReq := client.CreateOrderRequest{
		ProductID:     productID,
		Side:          side,
		Type:          orderType,
		BaseQuantity:  quantity,
		ClientOrderID: order.Reference,
	}

	if order.Type == models.ORDER_TYPE_LIMIT && order.LimitPrice != nil {
		createReq.LimitPrice = order.LimitPrice.String()
	}

	// Map time in force
	switch order.TimeInForce {
	case models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED:
		createReq.TimeInForce = "GTC"
	case models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME:
		createReq.TimeInForce = "GTD"
		// Set expiration time for GTD orders
		if order.ExpiresAt != nil {
			createReq.ExpiryTime = order.ExpiresAt.Format(time.RFC3339)
		}
	case models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL:
		createReq.TimeInForce = "IOC"
	case models.TIME_IN_FORCE_FILL_OR_KILL:
		createReq.TimeInForce = "FOK"
	}

	resp, err := p.client.CreateOrder(ctx, createReq)
	if err != nil {
		return models.CreateOrderResponse{}, fmt.Errorf("failed to create order: %w", err)
	}

	// Fetch the created order to get full details
	createdOrder, err := p.client.GetOrder(ctx, resp.OrderId)
	if err != nil {
		// If we can't fetch, return with just the ID
		orderID := resp.OrderId
		return models.CreateOrderResponse{
			PollingOrderID: &orderID,
		}, nil
	}

	pspOrder, err := sdkOrderToPSPOrder(createdOrder.Order)
	if err != nil {
		orderID := resp.OrderId
		return models.CreateOrderResponse{
			PollingOrderID: &orderID,
		}, nil
	}

	return models.CreateOrderResponse{
		Order: &pspOrder,
	}, nil
}

func (p *Plugin) cancelOrder(ctx context.Context, req models.CancelOrderRequest) (models.CancelOrderResponse, error) {
	_, err := p.client.CancelOrder(ctx, req.OrderID)
	if err != nil {
		return models.CancelOrderResponse{}, fmt.Errorf("failed to cancel order: %w", err)
	}

	// Fetch the cancelled order to get updated status
	cancelledOrder, err := p.client.GetOrder(ctx, req.OrderID)
	if err != nil {
		return models.CancelOrderResponse{}, fmt.Errorf("failed to get cancelled order: %w", err)
	}

	pspOrder, err := sdkOrderToPSPOrder(cancelledOrder.Order)
	if err != nil {
		return models.CancelOrderResponse{}, fmt.Errorf("failed to convert cancelled order: %w", err)
	}

	return models.CancelOrderResponse{
		Order: pspOrder,
	}, nil
}

func sdkOrderToPSPOrder(order *model.Order) (models.PSPOrder, error) {
	raw, _ := json.Marshal(order)

	// Parse product ID to get source/target assets (e.g., "BTC-USD" -> "BTC", "USD")
	parts := strings.Split(order.ProductId, "-")
	if len(parts) != 2 {
		return models.PSPOrder{}, fmt.Errorf("invalid product ID: %s", order.ProductId)
	}
	sourceAsset := parts[0]
	targetAsset := parts[1]

	// Map direction
	direction := models.ORDER_DIRECTION_BUY
	if strings.ToUpper(order.Side) == "SELL" {
		direction = models.ORDER_DIRECTION_SELL
	}

	// Map order type
	orderType := models.ORDER_TYPE_MARKET
	if strings.ToUpper(order.Type) == "LIMIT" {
		orderType = models.ORDER_TYPE_LIMIT
	}

	// Map status
	status := mapCoinbaseStatus(order.Status, order.BaseQuantity, order.FilledQuantity)

	// Map time in force
	timeInForce := models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
	switch strings.ToUpper(order.TimeInForce) {
	case "GTC", "GOOD_UNTIL_CANCELLED":
		timeInForce = models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
	case "GTD", "GOOD_UNTIL_DATE_TIME":
		timeInForce = models.TIME_IN_FORCE_GOOD_UNTIL_DATE_TIME
	case "IOC", "IMMEDIATE_OR_CANCEL":
		timeInForce = models.TIME_IN_FORCE_IMMEDIATE_OR_CANCEL
	case "FOK", "FILL_OR_KILL":
		timeInForce = models.TIME_IN_FORCE_FILL_OR_KILL
	}

	// Parse quantities
	baseQuantityOrdered, err := parseOrderQuantity(order.BaseQuantity, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse base quantity: %w", err)
	}

	baseQuantityFilled, err := parseOrderQuantity(order.FilledQuantity, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse filled quantity: %w", err)
	}

	// Parse fees
	var fee *big.Int
	if order.Commission != "" {
		fee, err = parseOrderQuantity(order.Commission, targetAsset)
		if err != nil {
			// Log but don't fail
			fee = big.NewInt(0)
		}
	}

	// Parse limit price
	var limitPrice *big.Int
	if order.LimitPrice != "" {
		limitPrice, err = parseOrderQuantity(order.LimitPrice, targetAsset)
		if err != nil {
			limitPrice = nil
		}
	}

	// Parse created time
	createdAt, _ := time.Parse(time.RFC3339, order.Created)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return models.PSPOrder{
		Reference:           order.Id,
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

func mapCoinbaseStatus(cbStatus, baseQuantity, filledQuantity string) models.OrderStatus {
	switch strings.ToUpper(cbStatus) {
	case "PENDING":
		return models.ORDER_STATUS_PENDING
	case "OPEN":
		// Check if partially filled
		if filledQuantity != "" && filledQuantity != "0" && baseQuantity != filledQuantity {
			return models.ORDER_STATUS_PARTIALLY_FILLED
		}
		return models.ORDER_STATUS_OPEN
	case "FILLED":
		return models.ORDER_STATUS_FILLED
	case "CANCELLED":
		return models.ORDER_STATUS_CANCELLED
	case "EXPIRED":
		return models.ORDER_STATUS_EXPIRED
	case "FAILED":
		return models.ORDER_STATUS_FAILED
	default:
		return models.ORDER_STATUS_PENDING
	}
}

func parseOrderQuantity(quantityStr string, asset string) (*big.Int, error) {
	if quantityStr == "" {
		return big.NewInt(0), nil
	}

	precision := GetPrecision(asset)
	amount, err := currency.GetAmountWithPrecisionFromString(quantityStr, precision)
	if err != nil {
		return nil, err
	}

	return amount, nil
}
