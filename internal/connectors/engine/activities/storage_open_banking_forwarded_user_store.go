package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingForwardedUsersStore(ctx context.Context, psuID uuid.UUID, from models.OpenBankingForwardedUser) error {
	return temporalStorageError(a.storage.OpenBankingForwardedUserUpsert(ctx, psuID, from))
}

var StorageOpenBankingForwardedUsersStoreActivity = Activities{}.StorageOpenBankingForwardedUsersStore

func StorageOpenBankingForwardedUsersStore(ctx workflow.Context, psuID uuid.UUID, from models.OpenBankingForwardedUser) error {
	return executeActivity(ctx, StorageOpenBankingForwardedUsersStoreActivity, nil, psuID, from)
}
