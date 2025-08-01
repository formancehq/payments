package v3

import (
	"io"
	"net/http"

	"github.com/formancehq/go-libs/v3/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

func connectorsWebhooks(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsWebhooks")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil && err != io.EOF {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}

		headers := r.Header
		queryValues := r.URL.Query()
		path := r.URL.Path
		username, password, ok := r.BasicAuth()

		webhook := models.Webhook{
			ID:          uuid.New().String(),
			ConnectorID: connectorID,
			QueryValues: queryValues,
			Headers:     headers,
			Body:        body,
		}

		if ok {
			webhook.BasicAuth = &models.BasicAuth{
				Username: username,
				Password: password,
			}
		}

		err = backend.ConnectorsHandleWebhooks(ctx, r.URL.String(), path, webhook)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.RawOk(w, nil)
	}
}
