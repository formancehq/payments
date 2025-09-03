package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingProviderPSUsGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.OpenBankingProviderPSU, error) {
	openBankingProviderPSU, err := a.storage.OpenBankingProviderPSUGet(ctx, psuID, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return openBankingProviderPSU, nil
}

var StorageOpenBankingProviderPSUsGetActivity = Activities{}.StorageOpenBankingProviderPSUsGet

func StorageOpenBankingProviderPSUsGet(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.OpenBankingProviderPSU, error) {
	var result models.OpenBankingProviderPSU
	err := executeActivity(ctx, StorageOpenBankingProviderPSUsGetActivity, &result, psuID, connectorID)
	return &result, err
}
