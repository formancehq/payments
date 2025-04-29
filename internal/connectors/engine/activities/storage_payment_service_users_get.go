package activities

import (
	context "context"

	"github.com/formancehq/payments/internal/models"
	"github.com/google/uuid"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentServiceUsersGet(ctx context.Context, id uuid.UUID) (*models.PaymentServiceUser, error) {
	psu, err := a.storage.PaymentServiceUsersGet(ctx, id)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return psu, nil
}

var StoragePaymentServiceUsersGetActivity = Activities{}.StoragePaymentServiceUsersGet

func StoragePaymentServiceUsersGet(ctx workflow.Context, id uuid.UUID) (*models.PaymentServiceUser, error) {
	var result models.PaymentServiceUser
	err := executeActivity(ctx, StoragePaymentServiceUsersGetActivity, &result, id)
	return &result, err
}
