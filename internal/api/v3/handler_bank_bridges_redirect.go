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

func bankBridgesRedirect(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_bankBridgesRedirectURI")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		headers := r.Header
		queryValues := r.URL.Query()
		body, err := io.ReadAll(r.Body)
		if err != nil && err != io.EOF {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		redirectURL, err := backend.PaymentServiceUsersCompleteLinkFlow(
			ctx,
			connectorID,
			models.HTTPCallInformation{
				QueryValues: queryValues,
				Headers:     headers,
				Body:        body,
			},
		)
		if err != nil {
			handleServiceErrors(w, r, err)
			return
		}

		if queryValues.Get(models.NoRedirectQueryParamID) == "true" {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		http.Redirect(w, r, redirectURL, http.StatusTemporaryRedirect)
	}
}
