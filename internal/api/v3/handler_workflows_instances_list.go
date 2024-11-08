package v3

import (
	"net/http"

	"github.com/formancehq/go-libs/v2/api"
	"github.com/formancehq/go-libs/v2/bun/bunpaginate"
	"github.com/formancehq/go-libs/v2/pointer"
	"github.com/formancehq/go-libs/v2/query"
	"github.com/formancehq/payments/internal/api/backend"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/otel"
	"github.com/formancehq/payments/internal/storage"
	"go.opentelemetry.io/otel/attribute"
)

func workflowsInstancesList(backend backend.Backend) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer().Start(r.Context(), "v3_workflowsInstancesList")
		defer span.End()

		span.SetAttributes(attribute.String("connectorID", connectorID(r)))
		connectorID, err := models.ConnectorIDFromString(connectorID(r))
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrInvalidID, err)
			return
		}

		span.SetAttributes(attribute.String("scheduleID", scheduleID(r)))
		scheduleID := scheduleID(r)

		query, err := bunpaginate.Extract[storage.ListInstancesQuery](r, func() (*storage.ListInstancesQuery, error) {
			pageSize, err := bunpaginate.GetPageSize(r)
			if err != nil {
				return nil, err
			}
			span.SetAttributes(attribute.Int64("pageSize", int64(pageSize)))

			options := pointer.For(bunpaginate.NewPaginatedQueryOptions(storage.InstanceQuery{}).WithPageSize(pageSize))
			options = pointer.For(options.WithQueryBuilder(
				query.And(
					query.Match("connector_id", connectorID),
					query.Match("schedule_id", scheduleID),
				),
			))

			return pointer.For(storage.NewListInstancesQuery(*options)), nil
		})
		if err != nil {
			otel.RecordError(span, err)
			api.BadRequest(w, ErrValidation, err)
			return
		}

		cursor, err := backend.WorkflowsInstancesList(ctx, *query)
		if err != nil {
			otel.RecordError(span, err)
			handleServiceErrors(w, r, err)
			return
		}

		api.RenderCursor(w, *cursor)
	}
}
