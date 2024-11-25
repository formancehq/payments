package activities

import (
	"context"

	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

func (a Activities) StoragePaymentInitiationsAdjusmentsIfPredicateStore(ctx context.Context, adj models.PaymentInitiationAdjustment, unAcceptablePreviousStatus []models.PaymentInitiationAdjustmentStatus) (bool, error) {
	inserted, err := a.storage.PaymentInitiationAdjustmentsUpsertIfPredicate(ctx, adj, func(pia models.PaymentInitiationAdjustment) bool {
		for _, status := range unAcceptablePreviousStatus {
			if pia.Status == status {
				return false
			}
		}
		return true
	})
	return inserted, temporalStorageError(err)
}

var StoragePaymentInitiationsAdjusmentsIfStatusEqualStoreActivity = Activities{}.StoragePaymentInitiationsAdjusmentsIfPredicateStore

func StoragePaymentInitiationsAdjusmentsIfPredicateStore(ctx workflow.Context, adj models.PaymentInitiationAdjustment, unAcceptablePreviousStatus []models.PaymentInitiationAdjustmentStatus) (bool, error) {
	var result bool
	err := executeActivity(ctx, StoragePaymentInitiationsAdjustmentsStoreActivity, &result, adj, unAcceptablePreviousStatus)
	return result, err
}
