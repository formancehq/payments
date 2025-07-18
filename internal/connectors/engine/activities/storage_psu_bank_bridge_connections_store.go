package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgeConnectionsStore(ctx context.Context, psuID uuid.UUID, from models.PSUBankBridgeConnection) error {
	return temporalStorageError(a.storage.PSUBankBridgeConnectionsUpsert(ctx, psuID, from))
}

var StoragePSUBankBridgeConnectionsStoreActivity = Activities{}.StoragePSUBankBridgeConnectionsStore

func StoragePSUBankBridgeConnectionsStore(ctx workflow.Context, psuID uuid.UUID, from models.PSUBankBridgeConnection) error {
	return executeActivity(ctx, StoragePSUBankBridgeConnectionsStoreActivity, nil, psuID, from)
}
