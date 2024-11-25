package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
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
	paymentID, err := w.pollTransfer(ctx, pollTransfer)
	if err != nil {
		return w.updateTasksError(
			ctx,
			pollTransfer.TaskID,
			pollTransfer.ConnectorID,
			err,
		)
	}
	return w.updateTaskSuccess(
		ctx,
		pollTransfer.TaskID,
		pollTransfer.ConnectorID,
		paymentID,
	)
}

func (w Workflow) pollTransfer(
	ctx workflow.Context,
	pollTransfer PollTransfer,
) (string, error) {
	pollTransferStatusResponse, err := activities.PluginPollTransferStatus(
		infiniteRetryContext(ctx),
		pollTransfer.ConnectorID,
		models.PollTransferStatusRequest{
			TransferID: pollTransfer.TransferID,
		},
	)
	if err != nil {
		return "", err
	}

	if pollTransferStatusResponse.Payment == nil {
		// payment not yet available, waiting for the next polling
		return "", nil
	}

	payment := models.FromPSPPaymentToPayment(*pollTransferStatusResponse.Payment, pollTransfer.ConnectorID)

	if err := w.storePIPaymentWithStatus(
		ctx,
		payment,
		pollTransfer.PaymentInitiationID,
		getPIStatusFromPayment(payment.Status),
		pollTransfer.ConnectorID,
	); err != nil {
		return "", err
	}

	// everything is done, delete the related schedule
	if err := activities.TemporalDeleteSchedule(ctx, pollTransfer.ScheduleID); err != nil {
		return "", err
	}

	return payment.ID.String(), activities.StorageSchedulesDelete(ctx, pollTransfer.ScheduleID)
}

const RunPollTransfer = "PollTransfer"
