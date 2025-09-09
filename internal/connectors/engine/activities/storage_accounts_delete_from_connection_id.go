package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsDeleteFromConnectionID(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return temporalStorageError(a.storage.AccountsDeleteFromOpenBankingConnectionID(ctx, psuID, connectorID, connectionID))
}

var StorageAccountsDeleteFromConnectionIDActivity = Activities{}.StorageAccountsDeleteFromConnectionID

func StorageAccountsDeleteFromConnectionID(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return executeActivity(ctx, StorageAccountsDeleteFromConnectionIDActivity, nil, psuID, connectorID, connectionID)
}
