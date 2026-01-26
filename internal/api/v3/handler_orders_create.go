package v3

import (
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreateOrderRequest struct {
	Reference           string            `json:"reference" validate:"required,gte=3,lte=1000"`
	ConnectorID         string            `json:"connectorID" validate:"required,connectorID"`
	Direction           string            `json:"direction" validate:"required,oneof=BUY SELL"`
	SourceAsset         string            `json:"sourceAsset" validate:"required,asset"`
	TargetAsset         string            `json:"targetAsset" validate:"required,asset"`
	Type                string            `json:"type" validate:"required,oneof=MARKET LIMIT STOP_LIMIT"`
	BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered" validate:"required"`
	LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
	StopPrice           *big.Int          `json:"stopPrice,omitempty"`
	TimeInForce         string            `json:"timeInForce" validate:"omitempty,oneof=GOOD_UNTIL_CANCELLED GOOD_UNTIL_DATE_TIME IMMEDIATE_OR_CANCEL FILL_OR_KILL GTC GTD IOC FOK"`
	ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
	Metadata            map[string]string `json:"metadata"`
	SkipValidation      bool              `json:"skipValidation,omitempty"` // Skip trading pair and order size validation
	SendToExchange      *bool             `json:"sendToExchange,omitempty"` // Send order to exchange via workflow (default: true)
	WaitResult          bool              `json:"waitResult,omitempty"`     // Wait for exchange response
}

// OrderValidationWarning represents a non-blocking validation warning
type OrderValidationWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// CreateOrderResponse wraps the order with any validation warnings
type CreateOrderResponse struct {
	Order    models.Order             `json:"order"`
	Warnings []OrderValidationWarning `json:"warnings,omitempty"`
	TaskID   *models.TaskID           `json:"taskID,omitempty"` // Task ID if order was sent to exchange
}

func ordersCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_ordersCreate")
		defer span.End()

		var req CreateOrderRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromOrderCreateRequest(span, req)

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		// Validate LIMIT orders require limitPrice
		if req.Type == "LIMIT" && req.LimitPrice == nil {
			otel.RecordError(span, fmt.Errorf("limitPrice is required for LIMIT orders"))
			api.BadRequest(w, ErrValidation, fmt.Errorf("limitPrice is required for LIMIT orders"))
			return
		}

		// Validate STOP_LIMIT orders require both limitPrice and stopPrice
		if req.Type == "STOP_LIMIT" {
			if req.LimitPrice == nil {
				otel.RecordError(span, fmt.Errorf("limitPrice is required for STOP_LIMIT orders"))
				api.BadRequest(w, ErrValidation, fmt.Errorf("limitPrice is required for STOP_LIMIT orders"))
				return
			}
			if req.StopPrice == nil {
				otel.RecordError(span, fmt.Errorf("stopPrice is required for STOP_LIMIT orders"))
				api.BadRequest(w, ErrValidation, fmt.Errorf("stopPrice is required for STOP_LIMIT orders"))
				return
			}
		}

		// Validate GTD orders require expiresAt
		if (req.TimeInForce == "GOOD_UNTIL_DATE_TIME" || req.TimeInForce == "GTD") && req.ExpiresAt == nil {
			otel.RecordError(span, fmt.Errorf("expiresAt is required for GTD orders"))
			api.BadRequest(w, ErrValidation, fmt.Errorf("expiresAt is required for GTD (GOOD_UNTIL_DATE_TIME) orders"))
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)

		// Validate trading pair and order size (unless skipValidation is true)
		var warnings []OrderValidationWarning
		if !req.SkipValidation {
			validationWarnings, validationErr := validateOrderAgainstConnector(
				ctx, backend, connectorID, req.SourceAsset, req.TargetAsset, req.BaseQuantityOrdered,
			)
			if validationErr != nil {
				otel.RecordError(span, validationErr)
				api.BadRequest(w, ErrValidation, validationErr)
				return
			}
			warnings = validationWarnings
		}

		direction, err := models.OrderDirectionFromString(req.Direction)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		orderType, err := models.OrderTypeFromString(req.Type)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		timeInForce := models.TIME_IN_FORCE_GOOD_UNTIL_CANCELLED
		if req.TimeInForce != "" {
			timeInForce, err = models.TimeInForceFromString(req.TimeInForce)
			if err != nil {
				otel.RecordError(span, err)
				api.BadRequest(w, ErrValidation, err)
				return
			}
		}

		now := time.Now().UTC()
		orderID := models.OrderID{
			Reference:   req.Reference,
			ConnectorID: connectorID,
		}

		order := models.Order{
			ID:                  orderID,
			ConnectorID:         connectorID,
			Reference:           req.Reference,
			CreatedAt:           now,
			UpdatedAt:           now,
			Direction:           direction,
			SourceAsset:         req.SourceAsset,
			TargetAsset:         req.TargetAsset,
			Type:                orderType,
			Status:              models.ORDER_STATUS_PENDING,
			BaseQuantityOrdered: req.BaseQuantityOrdered,
			BaseQuantityFilled:  big.NewInt(0),
			LimitPrice:          req.LimitPrice,
			StopPrice:           req.StopPrice,
			TimeInForce:         timeInForce,
			ExpiresAt:           req.ExpiresAt,
			Metadata:            req.Metadata,
		}

		// Create initial adjustment
		raw, err := json.Marshal(req)
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}

		order.Adjustments = append(order.Adjustments, models.OrderAdjustment{
			ID: models.OrderAdjustmentID{
				OrderID:   orderID,
				Reference: req.Reference,
				CreatedAt: now,
				Status:    models.ORDER_STATUS_PENDING,
			},
			Reference: req.Reference,
			CreatedAt: now,
			Status:    models.ORDER_STATUS_PENDING,
			Metadata:  req.Metadata,
			Raw:       raw,
		})

		// Determine if we should send to exchange (default: true)
		sendToExchange := true
		if req.SendToExchange != nil {
			sendToExchange = *req.SendToExchange
		}

		task, err := backend.OrdersCreate(ctx, order, sendToExchange, req.WaitResult)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		// Return order with any validation warnings and task info
		response := CreateOrderResponse{
			Order:    order,
			Warnings: warnings,
		}

		// If a task was created, add task ID to the response
		if task.ID.Reference != "" {
			response.TaskID = &task.ID
		}

		api.Created(w, response)
	}
}

