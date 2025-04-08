package v2

import (
	"encoding/json"
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

type CreateTransferInitiationRequest struct {
	Reference            string            `json:"reference" validate:"required"`
	ScheduledAt          time.Time         `json:"scheduledAt" validate:""`
	Description          string            `json:"description" validate:""`
	SourceAccountID      string            `json:"sourceAccountID" validate:"omitempty,accountID"`
	DestinationAccountID string            `json:"destinationAccountID" validate:"required,accountID"`
	ConnectorID          string            `json:"connectorID" validate:"required,connectorID"`
	Type                 string            `json:"type" validate:"required,paymentInitiationType"`
	Amount               *big.Int          `json:"amount" validate:"required"`
	Asset                string            `json:"asset" validate:"required,asset"`
	Validated            bool              `json:"validated" validate:""`
	Metadata             map[string]string `json:"metadata" validate:""`
}

func transferInitiationsCreate(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_transferInitiationsCreate")
		defer span.End()

		payload := CreateTransferInitiationRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		setSpanAttributesFromRequest(span, payload)

		if _, err := validator.Validate(payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		connectorID, err := models.ConnectorIDFromString(payload.ConnectorID)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		pi := models.PaymentInitiation{
			ID: models.PaymentInitiationID{
				Reference:   payload.Reference,
				ConnectorID: connectorID,
			},
			ConnectorID: connectorID,
			Reference:   payload.Reference,
			CreatedAt:   time.Now(),
			ScheduledAt: payload.ScheduledAt,
			Description: payload.Description,
			Type:        models.MustPaymentInitiationTypeFromString(payload.Type),
			Amount:      payload.Amount,
			Asset:       payload.Asset,
			Metadata:    payload.Metadata,
		}

		if payload.SourceAccountID != "" {
			pi.SourceAccountID = pointer.For(models.MustAccountIDFromString(payload.SourceAccountID))
		}

		if payload.DestinationAccountID != "" {
			pi.DestinationAccountID = pointer.For(models.MustAccountIDFromString(payload.DestinationAccountID))
		}

		_, err = backend.PaymentInitiationsCreate(ctx, pi, payload.Validated, true)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		resp := translatePaymentInitiationToResponse(&pi)
		lastAdjustment, err := backend.PaymentInitiationAdjustmentsGetLast(ctx, pi.ID)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		if lastAdjustment != nil {
			resp.Status = lastAdjustment.Status.String()
			resp.Error = func() string {
				if lastAdjustment.Error == nil {
					return ""
				}
				return lastAdjustment.Error.Error()
			}()
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[transferInitiationResponse]{
			Data: &resp,
		})
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}
	}
}

func setSpanAttributesFromRequest(span trace.Span, transfer CreateTransferInitiationRequest) {
	span.SetAttributes(
		attribute.String("reference", transfer.Reference),
		attribute.String("scheduledAt", transfer.ScheduledAt.String()),
		attribute.String("description", transfer.Description),
		attribute.String("sourceAccountID", transfer.SourceAccountID),
		attribute.String("destinationAccountID", transfer.DestinationAccountID),
		attribute.String("connectorID", transfer.ConnectorID),
		attribute.String("type", transfer.Type),
		attribute.String("amount", transfer.Amount.String()),
		attribute.String("asset", transfer.Asset),
		attribute.String("validated", transfer.Asset),
	)
}
