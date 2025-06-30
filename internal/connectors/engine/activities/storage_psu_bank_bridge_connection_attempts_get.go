package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUBankBridgeConnectionAttemptsGet(ctx context.Context, attemptID uuid.UUID) (*models.PSUBankBridgeConnectionAttempt, error) {
	attempt, err := a.storage.PSUBankBridgeConnectionAttemptsGet(ctx, attemptID)
	if err != nil {
		return nil, temporalStorageError(err)
	}

	return attempt, nil
}

var StoragePSUBankBridgeConnectionAttemptsGetActivity = Activities{}.StoragePSUBankBridgeConnectionAttemptsGet

func StoragePSUBankBridgeConnectionAttemptsGet(ctx workflow.Context, attemptID uuid.UUID) (*models.PSUBankBridgeConnectionAttempt, error) {
	var ret *models.PSUBankBridgeConnectionAttempt
	err := executeActivity(ctx, StoragePSUBankBridgeConnectionAttemptsGetActivity, &ret, attemptID)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
