package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/workflow"
)

type PollTransfer struct {
	TaskID              models.TaskID
	ConnectorID         models.ConnectorID
	PaymentInitiationID models.PaymentInitiationID
	TransferID          string
	ScheduleID          string
}

func (w Workflow) runPollTransfer(
	ctx workflow.Context,
	pollTransfer PollTransfer,
) error {
	pollTransferStatusResponse, err := activities.PluginPollTransferStatus(
		infiniteRetryContext(ctx),
		pollTransfer.ConnectorID,
		models.PollTransferStatusRequest{
			TransferID: pollTransfer.TransferID,
		},
	)
	if err != nil {
		return err
	}

	if pollTransferStatusResponse.Payment == nil {
		// payment not yet available, waiting for the next polling
		return nil
	}

	payment := models.FromPSPPaymentToPayment(*pollTransferStatusResponse.Payment, pollTransfer.ConnectorID)

	if err := w.storePIPayment(ctx, payment, pollTransfer.PaymentInitiationID, pollTransfer.ConnectorID); err != nil {
		return err
	}

	if err := w.updateTaskSuccess(
		ctx,
		pollTransfer.TaskID,
		pollTransfer.ConnectorID,
		payment.ID.String(),
	); err != nil {
		return err
	}

	// everything is done, delete the related schedule
	if err := activities.TemporalDeleteSchedule(ctx, pollTransfer.ScheduleID); err != nil {
		return err
	}

	return activities.StorageSchedulesDelete(ctx, pollTransfer.ScheduleID)
}

const RunPollTransfer = "PollTransfer"

func (w Workflow) storePIPayment(
	ctx workflow.Context,
	payment models.Payment,
	paymentInitiationID models.PaymentInitiationID,
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
			Status:              models.PAYMENT_INITIATION_ADJUSTMENT_STATUS_PROCESSED,
		},
		nil,
		nil,
	)
	if err != nil {
		return err
	}

	return nil
}
