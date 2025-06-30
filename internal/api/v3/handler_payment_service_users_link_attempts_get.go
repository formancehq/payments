package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// TODO(polo): add tests
func paymentServiceUsersLinkAttemptGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentServiceUsersLinkAttemptGet")
		defer span.End()

		span.SetAttributes(attribute.String("attemptID", attemptID(r)))
		attemptID, err := uuid.Parse(attemptID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		attempt, err := backend.PaymentServiceUsersLinkAttemptsGet(ctx, attemptID)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Ok(w, attempt)
	}
}
