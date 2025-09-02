package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUOpenBankingConnectionAttemptsGet(ctx context.Context, attemptID uuid.UUID) (*models.PSUOpenBankingConnectionAttempt, error) {
	attempt, err := a.storage.PSUOpenBankingConnectionAttemptsGet(ctx, attemptID)
	if err != nil {
		return nil, temporalStorageError(err)
	}

	return attempt, nil
}

var StoragePSUOpenBankingConnectionAttemptsGetActivity = Activities{}.StoragePSUOpenBankingConnectionAttemptsGet

func StoragePSUOpenBankingConnectionAttemptsGet(ctx workflow.Context, attemptID uuid.UUID) (*models.PSUOpenBankingConnectionAttempt, error) {
	var ret *models.PSUOpenBankingConnectionAttempt
	err := executeActivity(ctx, StoragePSUOpenBankingConnectionAttemptsGetActivity, &ret, attemptID)
	if err != nil {
		return nil, err
	}

	return ret, nil
}
