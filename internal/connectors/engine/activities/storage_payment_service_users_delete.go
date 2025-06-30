package activities

import (
	"context"

	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentServiceUsersDelete(ctx context.Context, paymentServiceUserID string) error {
	return a.storage.PaymentServiceUsersDelete(ctx, paymentServiceUserID)
}

var StoragePaymentServiceUsersDeleteActivity = Activities{}.StoragePaymentServiceUsersDelete

func StoragePaymentServiceUsersDelete(ctx workflow.Context, paymentServiceUserID string) error {
	return executeActivity(ctx, StoragePaymentServiceUsersDeleteActivity, nil, paymentServiceUserID)
}
