package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StorageCounterPartiesAddRelatedAccount(ctx context.Context, counterPartyID uuid.UUID, relatedAccount models.CounterPartiesRelatedAccount) error {
	return temporalStorageError(a.storage.CounterPartiesAddRelatedAccount(ctx, counterPartyID, relatedAccount))
}

var StorageCounterPartiesAddRelatedAccountActivity = Activities{}.StorageCounterPartiesAddRelatedAccount

func StorageCounterPartiesAddRelatedAccount(ctx workflow.Context, counterPartyID uuid.UUID, relatedAccount models.CounterPartiesRelatedAccount) error {
	return executeActivity(ctx, StorageCounterPartiesAddRelatedAccountActivity, nil, counterPartyID, relatedAccount)
}
