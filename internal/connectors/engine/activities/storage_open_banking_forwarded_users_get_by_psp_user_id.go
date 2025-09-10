package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingForwardedUsersGetByPSPUserID(ctx context.Context, pspUserID string, connectorID models.ConnectorID) (*models.OpenBankingForwardedUser, error) {
	obProviderPSU, err := a.storage.OpenBankingForwardedUserGetByPSPUserID(ctx, pspUserID, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return obProviderPSU, nil
}

var StorageOpenBankingForwardedUsersGetByPSPUserIDActivity = Activities{}.StorageOpenBankingForwardedUsersGetByPSPUserID

func StorageOpenBankingForwardedUsersGetByPSPUserID(ctx workflow.Context, pspUserID string, connectorID models.ConnectorID) (*models.OpenBankingForwardedUser, error) {
	var result models.OpenBankingForwardedUser
	err := executeActivity(ctx, StorageOpenBankingForwardedUsersGetByPSPUserIDActivity, &result, pspUserID, connectorID)
	return &result, err
}
