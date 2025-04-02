package workflow

import (
	"fmt"

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
			&pollTransfer.ConnectorID,
			err,
		)
	}

	if paymentID != "" {
		return w.updateTaskSuccess(
			ctx,
			pollTransfer.TaskID,
			&pollTransfer.ConnectorID,
			paymentID,
		)
	}

	return nil
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

	paymentID := ""
	var piErr error
	switch {
	case pollTransferStatusResponse.Payment == nil && pollTransferStatusResponse.Error == nil:
		// payment not yet available and no error, waiting for the next polling
		return "", nil

	case pollTransferStatusResponse.Payment != nil:
		payment := models.FromPSPPaymentToPayment(*pollTransferStatusResponse.Payment, pollTransfer.ConnectorID)

		if err := w.storePIPaymentWithStatus(
			ctx,
			payment,
			pollTransfer.PaymentInitiationID,
			getPIStatusFromPayment(payment.Status),
		); err != nil {
			return "", err
		}

		paymentID = payment.ID.String()

	case pollTransferStatusResponse.Error != nil:
		// Means that the payment initiation failed, and we need to register
		// the error in the task as well as stopping the schedule polling.
		piErr = fmt.Errorf("%s", *pollTransferStatusResponse.Error)
	}

	// everything is done, delete the related schedule
	if err := activities.TemporalScheduleDelete(
		infiniteRetryContext(ctx),
		pollTransfer.ScheduleID,
	); err != nil {
		return "", err
	}

	if err := activities.StorageSchedulesDelete(
		infiniteRetryContext(ctx),
		pollTransfer.ScheduleID,
	); err != nil {
		return "", err
	}

	return paymentID, piErr
}

const RunPollTransfer = "PollTransfer"
