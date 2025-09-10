package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsDeleteFromPSUIDAndConnectorID(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.AccountsDeleteFromConnectorIDAndPSUID(ctx, connectorID, psuID))
}

var StorageAccountsDeleteFromPSUIDAndConnectorIDActivity = Activities{}.StorageAccountsDeleteFromPSUIDAndConnectorID

func StorageAccountsDeleteFromPSUIDAndConnectorID(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageAccountsDeleteFromPSUIDAndConnectorIDActivity, nil, psuID, connectorID)
}
