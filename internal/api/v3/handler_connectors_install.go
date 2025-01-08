package v3

import (
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

var connectorConfigMaxBytes int64 = 500000

func connectorsInstall(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsInstall")
		defer span.End()

		body := http.MaxBytesReader(w, r.Body, connectorConfigMaxBytes)
		config, err := io.ReadAll(body)
		if err != nil {
			otel.RecordError(span, err)
			if _, ok := err.(*http.MaxBytesError); ok {
				api.WriteErrorResponse(w, http.StatusRequestEntityTooLarge, ErrMissingOrInvalidBody, err)
				return
			}
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		span.SetAttributes(attribute.String("config", string(config)))
		span.SetAttributes(attribute.String("provider", connector(r)))

		provider := connector(r)

		connectorID, err := backend.ConnectorsInstall(ctx, provider, config)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Accepted(w, connectorID.String())
	}
}
