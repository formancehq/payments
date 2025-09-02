package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUOpenBankingConnectionsDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return a.storage.PSUOpenBankingConnectionsDelete(ctx, psuID, connectorID, connectionID)
}

var StoragePSUOpenBankingConnectionsDeleteActivity = Activities{}.StoragePSUOpenBankingConnectionsDelete

func StoragePSUOpenBankingConnectionsDelete(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return executeActivity(ctx, StoragePSUOpenBankingConnectionsDeleteActivity, nil, psuID, connectorID, connectionID)
}
