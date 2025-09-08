package activities

import (
	"context"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDeleteFromConnectionID(ctx context.Context, psuID uuid.UUID, connectionID string) error {
	return temporalStorageError(a.storage.PaymentsDeleteFromOpenBankingConnectionID(ctx, psuID, connectionID))
}

var StoragePaymentsDeleteFromConnectionIDActivity = Activities{}.StoragePaymentsDeleteFromConnectionID

func StoragePaymentsDeleteFromConnectionID(ctx workflow.Context, psuID uuid.UUID, connectionID string) error {
	return executeActivity(ctx, StoragePaymentsDeleteFromConnectionIDActivity, nil, psuID, connectionID)
}
