package activities

import (
	"context"
	"time"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUOpenBankingConnectionsLastUpdatedAtUpdate(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, updatedAt time.Time) error {
	return temporalStorageError(a.storage.OpenBankingConnectionsUpdateLastDataUpdate(ctx, psuID, connectorID, connectionID, updatedAt))
}

var StoragePSUOpenBankingConnectionsLastUpdatedAtUpdateActivity = Activities{}.StoragePSUOpenBankingConnectionsLastUpdatedAtUpdate

func StoragePSUOpenBankingConnectionsLastUpdatedAtUpdate(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID, connectionID string, updatedAt time.Time) error {
	return executeActivity(ctx, StoragePSUOpenBankingConnectionsLastUpdatedAtUpdateActivity, nil, psuID, connectorID, connectionID, updatedAt)
}
