package activities

import (
	"context"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageAccountsDeleteFromPSUID(ctx context.Context, psuID uuid.UUID) error {
	return temporalStorageError(a.storage.AccountsDeleteFromPSUID(ctx, psuID))
}

var StorageAccountsDeleteFromPSUIDActivity = Activities{}.StorageAccountsDeleteFromPSUID

func StorageAccountsDeleteFromPSUID(ctx workflow.Context, psuID uuid.UUID) error {
	return executeActivity(ctx, StorageAccountsDeleteFromPSUIDActivity, nil, psuID)
}
