package v2

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
)

func transferInitiationsList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_transferInitiationsList")
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

		data := make([]transferInitiationResponse, len(cursor.Data))
		for i := range cursor.Data {
			data[i] = translatePaymentInitiationToResponse(&cursor.Data[i])

			lastAdjustment, err := backend.PaymentInitiationAdjustmentsGetLast(ctx, cursor.Data[i].ID)
			if err != nil {
				otel.RecordError(span, err)
				handleServiceErrors(w, r, err)
				return
			}

			if lastAdjustment != nil {
				data[i].Status = lastAdjustment.Status.String()
				data[i].Error = func() string {
					if lastAdjustment.Error == nil {
						return ""
					}
					return lastAdjustment.Error.Error()
				}()
			}
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[transferInitiationResponse]{
			Cursor: &bunpaginate.Cursor[transferInitiationResponse]{
				PageSize: cursor.PageSize,
				HasMore:  cursor.HasMore,
				Previous: cursor.Previous,
				Next:     cursor.Next,
				Data:     data,
			},
		})
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
	}
}
