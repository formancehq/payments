package v3

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
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
	Type                string            `json:"type" validate:"required,oneof=MARKET LIMIT"`
	BaseQuantityOrdered *big.Int          `json:"baseQuantityOrdered" validate:"required"`
	LimitPrice          *big.Int          `json:"limitPrice,omitempty"`
	TimeInForce         string            `json:"timeInForce" validate:"omitempty,oneof=GOOD_UNTIL_CANCELLED GOOD_UNTIL_DATE_TIME IMMEDIATE_OR_CANCEL FILL_OR_KILL GTC GTD IOC FOK"`
	ExpiresAt           *time.Time        `json:"expiresAt,omitempty"`
	Metadata            map[string]string `json:"metadata"`
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

		// Validate GTD orders require expiresAt
		if (req.TimeInForce == "GOOD_UNTIL_DATE_TIME" || req.TimeInForce == "GTD") && req.ExpiresAt == nil {
			otel.RecordError(span, fmt.Errorf("expiresAt is required for GTD orders"))
			api.BadRequest(w, ErrValidation, fmt.Errorf("expiresAt is required for GTD (GOOD_UNTIL_DATE_TIME) orders"))
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)

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

		err = backend.OrdersCreate(ctx, order)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, order)
	}
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
	span.SetAttributes(attribute.String("timeInForce", req.TimeInForce))
	if req.ExpiresAt != nil {
		span.SetAttributes(attribute.String("expiresAt", req.ExpiresAt.String()))
	}
	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
