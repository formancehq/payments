package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgesStore(ctx context.Context, psuID uuid.UUID, from models.PSUBankBridge) error {
	return temporalStorageError(a.storage.PSUBankBridgesUpsert(ctx, psuID, from))
}

var StoragePSUBankBridgesStoreActivity = Activities{}.StoragePSUBankBridgesStore

func StoragePSUBankBridgesStore(ctx workflow.Context, psuID uuid.UUID, from models.PSUBankBridge) error {
	return executeActivity(ctx, StoragePSUBankBridgesStoreActivity, nil, psuID, from)
}
