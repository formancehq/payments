package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgesDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return a.storage.PSUBankBridgesDelete(ctx, psuID, connectorID)
}

var StoragePSUBankBridgesDeleteActivity = Activities{}.StoragePSUBankBridgesDelete

func StoragePSUBankBridgesDelete(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StoragePSUBankBridgesDeleteActivity, nil, psuID, connectorID)
}
