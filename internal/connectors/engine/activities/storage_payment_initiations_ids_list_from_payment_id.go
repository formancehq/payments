package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

// Deprecated: should not be used after version 3.0; we keep it in 3.1 for ongoing workflows.
func (a Activities) StoragePaymentInitiationIDsListFromPaymentID(ctx context.Context, paymentID models.PaymentID) ([]models.PaymentInitiationID, error) {
	cursor, err := a.storage.PaymentInitiationIDsListFromPaymentID(ctx, paymentID)
	if err != nil {
		return nil, temporalStorageError(err)
	}
	return cursor, nil
}

// Deprecated: should not be used after version 3.0; we keep it in 3.1 for ongoing workflows.
var StoragePaymentInitiationIDsListFromPaymentIDActivity = Activities{}.StoragePaymentInitiationIDsListFromPaymentID //lint:ignore SA1019 (ignore deprecation)

// Deprecated: should not be used after version 3.0; we keep it in 3.1 for ongoing workflows.
func StoragePaymentInitiationIDsListFromPaymentID(ctx workflow.Context, paymentID models.PaymentID) ([]models.PaymentInitiationID, error) {
	ret := []models.PaymentInitiationID{}
	//lint:ignore SA1019 (ignore deprecation)
	if err := executeActivity(ctx, StoragePaymentInitiationIDsListFromPaymentIDActivity, &ret, paymentID); err != nil {
		return nil, err
	}
	return ret, nil
}
