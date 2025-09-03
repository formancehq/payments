package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageOpenBankingProviderPSUsDelete(ctx context.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return a.storage.OpenBankingProviderPSUDelete(ctx, psuID, connectorID)
}

var StorageOpenBankingProviderPSUsDeleteActivity = Activities{}.StorageOpenBankingProviderPSUsDelete

func StorageOpenBankingProviderPSUsDelete(ctx workflow.Context, psuID uuid.UUID, connectorID models.ConnectorID) error {
	return executeActivity(ctx, StorageOpenBankingProviderPSUsDeleteActivity, nil, psuID, connectorID)
}
