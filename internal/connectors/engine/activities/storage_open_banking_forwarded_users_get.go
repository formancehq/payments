package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingForwardedUsersGet(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.OpenBankingForwardedUser, error) {
	openBankingForwardedUser, err := a.storage.OpenBankingForwardedUserGet(ctx, psuID, connectorID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return openBankingForwardedUser, nil
}

var StorageOpenBankingForwardedUsersGetActivity = Activities{}.StorageOpenBankingForwardedUsersGet

func StorageOpenBankingForwardedUsersGet(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) (*models.OpenBankingForwardedUser, error) {
	var result models.OpenBankingForwardedUser
	err := executeActivity(ctx, StorageOpenBankingForwardedUsersGetActivity, &result, psuID, connectorID)
	return &result, err
}
