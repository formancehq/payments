package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingConnectionAttemptsUpdateStatus(ctx context.Context, id uuid.UUID, status models.OpenBankingConnectionAttemptStatus, error *string) error {
	return temporalStorageError(a.storage.OpenBankingConnectionAttemptsUpdateStatus(ctx, id, status, error))
}

var StorageOpenBankingConnectionAttemptsUpdateStatusActivity = Activities{}.StorageOpenBankingConnectionAttemptsUpdateStatus

func StorageOpenBankingConnectionAttemptsUpdateStatus(ctx workflow.Context, id uuid.UUID, status models.OpenBankingConnectionAttemptStatus, error *string) error {
	return executeActivity(ctx, StorageOpenBankingConnectionAttemptsUpdateStatusActivity, nil, id, status, error)
}
