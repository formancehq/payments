package v2

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/api/common"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"go.opentelemetry.io/otel/attribute"
)

func tasksGet(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v2_tasksGet")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		span.SetAttributes(attribute.String("taskID", taskID(r)))
		taskID := taskID(r)

		schedule, err := backend.SchedulesGet(ctx, taskID, connectorID)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		raw, err := json.Marshal(schedule)
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}

		data := listTasksResponseElement{
			ID:          schedule.ID,
			ConnectorID: schedule.ConnectorID.String(),
			CreatedAt:   schedule.CreatedAt.Format(time.RFC3339),
			UpdatedAt:   schedule.CreatedAt.Format(time.RFC3339),
			Descriptor:  raw,
			Status:      "ACTIVE",
		}

		err = json.NewEncoder(w).Encode(api.BaseResponse[listTasksResponseElement]{
			Data: &data,
		})
		if err != nil {
			otel.RecordError(span, err)
			common.InternalServerError(w, r, err)
			return
		}
	}
}
