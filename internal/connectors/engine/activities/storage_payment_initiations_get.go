package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationsGet(ctx context.Context, id models.PaymentInitiationID) (*models.PaymentInitiation, error) {
	return a.storage.PaymentInitiationsGet(ctx, id)
}

var StoragePaymentInitiationsGetActivity = Activities{}.StoragePaymentInitiationsGet

func StoragePaymentInitiationsGet(ctx workflow.Context, id models.PaymentInitiationID) (*models.PaymentInitiation, error) {
	var result models.PaymentInitiation
	err := executeActivity(ctx, StoragePaymentInitiationsGetActivity, &result, id)
	return &result, err
}
