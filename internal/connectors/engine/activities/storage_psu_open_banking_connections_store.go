package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUOpenBankingConnectionsStore(ctx context.Context, psuID uuid.UUID, from models.PSUOpenBankingConnection) error {
	return temporalStorageError(a.storage.PSUOpenBankingConnectionsUpsert(ctx, psuID, from))
}

var StoragePSUOpenBankingConnectionsStoreActivity = Activities{}.StoragePSUOpenBankingConnectionsStore

func StoragePSUOpenBankingConnectionsStore(ctx workflow.Context, psuID uuid.UUID, from models.PSUOpenBankingConnection) error {
	return executeActivity(ctx, StoragePSUOpenBankingConnectionsStoreActivity, nil, psuID, from)
}
