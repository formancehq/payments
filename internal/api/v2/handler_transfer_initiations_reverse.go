package v2

import (
	"encoding/json"
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

type reverseTransferInitiationRequest struct {
	Reference   string            `json:"reference" validate:"required,gt=3,lt=1000"`
	Description string            `json:"description" validate:"omitempty,lt=10000"`
	Amount      *big.Int          `json:"amount" validate:"required"`
	Asset       string            `json:"asset" validate:"required,asset"`
	Metadata    map[string]string `json:"metadata" validate:""`
}

func transferInitiationsReverse(backend backend.Backend, validator *validation.Validator) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_transferInitiationsReverse")
		defer span.End()

		span.SetAttributes(attribute.String("transferInitiationID", transferInitiationID(r)))
		id, err := models.PaymentInitiationIDFromString(transferInitiationID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		payload := reverseTransferInitiationRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		setReversalSpanAttributesFromRequest(span, payload)

		if _, err := validator.Validate(payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		_, err = backend.PaymentInitiationReversalsCreate(ctx, models.PaymentInitiationReversal{
			ID: models.PaymentInitiationReversalID{
				Reference:   payload.Reference,
				ConnectorID: id.ConnectorID,
			},
			ConnectorID:         id.ConnectorID,
			PaymentInitiationID: id,
			Reference:           payload.Reference,
			CreatedAt:           time.Now(),
			Description:         payload.Description,
			Amount:              payload.Amount,
			Asset:               payload.Asset,
			Metadata:            payload.Metadata,
		}, true)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.NoContent(w)
	}
}

func setReversalSpanAttributesFromRequest(span trace.Span, reversal reverseTransferInitiationRequest) {
	span.SetAttributes(
		attribute.String("reference", reversal.Reference),
		attribute.String("description", reversal.Description),
		attribute.String("asset", reversal.Asset),
		attribute.String("amount", reversal.Amount.String()),
	)
}
