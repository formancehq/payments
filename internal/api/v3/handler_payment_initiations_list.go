package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
)

func paymentInitiationsList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_paymentInitiationsList")
		defer span.End()

		query, err := bunpaginate.Extract[storage.ListPaymentInitiationsQuery](r, func() (*storage.ListPaymentInitiationsQuery, error) {
			options, err := getPagination(span, r, storage.PaymentInitiationQuery{})
			if err != nil {
				return nil, err
			}
			return pointer.For(storage.NewListPaymentInitiationsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.PaymentInitiationsList(ctx, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		pis := make([]models.PaymentInitiationExpanded, 0, len(cursor.Data))
		for _, pi := range cursor.Data {
			lastAdjustment, err := backend.PaymentInitiationAdjustmentsGetLast(ctx, pi.ID)
			if err != nil {
				otel.RecordError(span, err)
				handleServiceErrors(w, r, err)
				return
			}

			pis = append(pis, models.PaymentInitiationExpanded{
				PaymentInitiation: pi,
				Status:            lastAdjustment.Status.String(),
				Error:             lastAdjustment.Error,
			})
		}

		api.RenderCursor(w, bunpaginate.Cursor[models.PaymentInitiationExpanded]{
			PageSize: cursor.PageSize,
			HasMore:  cursor.HasMore,
			Previous: cursor.Previous,
			Next:     cursor.Next,
			Data:     pis,
		})
	}
}
