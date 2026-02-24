package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

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

	ordersResp, err := p.client.ListOrders(ctx, state.Cursor, req.PageSize)
	if err != nil {
		return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to list orders: %w", err)
	}

	pspOrders := make([]models.PSPOrder, 0, len(ordersResp.Orders))
	var lastDate time.Time
	for _, order := range ordersResp.Orders {
		pspOrder, err := p.clientOrderToPSPOrder(order)
		if err != nil {
			p.logger.Errorf("failed to convert order %s: %v", order.ID, err)
			continue
		}
		pspOrders = append(pspOrders, pspOrder)

		orderCreatedAt, _ := time.Parse(time.RFC3339, order.CreatedAt)
		if orderCreatedAt.After(lastDate) {
			lastDate = orderCreatedAt
		}
	}

	newCursor := ordersResp.Pagination.NextCursor
	hasMore := ordersResp.Pagination.HasNext

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

func (p *Plugin) clientOrderToPSPOrder(order client.Order) (models.PSPOrder, error) {
	raw, _ := json.Marshal(order)

	// Parse product ID to get source/target assets (e.g., "BTC-USD" -> "BTC", "USD")
	parts := strings.Split(order.ProductID, "-")
	if len(parts) != 2 {
		return models.PSPOrder{}, fmt.Errorf("invalid product ID: %s", order.ProductID)
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
	baseQuantityOrdered, err := p.parseOrderQuantity(order.BaseQuantity, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse base quantity: %w", err)
	}

	baseQuantityFilled, err := p.parseOrderQuantity(order.FilledQuantity, sourceAsset)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse filled quantity: %w", err)
	}

	// Parse fees
	var fee *big.Int
	if order.Commission != "" {
		fee, err = p.parseOrderQuantity(order.Commission, targetAsset)
		if err != nil {
			fee = big.NewInt(0)
		}
	}

	// Parse limit price
	var limitPrice *big.Int
	if order.LimitPrice != "" {
		limitPrice, err = p.parseOrderQuantity(order.LimitPrice, targetAsset)
		if err != nil {
			limitPrice = nil
		}
	}

	// Parse created time
	createdAt, _ := time.Parse(time.RFC3339, order.CreatedAt)
	if createdAt.IsZero() {
		createdAt = time.Now()
	}

	return models.PSPOrder{
		Reference:           order.ID,
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

func (p *Plugin) parseOrderQuantity(quantityStr string, asset string) (*big.Int, error) {
	if quantityStr == "" {
		return big.NewInt(0), nil
	}

	precision := p.getPrecision(asset)
	amount, err := currency.GetAmountWithPrecisionFromString(quantityStr, precision)
	if err != nil {
		return nil, err
	}

	return amount, nil
}

func (p *Plugin) getPrecision(asset string) int {
	if p.currencies != nil {
		if precision, ok := p.currencies[strings.ToUpper(asset)]; ok {
			return precision
		}
	}
	// Default precision for unknown assets
	return 8
}
