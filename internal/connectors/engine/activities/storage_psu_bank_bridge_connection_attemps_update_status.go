package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgeConnectionAttemptsUpdateStatus(ctx context.Context, id uuid.UUID, status models.PSUBankBridgeConnectionAttemptStatus, error *string) error {
	return temporalStorageError(a.storage.PSUBankBridgeConnectionAttemptsUpdateStatus(ctx, id, status, error))
}

var StoragePSUBankBridgeConnectionAttemptsUpdateStatusActivity = Activities{}.StoragePSUBankBridgeConnectionAttemptsUpdateStatus

func StoragePSUBankBridgeConnectionAttemptsUpdateStatus(ctx workflow.Context, id uuid.UUID, status models.PSUBankBridgeConnectionAttemptStatus, error *string) error {
	return executeActivity(ctx, StoragePSUBankBridgeConnectionAttemptsUpdateStatusActivity, nil, id, status, error)
}
