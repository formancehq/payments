package v2

import (
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

func connectorsConfigUpdate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_connectorsConfigUpdate")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		rawConfig, err := io.ReadAll(r.Body)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}
		err = backend.ConnectorsConfigUpdate(ctx, connectorID, rawConfig)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.NoContent(w)
	}
}
