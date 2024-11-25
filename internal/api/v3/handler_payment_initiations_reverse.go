package v3

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type paymentInitiationsReverseRequest struct {
	Reference   string            `json:"reference"`
	Description string            `json:"description"`
	Amount      *big.Int          `json:"amount"`
	Asset       string            `json:"asset"`
	Metadata    map[string]string `json:"metadata"`
}

func (r *paymentInitiationsReverseRequest) Validate() error {
	if r.Reference == "" {
		return errors.New("reference is required")
	}

	if r.Amount == nil {
		return errors.New("amount is required")
	}

	if r.Asset == "" {
		return errors.New("asset is required")
	}

	return nil
}

type paymentInitiationsReverseResponse struct {
	PaymentInitiationReversalID string `json:"paymentInitiationReversalID"`
	TaskID                      string `json:"taskID"`
}

func paymentInitiationsReverse(backend backend.Backend) http.HandlerFunc {
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

		payload := paymentInitiationsReverseRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		populateSpanFromPaymentInitiationsReverseRequest(span, payload)

		if err := payload.Validate(); err != nil {
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

		api.Accepted(w, paymentInitiationsReverseResponse{
			PaymentInitiationReversalID: reversalID.String(),
			TaskID:                      task.ID.String(),
		})
	}
}

func populateSpanFromPaymentInitiationsReverseRequest(span trace.Span, r paymentInitiationsReverseRequest) {
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
