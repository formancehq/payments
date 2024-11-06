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
	pollPayoutStatusResponse, err := activities.PluginPollPayoutStatus(
		infiniteRetryContext(ctx),
		pollPayout.ConnectorID,
		models.PollPayoutStatusRequest{
			PayoutID: pollPayout.PayoutID,
		},
	)
	if err != nil {
		return err
	}

	if pollPayoutStatusResponse.Payment == nil {
		// payment not yet available, waiting for the next polling
		return nil
	}

	payment := models.FromPSPPaymentToPayment(*pollPayoutStatusResponse.Payment, pollPayout.ConnectorID)

	if err := w.storePIPayment(ctx, payment, pollPayout.PaymentInitiationID, pollPayout.ConnectorID); err != nil {
		return err
	}

	if err := w.updateTaskSuccess(
		ctx,
		pollPayout.TaskID,
		pollPayout.ConnectorID,
		payment.ID.String(),
	); err != nil {
		return err
	}

	// everything is done, delete the related schedule
	if err := activities.TemporalDeleteSchedule(ctx, pollPayout.ScheduleID); err != nil {
		return err
	}

	return activities.StorageSchedulesDelete(ctx, pollPayout.ScheduleID)
}

const RunPollPayout = "PollPayout"
