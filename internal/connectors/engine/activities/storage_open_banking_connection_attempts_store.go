package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePSUOpenBankingConnectionAttemptsStore(ctx context.Context, from models.OpenBankingConnectionAttempt) error {
	return temporalStorageError(a.storage.OpenBankingConnectionAttemptsUpsert(ctx, from))
}

var StoragePSUOpenBankingConnectionAttemptsStoreActivity = Activities{}.StoragePSUOpenBankingConnectionAttemptsStore

func StoragePSUOpenBankingConnectionAttemptsStore(ctx workflow.Context, from models.OpenBankingConnectionAttempt) error {
	return executeActivity(ctx, StoragePSUOpenBankingConnectionAttemptsStoreActivity, nil, from)
}
