package activities

import (
	"context"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsDeleteFromConnectionID(ctx context.Context, psuID uuid.UUID, connectionID string) error {
	return temporalStorageError(a.storage.AccountsDeleteFromOpenBankingConnectionID(ctx, psuID, connectionID))
}

var StorageAccountsDeleteFromConnectionIDActivity = Activities{}.StorageAccountsDeleteFromConnectionID

func StorageAccountsDeleteFromConnectionID(ctx workflow.Context, psuID uuid.UUID, connectionID string) error {
	return executeActivity(ctx, StorageAccountsDeleteFromConnectionIDActivity, nil, psuID, connectionID)
}
