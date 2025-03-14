package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageCounterPartiesDeleteRelatedAccounts(ctx context.Context, connectorID models.ConnectorID) error {
	return temporalStorageError(a.storage.CounterPartiesDeleteRelatedAccountFromConnectorID(ctx, connectorID))
}

var StorageCounterPartiesDeleteRelatedAccountsActivity = Activities{}.StorageCounterPartiesDeleteRelatedAccounts

func StorageCounterPartiesDeleteRelatedAccounts(ctx workflow.Context, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageCounterPartiesDeleteRelatedAccountsActivity, nil, connectorID)
}
