package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgesGetByPSPUserID(ctx context.Context, pspUserID string, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
	bridge, err := a.storage.PSUBankBridgesGetByPSPUserID(ctx, pspUserID, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return bridge, nil
}

var StoragePSUBankBridgesGetByPSPUserIDActivity = Activities{}.StoragePSUBankBridgesGetByPSPUserID

func StoragePSUBankBridgesGetByPSPUserID(ctx workflow.Context, pspUserID string, connectorID models.ConnectorID) (*models.PSUBankBridge, error) {
	var result models.PSUBankBridge
	err := executeActivity(ctx, StoragePSUBankBridgesGetByPSPUserIDActivity, &result, pspUserID, connectorID)
	return &result, err
}
