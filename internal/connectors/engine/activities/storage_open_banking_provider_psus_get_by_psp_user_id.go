package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingProviderPSUsGetByPSPUserID(ctx context.Context, pspUserID string, connectorID models.ConnectorID) (*models.OpenBankingProviderPSU, error) {
	obProviderPSU, err := a.storage.OpenBankingProviderPSUGetByPSPUserID(ctx, pspUserID, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return obProviderPSU, nil
}

var StorageOpenBankingProviderPSUsGetByPSPUserIDActivity = Activities{}.StorageOpenBankingProviderPSUsGetByPSPUserID

func StorageOpenBankingProviderPSUsGetByPSPUserID(ctx workflow.Context, pspUserID string, connectorID models.ConnectorID) (*models.OpenBankingProviderPSU, error) {
	var result models.OpenBankingProviderPSU
	err := executeActivity(ctx, StorageOpenBankingProviderPSUsGetByPSPUserIDActivity, &result, pspUserID, connectorID)
	return &result, err
}
