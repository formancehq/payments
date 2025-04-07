package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
	"go.opentelemetry.io/otel/attribute"
)

func paymentInitiationPaymentsList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentInitiationPaymentsList")
		defer span.End()

		query, err := bunpaginate.Extract[storage.ListPaymentInitiationRelatedPaymentsQuery](r, func() (*storage.ListPaymentInitiationRelatedPaymentsQuery, error) {
			options, err := getPagination(span, r, storage.PaymentInitiationRelatedPaymentsQuery{})
			if err != nil {
				return nil, err
			}
			return pointer.For(storage.NewListPaymentInitiationRelatedPaymentsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		span.SetAttributes(attribute.String("paymentInitiationID", paymentInitiationID(r)))
		id, err := models.PaymentInitiationIDFromString(paymentInitiationID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		cursor, err := backend.PaymentInitiationRelatedPaymentsList(ctx, id, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.RenderCursor(w, *cursor)
	}
}
