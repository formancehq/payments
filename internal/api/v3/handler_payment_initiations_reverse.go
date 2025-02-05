package v3

import (
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/validation"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type PaymentInitiationsReverseRequest struct {
	Reference   string            `json:"reference" validate:"required,gt=3,lt=1000"`
	Description string            `json:"description" validate:"omitempty,lt=10000"`
	Amount      *big.Int          `json:"amount" validate:"required"`
	Asset       string            `json:"asset" validate:"required,asset"`
	Metadata    map[string]string `json:"metadata" validate:""`
}

type PaymentInitiationsReverseResponse struct {
	PaymentInitiationReversalID string `json:"paymentInitiationReversalID"`
	TaskID                      string `json:"taskID"`
}

func paymentInitiationsReverse(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentInitiationsReverse")
		defer span.End()

		span.SetAttributes(attribute.String("paymentInitiationID", paymentInitiationID(r)))
		id, err := models.PaymentInitiationIDFromString(paymentInitiationID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		payload := PaymentInitiationsReverseRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromPaymentInitiationsReverseRequest(span, payload)

		if _, err := validator.Validate(payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		reversalID := models.PaymentInitiationReversalID{
			Reference:   payload.Reference,
			ConnectorID: id.ConnectorID,
		}
		task, err := backend.PaymentInitiationReversalsCreate(ctx, models.PaymentInitiationReversal{
			ID:                  reversalID,
			ConnectorID:         id.ConnectorID,
			PaymentInitiationID: id,
			Reference:           payload.Reference,
			CreatedAt:           time.Now(),
			Description:         payload.Description,
			Amount:              payload.Amount,
			Asset:               payload.Asset,
			Metadata:            payload.Metadata,
		}, false)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Accepted(w, PaymentInitiationsReverseResponse{
			PaymentInitiationReversalID: reversalID.String(),
			TaskID:                      task.ID.String(),
		})
	}
}

func populateSpanFromPaymentInitiationsReverseRequest(span trace.Span, r PaymentInitiationsReverseRequest) {
	span.SetAttributes(
		attribute.String("reference", r.Reference),
		attribute.String("description", r.Description),
		attribute.String("asset", r.Asset),
	)

	if r.Amount != nil {
		span.SetAttributes(attribute.String("amount", r.Amount.String()))
	}

	for k, v := range r.Metadata {
		span.SetAttributes(attribute.String(fmt.Sprintf("metadata[%s]", k), v))
	}
}
