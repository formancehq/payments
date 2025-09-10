package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

type StoragePSUOpenBankingConnectionsGetFromConnectionIDResult struct {
	Connection *models.OpenBankingConnection
	PSUID      uuid.UUID
}

func (a Activities) StoragePSUOpenBankingConnectionsGetFromConnectionID(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*StoragePSUOpenBankingConnectionsGetFromConnectionIDResult, error) {
	connection, psuID, err := a.storage.OpenBankingConnectionsGetFromConnectionID(ctx, connectorID, connectionID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return &StoragePSUOpenBankingConnectionsGetFromConnectionIDResult{
		Connection: connection,
		PSUID:      psuID,
	}, nil
}

var StoragePSUOpenBankingConnectionsGetFromConnectionIDActivity = Activities{}.StoragePSUOpenBankingConnectionsGetFromConnectionID

func StoragePSUOpenBankingConnectionsGetFromConnectionID(ctx workflow.Context, connectorID models.ConnectorID, connectionID string) (*models.OpenBankingConnection, uuid.UUID, error) {
	var result StoragePSUOpenBankingConnectionsGetFromConnectionIDResult
	err := executeActivity(ctx, StoragePSUOpenBankingConnectionsGetFromConnectionIDActivity, &result, connectorID, connectionID)
	return result.Connection, result.PSUID, err
}
