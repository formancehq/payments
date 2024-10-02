package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/api"
	"github.com/formancehq/go-libs/bun/bunpaginate"
	"github.com/formancehq/go-libs/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
)

func paymentInitiationPaymentsList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentInitiationPaymentsList")
		defer span.End()

		query, err := bunpaginate.Extract[storage.ListPaymentInitiationRelatedPaymentsQuery](r, func() (*storage.ListPaymentInitiationRelatedPaymentsQuery, error) {
			options, err := getPagination(r, storage.PaymentInitiationRelatedPaymentsQuery{})
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
