package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePoolsRemoveAccountsFromConnectorID(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.PoolsRemoveAccountsFromConnectorID(ctx, connectorID))
}

var StoragePoolsRemoveAccountsFromConnectorIDActivity = Activities{}.StoragePoolsRemoveAccountsFromConnectorID

func StoragePoolsRemoveAccountsFromConnectorID(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePoolsRemoveAccountsFromConnectorIDActivity, nil, connectorID)
}
