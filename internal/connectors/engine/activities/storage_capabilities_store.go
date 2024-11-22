package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageCapabilitiesStore(ctx context.Context, connectorID models.ConnectorID, capabilities []models.Capability) error {
	return temporalStorageError(a.storage.CapabilitiesUpsert(ctx, connectorID, capabilities))
}

var StorageCapabilitiesStoreActivity = Activities{}.StorageCapabilitiesStore

func StorageCapabilitiesStore(ctx workflow.Context, connectorID models.ConnectorID, capabilities []models.Capability) error {
	return executeActivity(ctx, StorageCapabilitiesStoreActivity, nil, connectorID, capabilities)
}
