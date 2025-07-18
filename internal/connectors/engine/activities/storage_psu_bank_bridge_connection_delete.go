package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgeConnectionDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return a.storage.PSUBankBridgeConnectionsDelete(ctx, psuID, connectorID, connectionID)
}

var StoragePSUBankBridgeConnectionDeleteActivity = Activities{}.StoragePSUBankBridgeConnectionDelete

func StoragePSUBankBridgeConnectionDelete(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return executeActivity(ctx, StoragePSUBankBridgeConnectionDeleteActivity, nil, psuID, connectorID, connectionID)
}
