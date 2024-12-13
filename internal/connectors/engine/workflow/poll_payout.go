package workflow

import (
	"github.com/formancehq/payments/internal/connectors/engine/activities"
	"github.com/formancehq/payments/internal/models"
	"go.temporal.io/sdk/workflow"
)

type PollPayout struct {
	TaskID              models.TaskID
	ConnectorID         models.ConnectorID
	PaymentInitiationID models.PaymentInitiationID
	PayoutID            string
	ScheduleID          string
}

func (w Workflow) runPollPayout(
	ctx workflow.Context,
	pollPayout PollPayout,
) error {
	paymentID, err := w.pollPayout(ctx, pollPayout)
	if err != nil {
		return w.updateTasksError(
			ctx,
			pollPayout.TaskID,
			&pollPayout.ConnectorID,
			err,
		)
	}
	return w.updateTaskSuccess(
		ctx,
		pollPayout.TaskID,
		&pollPayout.ConnectorID,
		paymentID,
	)
}

func (w Workflow) pollPayout(
	ctx workflow.Context,
	pollPayout PollPayout,
) (string, error) {
	pollPayoutStatusResponse, err := activities.PluginPollPayoutStatus(
		infiniteRetryContext(ctx),
		pollPayout.ConnectorID,
		models.PollPayoutStatusRequest{
			PayoutID: pollPayout.PayoutID,
		},
	)
	if err != nil {
		return "", err
	}

	if pollPayoutStatusResponse.Payment == nil {
		// payment not yet available, waiting for the next polling
		return "", nil
	}

	payment := models.FromPSPPaymentToPayment(*pollPayoutStatusResponse.Payment, pollPayout.ConnectorID)

	if err := w.storePIPaymentWithStatus(
		ctx,
		payment,
		pollPayout.PaymentInitiationID,
		getPIStatusFromPayment(payment.Status),
		pollPayout.ConnectorID,
	); err != nil {
		return "", err
	}

	// everything is done, delete the related schedule
	if err := activities.TemporalDeleteSchedule(ctx, pollPayout.ScheduleID); err != nil {
		return "", err
	}

	return payment.ID.String(), activities.StorageSchedulesDelete(ctx, pollPayout.ScheduleID)
}

const RunPollPayout = "PollPayout"
