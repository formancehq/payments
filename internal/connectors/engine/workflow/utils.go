package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

func (w Workflow) storePIPaymentWithStatus(
	ctx workflow.Context,
	payment models.Payment,
	paymentInitiationID models.PaymentInitiationID,
	status models.PaymentInitiationAdjustmentStatus,
	connectorID models.ConnectorID,
) error {
	// payment is available, storing it
	err := activities.StoragePaymentsStore(
		infiniteRetryContext(ctx),
		[]models.Payment{payment},
	)
	if err != nil {
		return err
	}

	if err := workflow.ExecuteChildWorkflow(
		workflow.WithChildOptions(
			ctx,
			workflow.ChildWorkflowOptions{
				TaskQueue:         connectorID.String(),
				ParentClosePolicy: enums.PARENT_CLOSE_POLICY_ABANDON,
				SearchAttributes: map[string]interface{}{
					SearchAttributeStack: w.stack,
				},
			},
		),
		RunSendEvents,
		SendEvents{
			Payment: &payment,
		},
	).Get(ctx, nil); err != nil {
		return err
	}

	err = activities.StoragePaymentInitiationsRelatedPaymentsStore(
		infiniteRetryContext(ctx),
		paymentInitiationID,
		payment.ID,
		payment.CreatedAt,
	)
	if err != nil {
		return err
	}

	err = w.addPIAdjustment(
		ctx,
		models.PaymentInitiationAdjustmentID{
			PaymentInitiationID: paymentInitiationID,
			CreatedAt:           workflow.Now(ctx),
			Status:              status,
		},
		payment.Amount,
		&payment.Asset,
		nil,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}

func getPIStatusFromPayment(status models.PaymentStatus) models.PaymentInitiationAdjustmentStatus {
	switch status {
	case models.PAYMENT_STATUS_SUCCEEDED,
		models.PAYMENT_STATUS_CAPTURE,
		models.PAYMENT_STATUS_REFUND_REVERSED:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED

	case models.PAYMENT_STATUS_CANCELLED,
		models.PAYMENT_STATUS_CAPTURE_FAILED,
		models.PAYMENT_STATUS_FAILED,
		models.PAYMENT_STATUS_EXPIRED:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_FAILED

	case models.PAYMENT_STATUS_PENDING,
		models.PAYMENT_STATUS_AUTHORISATION:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSING

	case models.PAYMENT_STATUS_REFUNDED:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSED
	case models.PAYMENT_STATUS_REFUNDED_FAILURE:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_REVERSE_FAILED

	default:
		return models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_UNKNOWN
	}
}
