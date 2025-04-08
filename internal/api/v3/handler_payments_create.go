package v3

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreatePaymentRequest struct {
	Reference            string                             `json:"reference" validate:"required,gte=3,lte=1000"`
	ConnectorID          string                             `json:"connectorID" validate:"required,connectorID"`
	CreatedAt            time.Time                          `json:"createdAt" validate:"required,lte=now"`
	Type                 string                             `json:"type" validate:"required,paymentType"`
	InitialAmount        *big.Int                           `json:"initialAmount" validate:""`
	Amount               *big.Int                           `json:"amount" validate:"required"`
	Asset                string                             `json:"asset" validate:"required,asset"`
	Scheme               string                             `json:"scheme" validate:"required,paymentScheme"`
	SourceAccountID      *string                            `json:"sourceAccountID" validate:"omitempty,accountID"`
	DestinationAccountID *string                            `json:"destinationAccountID" validate:"omitempty,accountID"`
	Metadata             map[string]string                  `json:"metadata" validate:""`
	Adjustments          []CreatePaymentsAdjustmentsRequest `json:"adjustments" validate:"min=1,dive"`
}

type CreatePaymentsAdjustmentsRequest struct {
	Reference string            `json:"reference" validate:"required,gte=3,lte=1000"`
	CreatedAt time.Time         `json:"createdAt" validate:"required,lte=now"`
	Status    string            `json:"status" validate:"required,paymentStatus"`
	Amount    *big.Int          `json:"amount" validate:"required"`
	Asset     *string           `json:"asset" validate:"required,asset"`
	Metadata  map[string]string `json:"metadata" validate:""`
}

func paymentsCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_paymentsCreate")
		defer span.End()

		var req CreatePaymentRequest
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromPaymentCreateRequest(span, req)

		if _, err := validator.Validate(req); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID := models.MustConnectorIDFromString(req.ConnectorID)
		paymentType := models.MustPaymentTypeFromString(req.Type)
		pid := models.PaymentID{
			PaymentReference: models.PaymentReference{
				Reference: req.Reference,
				Type:      paymentType,
			},
			ConnectorID: connectorID,
		}

		payment := models.Payment{
			ID:            pid,
			ConnectorID:   connectorID,
			Reference:     req.Reference,
			CreatedAt:     req.CreatedAt.UTC(),
			Type:          paymentType,
			InitialAmount: req.InitialAmount,
			Amount:        req.Amount,
			Asset:         req.Asset,
			Scheme:        models.MustPaymentSchemeFromString(req.Scheme),
			SourceAccountID: func() *models.AccountID {
				if req.SourceAccountID == nil {
					return nil
				}
				return pointer.For(models.MustAccountIDFromString(*req.SourceAccountID))
			}(),
			DestinationAccountID: func() *models.AccountID {
				if req.DestinationAccountID == nil {
					return nil
				}
				return pointer.For(models.MustAccountIDFromString(*req.DestinationAccountID))
			}(),
			Metadata: req.Metadata,
		}

		for _, adj := range req.Adjustments {
			raw, err := json.Marshal(adj)
			if err != nil {
				otel.RecordError(span, err)
				common.InternalServerError(w, r, err)
				return
			}
			status := models.MustPaymentStatusFromString(adj.Status)

			payment.Adjustments = append(payment.Adjustments, models.PaymentAdjustment{
				ID: models.PaymentAdjustmentID{
					PaymentID: pid,
					Reference: adj.Reference,
					CreatedAt: adj.CreatedAt.UTC(),
					Status:    status,
				},
				Reference: adj.Reference,
				CreatedAt: adj.CreatedAt,
				Status:    status,
				Amount:    adj.Amount,
				Asset:     adj.Asset,
				Metadata:  adj.Metadata,
				Raw:       raw,
			})
		}

		err = backend.PaymentsCreate(ctx, payment)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, payment)
	}
}

func populateSpanFromPaymentCreateRequest(span trace.Span, req CreatePaymentRequest) {
	span.SetAttributes(attribute.String("reference", req.Reference))
	span.SetAttributes(attribute.String("connectorID", req.ConnectorID))
	span.SetAttributes(attribute.String("createdAt", req.CreatedAt.String()))
	span.SetAttributes(attribute.String("type", req.Type))
	span.SetAttributes(attribute.String("initialAmount", req.InitialAmount.String()))
	span.SetAttributes(attribute.String("amount", req.Amount.String()))
	span.SetAttributes(attribute.String("asset", req.Asset))
	span.SetAttributes(attribute.String("scheme", req.Scheme))
	if req.SourceAccountID != nil {
		span.SetAttributes(attribute.String("sourceAccountID", *req.SourceAccountID))
	}
	if req.DestinationAccountID != nil {
		span.SetAttributes(attribute.String("destinationAccountID", *req.DestinationAccountID))
	}
	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}

	for i, adj := range req.Adjustments {
		span.SetAttributes(attribute.String(fmt.Sprintf("adjustments[%d].reference", i), adj.Reference))
		span.SetAttributes(attribute.String(fmt.Sprintf("adjustments[%d].createdAt", i), adj.CreatedAt.String()))
		span.SetAttributes(attribute.String(fmt.Sprintf("adjustments[%d].status", i), adj.Status))
		span.SetAttributes(attribute.String(fmt.Sprintf("adjustments[%d].amount", i), adj.Amount.String()))
		if adj.Asset != nil {
			span.SetAttributes(attribute.String(fmt.Sprintf("adjustments[%d].asset", i), *adj.Asset))
		}
		for k, v := range adj.Metadata {
			span.SetAttributes(attribute.String(fmt.Sprintf("adjustments[%d].metadata[%s]", i, k), v))
		}
	}
}
