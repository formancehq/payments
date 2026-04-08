package activities

import (
	"context"

	"github.com/formancehq/go-libs/v3/bun/bunpaginate"
	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageInstancesGetScheduleErrors(ctx context.Context, connectorID models.ConnectorID, cursor *string) (*bunpaginate.Cursor[models.Instance], error) {
	var q storage.ListInstancesQuery
	if cursor != nil && *cursor != "" {
		if err := bunpaginate.UnmarshalCursor(*cursor, &q); err != nil {
			return nil, err
		}
	} else {
		q = storage.NewListInstancesQuery(bunpaginate.NewPaginatedQueryOptions(storage.InstanceQuery{}))
	}

	result, err := a.storage.InstancesGetScheduleErrors(ctx, connectorID, q)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return result, nil
}

var StorageInstancesGetScheduleErrorsActivity = Activities{}.StorageInstancesGetScheduleErrors

func StorageInstancesGetScheduleErrors(ctx workflow.Context, connectorID models.ConnectorID, cursor *string) (*bunpaginate.Cursor[models.Instance], error) {
	var result bunpaginate.Cursor[models.Instance]
	if err := executeActivity(ctx, StorageInstancesGetScheduleErrorsActivity, &result, connectorID, cursor); err != nil {
		return nil, err
	}
	return &result, nil
}
