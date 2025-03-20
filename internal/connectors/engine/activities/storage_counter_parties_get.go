package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageCounterPartiesGet(ctx context.Context, id uuid.UUID) (*models.CounterParty, error) {
	ba, err := a.storage.CounterPartiesGet(ctx, id)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return ba, nil
}

var StorageCounterPartiesGetActivity = Activities{}.StorageCounterPartiesGet

func StorageCounterPartiesGet(ctx workflow.Context, id uuid.UUID) (*models.CounterParty, error) {
	var result models.CounterParty
	err := executeActivity(ctx, StorageCounterPartiesGetActivity, &result, id)
	return &result, err
}
