package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationIDsListFromPaymentID(ctx context.Context, paymentID models.PaymentID) ([]models.PaymentInitiationID, error) {
	return a.storage.PaymentInitiationIDsListFromPaymentID(ctx, paymentID)
}

var StoragePaymentInitiationIDsListFromPaymentIDActivity = Activities{}.StoragePaymentInitiationIDsListFromPaymentID

func StoragePaymentInitiationIDsListFromPaymentID(ctx workflow.Context, paymentID models.PaymentID) ([]models.PaymentInitiationID, error) {
	ret := []models.PaymentInitiationID{}
	if err := executeActivity(ctx, StoragePaymentInitiationIDsListFromPaymentIDActivity, &ret, paymentID); err != nil {
		return nil, err
	}
	return ret, nil
}
