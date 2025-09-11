package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingConnectionAttemptsStore(ctx context.Context, from models.OpenBankingConnectionAttempt) error {
	return temporalStorageError(a.storage.OpenBankingConnectionAttemptsUpsert(ctx, from))
}

var StorageOpenBankingConnectionAttemptsStoreActivity = Activities{}.StorageOpenBankingConnectionAttemptsStore

func StorageOpenBankingConnectionAttemptsStore(ctx workflow.Context, from models.OpenBankingConnectionAttempt) error {
	return executeActivity(ctx, StorageOpenBankingConnectionAttemptsStoreActivity, nil, from)
}
