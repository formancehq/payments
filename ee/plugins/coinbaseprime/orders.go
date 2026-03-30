package coinbaseprime

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/currency"
	"github.com/formancehq/payments/ee/plugins/coinbaseprime/client"
	"github.com/formancehq/payments/internal/models"
)

type ordersState struct {
	Cursor string `json:"cursor"`
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
	for _, order := range ordersResp.Orders {
		pspOrder, err := p.clientOrderToPSPOrder(order)
		if err != nil {
			return models.FetchNextOrdersResponse{}, fmt.Errorf("failed to convert order %s: %w", order.ID, err)
		}
		pspOrders = append(pspOrders, pspOrder)
	}

	newCursor := ordersResp.Pagination.NextCursor
	hasMore := ordersResp.Pagination.HasNext

	newState := ordersState{
		Cursor: newCursor,
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
	raw, err := json.Marshal(order)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to marshal order raw: %w", err)
	}

	// Parse product ID to get base/quote assets (e.g., "BTC-USD" -> base="BTC", quote="USD")
	parts := strings.Split(order.ProductID, "-")
	if len(parts) != 2 {
		return models.PSPOrder{}, fmt.Errorf("invalid product ID: %s", order.ProductID)
	}
	baseSymbol := parts[0]
	quoteSymbol := parts[1]

	// Resolve assets with proper formatting (e.g., "BTC" -> "BTC/8")
	baseAsset, _, baseOk := p.resolveAssetAndPrecision(baseSymbol)
	if !baseOk {
		return models.PSPOrder{}, fmt.Errorf("unsupported base asset: %s", baseSymbol)
	}
	quoteAsset, _, quoteOk := p.resolveAssetAndPrecision(quoteSymbol)
	if !quoteOk {
		return models.PSPOrder{}, fmt.Errorf("unsupported quote asset: %s", quoteSymbol)
	}

	// Map direction and determine source/target based on trade direction
	// BUY BTC-USD: source=USD (what you spend), target=BTC (what you receive)
	// SELL BTC-USD: source=BTC (what you spend), target=USD (what you receive)
	direction := models.ORDER_DIRECTION_BUY
	sourceAsset := quoteAsset
	targetAsset := baseAsset
	if strings.ToUpper(order.Side) == "SELL" {
		direction = models.ORDER_DIRECTION_SELL
		sourceAsset = baseAsset
		targetAsset = quoteAsset
	}

	// Map order type
	orderType := models.ORDER_TYPE_MARKET
	switch strings.ToUpper(order.Type) {
	case "LIMIT":
		orderType = models.ORDER_TYPE_LIMIT
	case "STOP_LIMIT":
		orderType = models.ORDER_TYPE_STOP_LIMIT
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

	// Parse quantities using base asset precision
	baseQuantityOrdered, err := p.parseOrderQuantity(order.BaseQuantity, baseSymbol)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse base quantity: %w", err)
	}

	baseQuantityFilled, err := p.parseOrderQuantity(order.FilledQuantity, baseSymbol)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse filled quantity: %w", err)
	}

	// Parse fees using quote asset precision
	var fee *big.Int
	if order.Commission != "" {
		fee, err = p.parseOrderQuantity(order.Commission, quoteSymbol)
		if err != nil {
			return models.PSPOrder{}, fmt.Errorf("failed to parse commission: %w", err)
		}
	}

	// Parse limit price using quote asset precision
	var limitPrice *big.Int
	if order.LimitPrice != "" {
		limitPrice, err = p.parseOrderQuantity(order.LimitPrice, quoteSymbol)
		if err != nil {
			return models.PSPOrder{}, fmt.Errorf("failed to parse limit price: %w", err)
		}
	}

	// Parse created time — return error if unparseable to ensure idempotent adjustment IDs
	createdAt, err := time.Parse(time.RFC3339, order.CreatedAt)
	if err != nil {
		return models.PSPOrder{}, fmt.Errorf("failed to parse order createdAt %q: %w", order.CreatedAt, err)
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
		// Check if partially filled using numeric comparison
		if filledQuantity != "" && filledQuantity != "0" {
			filled, ok1 := new(big.Float).SetString(filledQuantity)
			base, ok2 := new(big.Float).SetString(baseQuantity)
			if ok1 && ok2 && filled.Sign() > 0 && filled.Cmp(base) < 0 {
				return models.ORDER_STATUS_PARTIALLY_FILLED
			}
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
