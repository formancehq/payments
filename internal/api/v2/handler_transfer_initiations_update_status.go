package v2

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

type deprecatedStatus string

const (
	VALIDATED deprecatedStatus = "VALIDATED"
	REJECTED  deprecatedStatus = "REJECTED"
)

type updateTransferInitiationStatusRequest struct {
	Status string `json:"status"`
}

func (r updateTransferInitiationStatusRequest) Validate() error {
	if r.Status != string(VALIDATED) && r.Status != string(REJECTED) {
		return errors.New("invalid status")
	}

	return nil
}

func transferInitiationsUpdateStatus(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_transferInitiationsUpdateStatus")
		defer span.End()

		span.SetAttributes(attribute.String("transferInitiationID", transferInitiationID(r)))
		id, err := models.PaymentInitiationIDFromString(transferInitiationID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		payload := updateTransferInitiationStatusRequest{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		span.SetAttributes(attribute.String("status", payload.Status))

		if err := payload.Validate(); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		switch deprecatedStatus(payload.Status) {
		case VALIDATED:
			_, err = backend.PaymentInitiationsApprove(ctx, id, true)
		case REJECTED:
			err = backend.PaymentInitiationsReject(ctx, id)
		default:
			// Not possible since we already validated the status in the request
		}
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.NoContent(w)
	}
}
