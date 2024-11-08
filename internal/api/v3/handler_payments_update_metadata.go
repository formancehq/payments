package v3

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

func validatePaymentsMetadata(metadata map[string]string) error {
	if len(metadata) == 0 {
		return errors.New("metadata must be provided")
	}
	return nil
}

func paymentsUpdateMetadata(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentsUpdateMetadata")
		defer span.End()

		span.SetAttributes(attribute.String("paymentID", paymentID(r)))
		id, err := models.PaymentIDFromString(paymentID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		var metadata map[string]string
		err = json.NewDecoder(r.Body).Decode(&metadata)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		populateSpanFromUpdateMetadataRequest(span, metadata)

		if err := validatePaymentsMetadata(metadata); err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		err = backend.PaymentsUpdateMetadata(ctx, id, metadata)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.NoContent(w)
	}
}
