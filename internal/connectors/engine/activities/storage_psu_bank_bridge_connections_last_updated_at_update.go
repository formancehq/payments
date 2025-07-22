package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgeConnectionsLastUpdatedAtUpdate(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, updatedAt time.Time) error {
	return temporalStorageError(a.storage.PSUBankBridgeConnectionsUpdateLastDataUpdate(ctx, psuID, connectorID, connectionID, updatedAt))
}

var StoragePSUBankBridgeConnectionsLastUpdatedAtUpdateActivity = Activities{}.StoragePSUBankBridgeConnectionsLastUpdatedAtUpdate

func StoragePSUBankBridgeConnectionsLastUpdatedAtUpdate(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, updatedAt time.Time) error {
	return executeActivity(ctx, StoragePSUBankBridgeConnectionsLastUpdatedAtUpdateActivity, nil, psuID, connectorID, connectionID, updatedAt)
}
