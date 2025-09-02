package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUOpenBankingConnectionAttemptsUpdateStatus(ctx context.Context, id uuid.UUID, status models.PSUOpenBankingConnectionAttemptStatus, error *string) error {
	return temporalStorageError(a.storage.PSUOpenBankingConnectionAttemptsUpdateStatus(ctx, id, status, error))
}

var StoragePSUOpenBankingConnectionAttemptsUpdateStatusActivity = Activities{}.StoragePSUOpenBankingConnectionAttemptsUpdateStatus

func StoragePSUOpenBankingConnectionAttemptsUpdateStatus(ctx workflow.Context, id uuid.UUID, status models.PSUOpenBankingConnectionAttemptStatus, error *string) error {
	return executeActivity(ctx, StoragePSUOpenBankingConnectionAttemptsUpdateStatusActivity, nil, id, status, error)
}
