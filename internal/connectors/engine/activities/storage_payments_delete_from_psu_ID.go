package activities

import (
	"context"

	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentsDeleteFromPSUID(ctx context.Context, psuID uuid.UUID) error {
	return temporalStorageError(a.storage.PaymentsDeleteFromPSUID(ctx, psuID))
}

var StoragePaymentsDeleteFromPSUIDActivity = Activities{}.StoragePaymentsDeleteFromPSUID

func StoragePaymentsDeleteFromPSUID(ctx workflow.Context, psuID uuid.UUID) error {
	return executeActivity(ctx, StoragePaymentsDeleteFromPSUIDActivity, nil, psuID)
}
