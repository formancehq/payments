package activities

import (
	"context"
	"errors"

	"github.com/formancehq/payments/internal/models"
	"github.com/formancehq/payments/internal/storage"
	"github.com/google/uuid"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type StoragePSUBankBridgeConnectionsGetFromConnectionIDResult struct {
	Connection *models.PSUBankBridgeConnection
	PSUID      uuid.UUID
}

func (a Activities) StoragePSUBankBridgeConnectionsGetFromConnectionID(ctx context.Context, connectorID models.ConnectorID, connectionID string) (*StoragePSUBankBridgeConnectionsGetFromConnectionIDResult, error) {
	connection, psuID, err := a.storage.PSUBankBridgeConnectionsGetFromConnectionID(ctx, connectorID, connectionID)
	if err != nil {
		switch {
		case errors.Is(err, storage.ErrDuplicateKeyValue),
			errors.Is(err, storage.ErrValidation),
			errors.Is(err, storage.ErrForeignKeyViolation):
			// Do not retry these errors
			return nil, temporal.NewNonRetryableApplicationError(err.Error(), ErrTypeStorage, err)
		default:
			return nil, temporal.NewApplicationErrorWithCause(err.Error(), ErrTypeStorage, err)
		}
	}
	return &StoragePSUBankBridgeConnectionsGetFromConnectionIDResult{
		Connection: connection,
		PSUID:      psuID,
	}, nil
}

var StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity = Activities{}.StoragePSUBankBridgeConnectionsGetFromConnectionID

func StoragePSUBankBridgeConnectionsGetFromConnectionID(ctx workflow.Context, connectorID models.ConnectorID, connectionID string) (*models.PSUBankBridgeConnection, uuid.UUID, error) {
	var result StoragePSUBankBridgeConnectionsGetFromConnectionIDResult
	err := executeActivity(ctx, StoragePSUBankBridgeConnectionsGetFromConnectionIDActivity, &result, connectorID, connectionID)
	return result.Connection, result.PSUID, err
}
