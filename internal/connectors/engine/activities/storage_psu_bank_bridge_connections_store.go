package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgeConnectionsStore(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, from models.PSUBankBridgeConnection) error {
	return temporalStorageError(a.storage.PSUBankBridgeConnectionsUpsert(ctx, psuID, connectorID, from))
}

var StoragePSUBankBridgeConnectionsStoreActivity = Activities{}.StoragePSUBankBridgeConnectionsStore

func StoragePSUBankBridgeConnectionsStore(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID, from models.PSUBankBridgeConnection) error {
	return executeActivity(ctx, StoragePSUBankBridgeConnectionsStoreActivity, nil, psuID, connectorID, from)
}
