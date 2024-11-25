package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationReversalsGet(ctx context.Context, id models.PaymentInitiationReversalID) (*models.PaymentInitiationReversal, error) {
	pi, err := a.storage.PaymentInitiationReversalsGet(ctx, id)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return pi, nil
}

var StoragePaymentInitiationReversalsGetActivity = Activities{}.StoragePaymentInitiationReversalsGet

func StoragePaymentInitiationReversalsGet(ctx workflow.Context, id models.PaymentInitiationReversalID) (*models.PaymentInitiationReversal, error) {
	var result models.PaymentInitiationReversal
	err := executeActivity(ctx, StoragePaymentInitiationReversalsGetActivity, &result, id)
	return &result, err
}
