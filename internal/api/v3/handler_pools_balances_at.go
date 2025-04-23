package v3

import (
	"fmt"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.opentelemetry.io/otel/attribute"
)

func poolsBalancesAt(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_poolsBalancesAt")
		defer span.End()

		span.SetAttributes(attribute.String("poolID", poolID(r)))
		id, err := uuid.Parse(poolID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		atTime := r.URL.Query().Get("at")
		span.SetAttributes(attribute.String("atTime", atTime))
		if atTime == "" {
			err := fmt.Errorf("query param 'at' is missing")
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		at, err := time.Parse(time.RFC3339, atTime)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, errors.Wrap(err, "invalid atTime"))
			return
		}
		if time.Now().Before(at) {
			err := fmt.Errorf("query param 'at' cannot be in the future")
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		balances, err := backend.PoolsBalancesAt(ctx, id, at)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Ok(w, balances)
	}
}
