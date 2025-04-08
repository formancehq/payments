package v3

import (
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

func connectorsConfigUpdate(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsConfigUpdate")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		body := http.MaxBytesReader(w, r.Body, connectorConfigMaxBytes)
		rawConfig, err := io.ReadAll(body)
		if err != nil {
			otel.RecordError(span, err)
			if _, ok := err.(*http.MaxBytesError); ok {
				api.WriteErrorResponse(w, http.StatusRequestEntityTooLarge, ErrMissingOrInvalidBody, err)
				return
			}
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		span.SetAttributes(attribute.String("config", string(rawConfig)))
		err = backend.ConnectorsConfigUpdate(ctx, connectorID, rawConfig)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}
		api.NoContent(w)
	}
}
