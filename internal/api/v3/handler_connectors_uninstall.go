package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v5/pkg/transport/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/pkg/domain/models"
	"go.opentelemetry.io/otel/attribute"
)

type ConnectorUninstallResponse struct {
	TaskID string `json:"taskID"`
}

func connectorsUninstall(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_connectorsUninstall")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		task, err := backend.ConnectorsUninstall(ctx, connectorID)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.Accepted(w, ConnectorUninstallResponse{
			TaskID: task.ID.String(),
		})
	}
}
