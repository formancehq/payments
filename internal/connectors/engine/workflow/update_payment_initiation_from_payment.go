package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type UpdatePaymentInitiationFromPayment struct {
	Payment *models.Payment
}

func (w Workflow) runUpdatePaymentInitiationFromPayment(
	ctx workflow.Context,
	updatePaymentInitiationFromPayment UpdatePaymentInitiationFromPayment,
) error {
	piIDs, err := activities.StoragePaymentInitiationIDsListFromPaymentID(
		infiniteRetryContext(ctx),
		updatePaymentInitiationFromPayment.Payment.ID,
	)
	if err != nil {
		return err
	}

	if len(piIDs) == 0 {
		// Nothing to do here
		return nil
	}

	for _, piID := range piIDs {
		adjustment := models.FromPaymentToPaymentInitiationAdjustment(
			updatePaymentInitiationFromPayment.Payment,
			piID,
		)

		if adjustment == nil {
			continue
		}

		if err := activities.StoragePaymentInitiationsAdjustmentsStore(
			infiniteRetryContext(ctx),
			*adjustment,
		); err != nil {
			return err
		}
	}

	return nil
}

var RunUpdatePaymentInitiationFromPayment any

func init() {
	RunUpdatePaymentInitiationFromPayment = Workflow{}.runUpdatePaymentInitiationFromPayment
}
