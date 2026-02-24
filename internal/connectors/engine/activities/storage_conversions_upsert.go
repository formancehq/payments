package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageConversionsUpsert(ctx context.Context, conversions []models.Conversion) error {
	return temporalStorageError(a.storage.ConversionsUpsert(ctx, conversions))
}

var StorageConversionsUpsertActivity = Activities{}.StorageConversionsUpsert

func StorageConversionsUpsert(ctx workflow.Context, conversions []models.Conversion) error {
	return executeActivity(ctx, StorageConversionsUpsertActivity, nil, conversions)
}
