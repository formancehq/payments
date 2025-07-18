package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgesGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
	bridge, err := a.storage.PSUBankBridgesGet(ctx, psuID, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return bridge, nil
}

var StoragePSUBankBridgesGetActivity = Activities{}.StoragePSUBankBridgesGet

func StoragePSUBankBridgesGet(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
	var result models.PSUBankBridge
	err := executeActivity(ctx, StoragePSUBankBridgesGetActivity, &result, psuID, connectorID)
	return &result, err
}