// validateOrderAgainstConnector validates the order against connector's tradable assets
// Returns validation warnings (non-blocking) and an error if validation fails completely
func validateOrderAgainstConnector(
	ctx context.Context,
	b backend.Backend,
	connectorID models.ConnectorID,
	sourceAsset, targetAsset string,
	quantity *big.Int,
) ([]OrderValidationWarning, error) {
	var warnings []OrderValidationWarning

	// Try to get tradable assets from connector
	assets, err := b.ConnectorsGetTradableAssets(ctx, connectorID)
	if err != nil {
		// If we can't get tradable assets, add a warning but don't fail
		warnings = append(warnings, OrderValidationWarning{
			Code:    "TRADABLE_ASSETS_UNAVAILABLE",
			Message: "Could not validate trading pair - tradable assets unavailable from connector",
		})
		return warnings, nil
	}

	// Build the pair in standard format
	pair := sourceAsset + "/" + targetAsset

	// Look for matching trading pair
	var matchedAsset *models.TradableAsset
	for i, asset := range assets {
		// Check various pair formats
		if strings.EqualFold(asset.Pair, pair) ||
			strings.EqualFold(asset.BaseAsset+"/"+asset.QuoteAsset, pair) ||
			strings.EqualFold(asset.BaseAsset+asset.QuoteAsset, sourceAsset+targetAsset) {
			matchedAsset = &assets[i]
			break
		}
	}

	if matchedAsset == nil {
		return nil, fmt.Errorf("trading pair %s is not supported by this connector", pair)
	}

	// Check if trading is enabled
	if matchedAsset.Status != "" && !strings.EqualFold(matchedAsset.Status, "online") &&
		!strings.EqualFold(matchedAsset.Status, "TRADING") &&
		!strings.EqualFold(matchedAsset.Status, "Enabled") {
		return nil, fmt.Errorf("trading pair %s is currently not available (status: %s)", pair, matchedAsset.Status)
	}

	// Validate minimum order size if available
	if matchedAsset.MinOrderSize != "" && quantity != nil {
		minSize, ok := new(big.Int).SetString(matchedAsset.MinOrderSize, 10)
		if !ok {
			// Try parsing as decimal
			minSize = parseMinOrderSize(matchedAsset.MinOrderSize, matchedAsset.SizePrecision)
		}
		if minSize != nil && quantity.Cmp(minSize) < 0 {
			return nil, fmt.Errorf("order quantity %s is below minimum order size %s for %s",
				quantity.String(), matchedAsset.MinOrderSize, pair)
		}
	}

	return warnings, nil
}

// parseMinOrderSize parses a minimum order size string that may be in decimal format
func parseMinOrderSize(minOrderSizeStr string, precision int) *big.Int {
	if minOrderSizeStr == "" {
		return nil
	}

	// Handle decimal format (e.g., "0.0001")
	parts := strings.Split(minOrderSizeStr, ".")
	intPart := parts[0]
	fracPart := ""
	if len(parts) > 1 {
		fracPart = parts[1]
	}

	// Use the precision from the asset or default to 8
	if precision <= 0 {
		precision = 8
	}

	// Pad or truncate fractional part
	if len(fracPart) > precision {
		fracPart = fracPart[:precision]
	} else {
		for len(fracPart) < precision {
			fracPart += "0"
		}
	}

	combined := intPart + fracPart
	combined = strings.TrimLeft(combined, "0")
	if combined == "" {
		combined = "0"
	}

	result := new(big.Int)
	result.SetString(combined, 10)
	return result
}

func populateSpanFromOrderCreateRequest(span trace.Span, req CreateOrderRequest) {
	span.SetAttributes(attribute.String("reference", req.Reference))
	span.SetAttributes(attribute.String("connectorID", req.ConnectorID))
	span.SetAttributes(attribute.String("direction", req.Direction))
	span.SetAttributes(attribute.String("sourceAsset", req.SourceAsset))
	span.SetAttributes(attribute.String("targetAsset", req.TargetAsset))
	span.SetAttributes(attribute.String("type", req.Type))
	if req.BaseQuantityOrdered != nil {
		span.SetAttributes(attribute.String("baseQuantityOrdered", req.BaseQuantityOrdered.String()))
	}
	if req.LimitPrice != nil {
		span.SetAttributes(attribute.String("limitPrice", req.LimitPrice.String()))
	}
	if req.StopPrice != nil {
		span.SetAttributes(attribute.String("stopPrice", req.StopPrice.String()))
	}
	span.SetAttributes(attribute.String("timeInForce", req.TimeInForce))
	if req.ExpiresAt != nil {
		span.SetAttributes(attribute.String("expiresAt", req.ExpiresAt.String()))
	}
	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
