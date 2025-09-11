package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type StorageOpenBankingConnectionsGetFromConnectionIDResult struct {
	Connection *models.OpenBankingConnection
	PSUID      uuid.UUID
}

func (a Activities) StorageOpenBankingConnectionsGetFromConnectionID(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*StorageOpenBankingConnectionsGetFromConnectionIDResult, error) {
	connection, psuID, err := a.storage.OpenBankingConnectionsGetFromConnectionID(ctx, connectorID, connectionID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return &StorageOpenBankingConnectionsGetFromConnectionIDResult{
		Connection: connection,
		PSUID:      psuID,
	}, nil
}

var StorageOpenBankingConnectionsGetFromConnectionIDActivity = Activities{}.StorageOpenBankingConnectionsGetFromConnectionID

func StorageOpenBankingConnectionsGetFromConnectionID(ctx workflow.Context, connectorID models.ConnectorID, connectionID string) (*models.OpenBankingConnection, uuid.UUID, error) {
	var result StorageOpenBankingConnectionsGetFromConnectionIDResult
	err := executeActivity(ctx, StorageOpenBankingConnectionsGetFromConnectionIDActivity, &result, connectorID, connectionID)
	return result.Connection, result.PSUID, err
}
