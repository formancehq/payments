package v2

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/go-libs/v3/pointer"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
)

func poolsList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_poolsList")
		defer span.End()

		query, err := bunpaginate.Extract[storage.ListPoolsQuery](r, func() (*storage.ListPoolsQuery, error) {
			options, err := getPagination(span, r, storage.PoolQuery{})
			if err != nil {
				return nil, err
			}
			return pointer.For(storage.NewListPoolsQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.PoolsList(ctx, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		data := make([]*PoolResponse, len(cursor.Data))
		for i := range cursor.Data {
			data[i] = &PoolResponse{
				ID:   cursor.Data[i].ID.String(),
				Name: cursor.Data[i].Name,
			}

			accounts := make([]string, len(cursor.Data[i].PoolAccounts))
			for j := range cursor.Data[i].PoolAccounts {
				accounts[j] = cursor.Data[i].PoolAccounts[j].String()
			}

			data[i].Accounts = accounts
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[*PoolResponse]{
			Cursor: &bunpaginate.Cursor[*PoolResponse]{
				PageSize: cursor.PageSize,
				HasMore:  cursor.HasMore,
				Previous: cursor.Previous,
				Next:     cursor.Next,
				Data:     data,
			},
		})
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}
	}
}
