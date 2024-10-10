package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
)

func paymentInitiationsGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentInitiationsGet")
		defer span.End()

		id, err := models.PaymentInitiationIDFromString(paymentInitiationID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		paymentInitiation, err := backend.PaymentInitiationsGet(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		lastAdjustment, err := backend.PaymentInitiationAdjustmentsGetLast(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		res := models.PaymentInitiationExpanded{
			PaymentInitiation: *paymentInitiation,
			Status:            lastAdjustment.Status.String(),
			Error:             lastAdjustment.Error,
		}

		api.Ok(w, res)
	}
}
