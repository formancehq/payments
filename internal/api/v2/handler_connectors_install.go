package v2

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

// NOTE: in order to maintain previous version compatibility, we need to keep the
// same response structure as the previous version of the API
type ConnectorInstallResponse struct {
	ConnectorID string `json:"connectorID"`
}

func connectorsInstall(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_connectorsInstall")
		defer span.End()

		provider := strings.ToLower(toV3Provider(connectorProvider(r)))

		span.SetAttributes(attribute.String("provider", provider))
		if provider == "" {
			otel.RecordError(span, errors.New("provider is required"))
			api.BadRequest(w, ErrValidation, errors.New("provider is required"))
			return
		}

		config, err := io.ReadAll(r.Body)
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrMissingOrInvalidBody, err)
			return
		}
		span.SetAttributes(attribute.String("config", string(config)))

		connectorID, err := backend.ConnectorsInstall(ctx, provider, config)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Created(w, ConnectorInstallResponse{
			ConnectorID: connectorID.String(),
		})
	}
}
