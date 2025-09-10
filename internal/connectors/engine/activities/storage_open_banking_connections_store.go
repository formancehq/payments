package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingConnectionsStore(ctx context.Context, psuID uuid.UUID, from models.OpenBankingConnection) error {
	return temporalStorageError(a.storage.OpenBankingConnectionsUpsert(ctx, psuID, from))
}

var StorageOpenBankingConnectionsStoreActivity = Activities{}.StorageOpenBankingConnectionsStore

func StorageOpenBankingConnectionsStore(ctx workflow.Context, psuID uuid.UUID, from models.OpenBankingConnection) error {
	return executeActivity(ctx, StorageOpenBankingConnectionsStoreActivity, nil, psuID, from)
}
