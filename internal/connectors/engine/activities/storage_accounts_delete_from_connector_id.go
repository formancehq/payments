package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsDeleteFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	return a.batchDeleteWithHeartbeat(ctx, connectorID, a.storage.AccountsDeleteFromConnectorIDBatch, "deleting accounts")
}

var StorageAccountsDeleteFromConnectorIDActivity = Activities{}.StorageAccountsDeleteFromConnectorID

func StorageAccountsDeleteFromConnectorID(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageAccountsDeleteFromConnectorIDActivity, nil, connectorID)
}