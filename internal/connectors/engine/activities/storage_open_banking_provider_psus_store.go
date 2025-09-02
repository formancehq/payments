package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingProviderPSUsStore(ctx context.Context, psuID uuid.UUID, from models.OpenBankingProviderPSU) error {
	return temporalStorageError(a.storage.OpenBankingProviderPSUUpsert(ctx, psuID, from))
}

var StorageOpenBankingProviderPSUsStoreActivity = Activities{}.StorageOpenBankingProviderPSUsStore

func StorageOpenBankingProviderPSUsStore(ctx workflow.Context, psuID uuid.UUID, from models.OpenBankingProviderPSU) error {
	return executeActivity(ctx, StorageOpenBankingProviderPSUsStoreActivity, nil, psuID, from)
}
