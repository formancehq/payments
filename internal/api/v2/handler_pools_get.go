package v2

import (
	"encoding/json"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

func poolsGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_poolsGet")
		defer span.End()

		span.SetAttributes(attribute.String("poolID", poolID(r)))
		id, err := uuid.Parse(poolID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		pool, err := backend.PoolsGet(ctx, id)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		data := &PoolResponse{
			ID:   pool.ID.String(),
			Name: pool.Name,
		}

		accounts := make([]string, len(pool.PoolAccounts))
		for i := range pool.PoolAccounts {
			accounts[i] = pool.PoolAccounts[i].AccountID.String()
		}
		data.Accounts = accounts

		err = json.NewEncoder(w).Encode(api.BaseResponse[PoolResponse]{
			Data: data,
		})
		if err != nil {
			otel.RecordError(span, err)
			api.InternalServerError(w, r, err)
			return
		}
	}
}
