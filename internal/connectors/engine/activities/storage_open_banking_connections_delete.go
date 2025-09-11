package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingConnectionsDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return a.storage.OpenBankingConnectionsDelete(ctx, psuID, connectorID, connectionID)
}

var StorageOpenBankingConnectionsDeleteActivity = Activities{}.StorageOpenBankingConnectionsDelete

func StorageOpenBankingConnectionsDelete(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string) error {
	return executeActivity(ctx, StorageOpenBankingConnectionsDeleteActivity, nil, psuID, connectorID, connectionID)
}
