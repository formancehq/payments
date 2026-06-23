package activities

import (
	"context"

	"github.com/formancehq/go-libs/v5/pkg/storage/bun/paginate"
	"github.com/formancehq/payments/pkg/domain/models"
	"github.com/formancehq/payments/internal/storage"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageInstancesListSchedulesAboveErrorThreshold(ctx context.Context, connectorID models.ConnectorID, cursor *string) (*paginate.Cursor[models.Instance], error) {
	var q storage.ListInstancesQuery
	if cursor != nil && *cursor != "" {
		if err := paginate.UnmarshalCursor(*cursor, &q); err != nil {
			return nil, err
		}
	} else {
		q = storage.NewListInstancesQuery(paginate.NewPaginatedQueryOptions(storage.InstanceQuery{}))
	}

	result, err := a.storage.InstancesListSchedulesAboveErrorThreshold(ctx, connectorID, a.healthCheckErrorThreshold, q)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return result, nil
}

var StorageInstancesListSchedulesAboveErrorThresholdActivity = Activities{}.StorageInstancesListSchedulesAboveErrorThreshold

func StorageInstancesListSchedulesAboveErrorThreshold(ctx workflow.Context, connectorID models.ConnectorID, cursor *string) (*paginate.Cursor[models.Instance], error) {
	var result paginate.Cursor[models.Instance]
	if err := executeActivity(ctx, StorageInstancesListSchedulesAboveErrorThresholdActivity, &result, connectorID, cursor); err != nil {
		return nil, err
	}
	return &result, nil
}
