package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDeleteFromPSUIDAndConnectorID(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.PaymentsDeleteFromConnectorIDAndPSUID(ctx, connectorID, psuID))
}

var StoragePaymentsDeleteFromPSUIDAndConnectorIDActivity = Activities{}.StoragePaymentsDeleteFromPSUIDAndConnectorID

func StoragePaymentsDeleteFromPSUIDAndConnectorID(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePaymentsDeleteFromPSUIDAndConnectorIDActivity, nil, psuID, connectorID)
}
