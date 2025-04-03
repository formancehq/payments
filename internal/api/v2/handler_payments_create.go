package v2

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type CreatePaymentRequest struct {
	Reference            string            `json:"reference" validate:"required,gte=3,lte=1000"`
	ConnectorID          string            `json:"connectorID" validate:"required,connectorID"`
	CreatedAt            time.Time         `json:"createdAt" validate:"required,lte=now"`
	Type                 string            `json:"type" validate:"required,paymentType"`
	Amount               *big.Int          `json:"amount" validate:"required"`
	Asset                string            `json:"asset" validate:"required,asset"`
	Scheme               string            `json:"scheme" validate:"required,paymentScheme"`
	Status               string            `json:"status" validate:"required,paymentStatus"`
	SourceAccountID      *string           `json:"sourceAccountID" validate:"omitempty,accountID"`
	DestinationAccountID *string           `json:"destinationAccountID" validate:"omitempty,accountID"`
	Metadata             map[string]string `json:"metadata" validate:""`
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
		status := models.MustPaymentStatusFromString(req.Status)
		raw, err := json.Marshal(req)
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}
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
			InitialAmount: req.Amount,
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

		// Create adjustments from main payments to keep the compatibility with the old API
		payment.Adjustments = []models.PaymentAdjustment{
			{
				ID: models.PaymentAdjustmentID{
					PaymentID: pid,
					Reference: req.Reference,
					CreatedAt: req.CreatedAt,
					Status:    status,
				},
				Reference: req.Reference,
				CreatedAt: req.CreatedAt,
				Status:    status,
				Amount:    req.Amount,
				Asset:     &req.Asset,
				Metadata:  req.Metadata,
				Raw:       raw,
			},
		}

		err = backend.PaymentsCreate(ctx, payment)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		// Compatibility with old API
		data := PaymentResponse{
			ID:            payment.ID.String(),
			Reference:     payment.Reference,
			Type:          payment.Type.String(),
			Provider:      toV2Provider(payment.ConnectorID.Provider),
			ConnectorID:   payment.ConnectorID.String(),
			Status:        payment.Status.String(),
			Amount:        payment.Amount,
			InitialAmount: payment.InitialAmount,
			Scheme:        toV2PaymentScheme(payment.Scheme),
			Asset:         payment.Asset,
			CreatedAt:     payment.CreatedAt,
			Metadata:      payment.Metadata,
		}

		if payment.SourceAccountID != nil {
			data.SourceAccountID = payment.SourceAccountID.String()
		}

		if payment.DestinationAccountID != nil {
			data.DestinationAccountID = payment.DestinationAccountID.String()
		}

		data.Adjustments = make([]paymentAdjustment, len(payment.Adjustments))
		for i := range payment.Adjustments {
			data.Adjustments[i] = paymentAdjustment{
				Reference: payment.Adjustments[i].ID.Reference,
				CreatedAt: payment.Adjustments[i].CreatedAt,
				Status:    payment.Adjustments[i].Status.String(),
				Amount:    payment.Adjustments[i].Amount,
				Raw:       payment.Adjustments[i].Raw,
			}
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[PaymentResponse]{
			Data: &data,
		})
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}
	}
}

func populateSpanFromPaymentCreateRequest(span trace.Span, req CreatePaymentRequest) {
	span.SetAttributes(attribute.String("reference", req.Reference))
	span.SetAttributes(attribute.String("connectorID", req.ConnectorID))
	span.SetAttributes(attribute.String("createdAt", req.CreatedAt.String()))
	span.SetAttributes(attribute.String("type", req.Type))
	span.SetAttributes(attribute.String("amount", req.Amount.String()))
	span.SetAttributes(attribute.String("asset", req.Asset))
	span.SetAttributes(attribute.String("scheme", req.Scheme))
	span.SetAttributes(attribute.String("status", req.Status))
	if req.SourceAccountID != nil {
		span.SetAttributes(attribute.String("sourceAccountID", *req.SourceAccountID))
	}
	if req.DestinationAccountID != nil {
		span.SetAttributes(attribute.String("destinationAccountID", *req.DestinationAccountID))
	}
	for k, v := range req.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
