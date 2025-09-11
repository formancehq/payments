package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingForwardedUsersDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return a.storage.OpenBankingForwardedUserDelete(ctx, psuID, connectorID)
}

var StorageOpenBankingForwardedUsersDeleteActivity = Activities{}.StorageOpenBankingForwardedUsersDelete

func StorageOpenBankingForwardedUsersDelete(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageOpenBankingForwardedUsersDeleteActivity, nil, psuID, connectorID)
}
