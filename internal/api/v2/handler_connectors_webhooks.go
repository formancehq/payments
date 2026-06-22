package v2

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
)

// v2WebhookProviders is the set of connectors that ever registered webhook URLs
// in v2 (/connectors/webhooks/{provider}/{connectorID}/). Only Adyen and
// MangoPay did, so any other provider in the path is rejected with 404 rather
// than silently accepted (EN-1091).
var v2WebhookProviders = map[string]struct{}{
	"adyen":    {},
	"mangopay": {},
}

func connectorsWebhooks(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_connectorsWebhooks")
		defer span.End()

		provider := strings.ToLower(connectorProvider(r))
		span.SetAttributes(attribute.String("provider", provider))
		if _, ok := v2WebhookProviders[provider]; !ok {
			err := fmt.Errorf("webhooks are not supported for connector %q in v2", connectorProvider(r))
			otel.RecordError(span, err)
			api.NotFound(w, err)
			return
		}

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
